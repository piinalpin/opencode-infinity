package integration_test

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func errorRequest(t *testing.T, server *httptest.Server, method, target string, body io.Reader) *http.Request {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), method, server.URL+target, body)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+modelGatewayToken)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Request-ID", "private-marker")
	return r
}

func readErrorResponse(t *testing.T, response *http.Response) []byte {
	t.Helper()
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertWireGatewayError(t *testing.T, response *http.Response, status int, code string) string {
	t.Helper()
	data := readErrorResponse(t, response)
	id := response.Header.Get("X-Request-ID")
	if response.StatusCode != status || len(id) != 36 || !strings.HasPrefix(id, "req_") || response.Header.Get("X-Infinity-Error-Source") != "gateway" {
		t.Fatalf("status/ID/source gateway salah: %d, ingin %d", response.StatusCode, status)
	}
	if _, err := hex.DecodeString(id[4:]); err != nil {
		t.Fatal("ID bukan hex server")
	}
	if response.Header.Get("Content-Type") != "application/json" || response.Header.Get("Location") != "" || response.Header.Get("Content-Encoding") != "" {
		t.Fatal("envelope gateway membawa metadata upstream/redirect")
	}
	if response.Request.Method == http.MethodHead {
		if len(data) != 0 {
			t.Fatal("HEAD error mengirim body pada wire")
		}
		return id
	}
	var envelope struct {
		Error struct{ Code, Type, Message string }
		ID    string `json:"request_id"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Error.Code != code || envelope.Error.Type == "" || envelope.Error.Message == "" || envelope.ID != id {
		t.Fatalf("envelope error gateway salah, ingin %s", code)
	}
	for _, secret := range []string{modelGatewayToken, modelAdminToken, modelPrimaryKey, modelSecondaryKey, "private-marker", "upstream.invalid"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("envelope gateway memantulkan secret/input")
		}
	}
	return id
}

func TestGatewayRequestErrors(t *testing.T) {
	h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{}`) }, func(http.ResponseWriter, *http.Request) { t.Error("penolakan mengganti upstream") })
	server := httptest.NewServer(h.handler)
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	valid := `{"model":"coding-default"}`
	ids := make(map[string]bool)
	for _, tc := range []struct {
		name, method, path, body, code, header string
		coding                                 []string
		status                                 int
		chunked, unauthorized                  bool
	}{
		{name: "empty", body: "", status: 400, code: "invalid_json"},
		{name: "truncated", body: `{"model":`, status: 400, code: "invalid_json"},
		{name: "trailing", body: valid + `{}`, status: 400, code: "invalid_json"},
		{name: "array", body: `[]`, status: 400, code: "invalid_json"},
		{name: "utf8", body: "{\"model\":\"coding-default\",\"x\":\"\xff\"}", status: 400, code: "invalid_json"},
		{name: "escaped-control", body: `{"model":"coding-default","m\u006fdel":"other"}`, status: 400, code: "invalid_json"},
		{name: "gzip", body: "\x1f\x8bprivate-marker", coding: []string{"gzip"}, status: 415, code: "unsupported_content_encoding"},
		{name: "br", body: valid, coding: []string{"br"}, status: 415, code: "unsupported_content_encoding"},
		{name: "deflate", body: valid, coding: []string{"deflate"}, status: 415, code: "unsupported_content_encoding"},
		{name: "stacked", body: valid, coding: []string{"identity, gzip"}, status: 415, code: "unsupported_content_encoding"},
		{name: "duplicate", body: valid, coding: []string{"identity", "identity"}, status: 415, code: "unsupported_content_encoding"},
		{name: "oversize-known", body: valid + strings.Repeat(" ", 4097), status: 413, code: "request_too_large"},
		{name: "oversize-chunked", body: valid + strings.Repeat(" ", 4097), chunked: true, status: 413, code: "request_too_large"},
		{name: "remap-oversize", body: valid + strings.Repeat(" ", 4096-len(valid)), status: 413, code: "request_too_large"},
		{name: "target", body: valid, header: "X-Relay-Target", status: 400, code: "forbidden_target_override"},
		{name: "prefix", body: valid, header: "X-Relay-Path", status: 400, code: "forbidden_target_override"},
		{name: "missing", method: "GET", path: "/private-marker?key=private-marker", status: 404, code: "endpoint_not_found"},
		{name: "method", method: "DELETE", path: "/providers/primary/v1/models", status: 405, code: "method_not_allowed"},
		{name: "canonical", method: "GET", path: "/v1//models?key=private-marker", status: 400, code: "invalid_request"},
		{name: "responses", path: "/v1/responses", body: valid, status: 501, code: "unsupported_api"},
		{name: "provider-responses", path: "/providers/primary/v1/responses", body: valid, status: 501, code: "unsupported_api"},
		{name: "head-missing", method: "HEAD", path: "/unknown", status: 404, code: "endpoint_not_found"},
		{name: "head-method", method: "HEAD", status: 405, code: "method_not_allowed"},
		{name: "head-responses", method: "HEAD", path: "/v1/responses", status: 501, code: "unsupported_api"},
		{name: "auth-first", method: "GET", path: "/v1//models", unauthorized: true, status: 401, code: "invalid_gateway_token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method, path := tc.method, tc.path
			if method == "" {
				method = "POST"
			}
			if path == "" {
				path = "/v1/chat/completions"
			}
			var body io.Reader = strings.NewReader(tc.body)
			if tc.chunked {
				body = io.NopCloser(body)
			}
			r := errorRequest(t, server, method, path, body)
			if tc.coding != nil {
				r.Header["Content-Encoding"] = tc.coding
			}
			if tc.header != "" {
				r.Header.Set(tc.header, "https://private-marker.invalid/other")
			}
			if tc.unauthorized {
				r.Header.Set("Authorization", "Bearer "+modelAdminToken)
			}
			response, err := client.Do(r)
			if err != nil {
				t.Fatal(err)
			}
			id := assertWireGatewayError(t, response, tc.status, tc.code)
			if ids[id] {
				t.Fatal("request ID dipakai ulang")
			}
			ids[id] = true
			if tc.status == 405 && response.Header.Get("Allow") == "" {
				t.Fatal("405 tanpa Allow")
			}
		})
	}
	if h.a.Snapshot().Accepted != 0 || h.b.Snapshot().Accepted != 0 || h.primary.Snapshot().Total != 0 || h.secondary.Snapshot().Total != 0 {
		t.Fatal("request ditolak membuka CONNECT/dispatch")
	}
	// 4093 byte menjadi tepat 4096 byte sesudah mapping; unknown length juga diterima.
	for _, chunked := range []bool{false, true} {
		body := valid + strings.Repeat(" ", 4093-len(valid))
		var reader io.Reader = strings.NewReader(body)
		if chunked {
			reader = io.NopCloser(reader)
		}
		r := errorRequest(t, server, "POST", "/v1/chat/completions", reader)
		if chunked {
			r.Header.Set("Content-Encoding", "identity")
		}
		response, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		if data := readErrorResponse(t, response); response.StatusCode != 200 || string(data) != `{}` {
			t.Fatal("request tepat batas gagal setelah penolakan")
		}
	}
	if h.primary.Snapshot().Total != 2 || h.secondary.Snapshot().Total != 0 {
		t.Fatal("jumlah dispatch batas valid salah")
	}
	h.verifyPath(t)
}

func TestGatewayErrorUpstream(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 422, 429, 500, 502, 503, 599} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			payload := []byte("opaque upstream error\x00\xff")
			coding := ""
			retryAfter := "120"
			if status == 429 {
				retryAfter = ""
			}
			if status == 503 {
				retryAfter = "Wed, 21 Oct 2015 07:28:00 GMT"
				var encoded bytes.Buffer
				z := gzip.NewWriter(&encoded)
				_, _ = z.Write(payload)
				if err := z.Close(); err != nil {
					t.Fatal(err)
				}
				payload, coding = encoded.Bytes(), "gzip"
			}
			upstream := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/octet-stream")
				if coding != "" {
					w.Header().Set("Content-Encoding", coding)
				}
				if status == 429 {
					w.Header().Add("Retry-After", "1")
					w.Header().Add("Retry-After", "2")
				} else {
					w.Header().Set("Retry-After", retryAfter)
				}
				w.Header().Set("X-Request-ID", "private-marker")
				w.Header().Set("X-Infinity-Error-Source", "gateway")
				w.Header().Set("Location", "https://private-marker.invalid")
				w.Header().Set("Set-Cookie", "private-marker")
				w.Header().Set("Authorization", "private-marker")
				w.WriteHeader(status)
				_, _ = w.Write(payload)
			}
			h := newModelHarness(t, upstream, upstream)
			for _, client := range h.clients {
				client.Transport.(*http.Transport).DisableCompression = true
			}
			server := httptest.NewServer(h.handler)
			defer server.Close()
			client := server.Client()
			client.Transport.(*http.Transport).DisableCompression = true
			for _, path := range []string{"/v1/chat/completions", "/providers/secondary/v1/chat/completions"} {
				r := errorRequest(t, server, "POST", path, strings.NewReader(`{"model":"coding-default","stream":true}`))
				response, err := client.Do(r)
				if err != nil {
					t.Fatal(err)
				}
				if data := readErrorResponse(t, response); response.StatusCode != status || !bytes.Equal(data, payload) || response.Header.Get("Content-Type") != "application/octet-stream" || response.Header.Get("Content-Encoding") != coding || response.Header.Get("Retry-After") != retryAfter || response.Header.Get("X-Infinity-Error-Source") != "upstream" {
					t.Fatal("status/body/metadata upstream tidak dipertahankan")
				}
				id := response.Header.Get("X-Request-ID")
				if len(id) != 36 || !strings.HasPrefix(id, "req_") || response.Header.Get("Location") != "" || response.Header.Get("Set-Cookie") != "" || response.Header.Get("Authorization") != "" {
					t.Fatal("header operasional upstream bocor")
				}
			}
			if h.primary.Snapshot().Total != 1 || h.secondary.Snapshot().Total != 1 {
				t.Fatal("error upstream memicu retry atau penggantian origin")
			}
			h.verifyPath(t)
		})
	}
}

func TestGatewayErrorRedirect(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			target := "https://private-marker.invalid/other"
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", target)
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "private-marker")
			}, func(http.ResponseWriter, *http.Request) { t.Error("redirect mencapai origin lain") }, "max_active_requests: 20", "max_active_requests: 1")
			for _, c := range h.clients {
				c.Transport.(*http.Transport).DisableCompression = true
			}
			target = "https://" + h.secondary.Authority + "/v2/chat/completions"
			server := httptest.NewServer(h.handler)
			defer server.Close()
			for _, path := range []string{"/v1/chat/completions", "/providers/primary/v1/chat/completions"} {
				r := errorRequest(t, server, "POST", path, strings.NewReader(`{"model":"coding-default","stream":true}`))
				response, err := server.Client().Do(r)
				if err != nil {
					t.Fatal(err)
				}
				assertWireGatewayError(t, response, 502, "upstream_redirect_rejected")
			}
			if h.primary.Snapshot().Total != 2 || h.secondary.Snapshot().Total != 0 || h.b.Snapshot().Accepted != 0 {
				t.Fatal("redirect diulang, slot bocor, atau origin lain dihubungi")
			}
			h.verifyPath(t)
		})
	}
}
