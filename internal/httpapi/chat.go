package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"opencode-infinity/internal/gateway"
	"strconv"
	"strings"
)

type modelHandler struct{ coordinator *gateway.Coordinator }

func (h modelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := newRequestID()
	w.Header().Set("X-Request-ID", id)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	var response *http.Response
	var err error
	catalog := r.Method == http.MethodGet || r.Method == http.MethodHead
	if catalog {
		response, err = h.coordinator.Models(r.PathValue("provider"))
	} else {
		value, ok := jsonContentType(r.Header)
		if !ok {
			writeModelError(w, id, &gateway.Failure{Code: "unsupported_media_type", Status: 415})
			return
		}
		media, _, parseErr := mime.ParseMediaType(value)
		if parseErr != nil || media != "application/json" {
			writeModelError(w, id, &gateway.Failure{Code: "unsupported_media_type", Status: 415})
			return
		}
		response, err = h.coordinator.Chat(r, r.PathValue("provider"))
	}
	if err != nil {
		var f *gateway.Failure
		if !errors.As(err, &f) {
			f = &gateway.Failure{Code: "upstream_transport_error", Status: 502}
		}
		writeModelError(w, id, f)
		return
	}
	defer response.Body.Close()
	w.Header().Set("X-Infinity-Error-Source", "upstream")
	if catalog {
		w.Header().Del("X-Infinity-Error-Source")
	}
	// Tidak menyalin cookie, authorization, Location, atau header operasional upstream.
	w.Header()["Content-Type"] = nil
	if v := response.Header.Get("Content-Type"); v != "" {
		w.Header().Set("Content-Type", v)
	}
	// Body opaque yang belum didekompresi tetap memerlukan metadata content coding.
	for _, value := range response.Header.Values("Content-Encoding") {
		w.Header().Add("Content-Encoding", value)
	}
	if v := response.Header.Get("Retry-After"); validRetryAfter(v) {
		w.Header().Set("Retry-After", v)
	}
	w.WriteHeader(response.StatusCode)
	if r.Method == http.MethodHead {
		return
	}
	buffer := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(writerOnly{w}, readerOnly{response.Body}, buffer); err != nil {
		// Hentikan framing respons parsial; jangan mengirim EOF sukses atau envelope pengganti.
		panic(http.ErrAbortHandler)
	}
}

type writerOnly struct{ io.Writer }
type readerOnly struct{ io.Reader }

func jsonContentType(header http.Header) (string, bool) {
	value := ""
	found := false
	for name, values := range header {
		if strings.EqualFold(name, "Content-Type") {
			if found || len(values) != 1 || len(values[0]) > 8192 {
				return "", false
			}
			found = true
			value = values[0]
		}
	}
	return value, found
}
func validRetryAfter(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	if _, err := http.ParseTime(value); err == nil {
		return true
	}
	for _, b := range []byte(value) {
		if b < '0' || b > '9' {
			return false
		}
	}
	_, err := strconv.ParseUint(value, 10, 63)
	return err == nil
}
