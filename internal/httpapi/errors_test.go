package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/config"
	"strings"
	"testing"
)

func assertGatewayError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) string {
	t.Helper()
	var payload struct {
		Error     struct{ Message, Type, Code string }
		RequestID string `json:"request_id"`
	}
	if w.Code != status || json.Unmarshal(w.Body.Bytes(), &payload) != nil || payload.Error.Code != code || payload.Error.Message == "" || payload.Error.Type == "" {
		t.Fatalf("error gateway salah: status=%d, code=%q; ingin %d %q", w.Code, payload.Error.Code, status, code)
	}
	id := payload.RequestID
	if len(id) != 36 || !strings.HasPrefix(id, "req_") {
		t.Fatal("request ID server tidak valid")
	}
	if _, err := hex.DecodeString(id[4:]); err != nil || w.Header().Get("X-Request-ID") != id || w.Header().Get("X-Infinity-Error-Source") != "gateway" {
		t.Fatal("request ID/source gateway tidak konsisten")
	}
	if w.Header().Get("Content-Type") != "application/json" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Location") != "" {
		t.Fatal("metadata error gateway salah")
	}
	for _, private := range []string{gatewayToken, adminToken, upstreamKey, "private-marker"} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatal("error gateway memantulkan input/secret")
		}
	}
	return id
}

func errorAPI(t *testing.T) http.Handler {
	t.Helper()
	h, err := NewAPI(apiConfig(t), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		t.Fatal("error route melakukan dispatch")
		return nil, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestGatewayErrorNotFound(t *testing.T) {
	h := errorAPI(t)
	ids := make(map[string]bool)
	for _, target := range []string{"/", "/private-marker?token=private-marker", "/v1/unknown", "/providers/primary/v1/unknown", "/v1/models/"} {
		for _, token := range []string{gatewayToken, "", adminToken, upstreamKey} {
			r := httptest.NewRequest("POST", target, nil)
			body := &unreadBody{}
			r.Body = body
			r.Header.Set("Authorization", "Bearer "+token)
			r.Header.Set("X-Request-ID", "private-marker")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			status, code := 404, "endpoint_not_found"
			if token != gatewayToken {
				status, code = 401, "invalid_gateway_token"
			}
			id := assertGatewayError(t, w, status, code)
			if ids[id] || body.reads != 0 || body.closes != 0 {
				t.Fatal("ID dipakai ulang atau fallback membaca/menutup body")
			}
			ids[id] = true
		}
	}
}

func TestGatewayErrorMethod(t *testing.T) {
	h := errorAPI(t)
	for _, route := range []struct{ path, allow string }{
		{"/v1/models", "GET, HEAD"}, {"/providers/primary/v1/models", "GET, HEAD"},
		{"/v1/chat/completions", "POST"}, {"/providers/primary/v1/chat/completions", "POST"},
	} {
		for _, method := range []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS", "CONNECT"} {
			if strings.Contains(route.allow, method) {
				continue
			}
			for _, token := range []string{gatewayToken, "", adminToken} {
				r := httptest.NewRequest(method, route.path, nil)
				r.Header.Set("Authorization", "Bearer "+token)
				body := &unreadBody{}
				r.Body = body
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				status, code, allow := 405, "method_not_allowed", route.allow
				if token != gatewayToken {
					status, code, allow = 401, "invalid_gateway_token", ""
				}
				assertGatewayError(t, w, status, code)
				if w.Header().Get("Allow") != allow || body.reads != 0 {
					t.Fatal("Allow salah atau penolakan method membaca body")
				}
			}
		}
	}
}

func TestGatewayErrorPath(t *testing.T) {
	h := errorAPI(t)
	for _, target := range []string{"/v1//models", "/v1/./models", "/v1/private-marker/../models", "//v1/models", "/providers/primary/../primary/v1/models", "//", "/."} {
		for _, token := range []string{gatewayToken, "", adminToken} {
			r := httptest.NewRequest("GET", target+"?secret=private-marker", nil)
			r.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			status, code := 400, "invalid_request"
			if token != gatewayToken {
				status, code = 401, "invalid_gateway_token"
			}
			assertGatewayError(t, w, status, code)
		}
	}
	// Escape tetap mengikuti matching per segmen ServeMux, bukan normalisasi URL.Path.
	for _, tc := range []struct {
		path, code string
		status     int
	}{
		{"/v1/%6dodels", "", 200}, {"/providers/%70rimary/v1/models", "", 200},
		{"/providers/%2e/v1/models", "upstream_not_found", 404},
		{"/providers/primary%2fother/v1/models", "upstream_not_found", 404},
		{"/v1/models/", "endpoint_not_found", 404}, {"/v1/models%2f", "endpoint_not_found", 404},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Authorization", "Bearer "+gatewayToken)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if tc.code != "" {
			assertGatewayError(t, w, tc.status, tc.code)
		} else if w.Code != tc.status || w.Header().Get("X-Infinity-Error-Source") != "" {
			t.Fatal("path escaped valid tidak mempertahankan katalog")
		}
	}
}

func TestGatewayErrorUnsupportedAPI(t *testing.T) {
	h := errorAPI(t)
	for _, target := range []string{"/v1/responses", "/providers/primary/v1/responses", "/providers/missing/v1/responses"} {
		for _, method := range []string{"GET", "HEAD", "POST", "OPTIONS"} {
			for _, token := range []string{gatewayToken, "", adminToken} {
				r := httptest.NewRequest(method, target, nil)
				r.Header.Set("Authorization", "Bearer "+token)
				body := &unreadBody{}
				r.Body = body
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				status, code := 501, "unsupported_api"
				if token != gatewayToken {
					status, code = 401, "invalid_gateway_token"
				}
				assertGatewayError(t, w, status, code)
				if body.reads != 0 {
					t.Fatal("endpoint unsupported membaca body")
				}
			}
		}
	}
}

func TestGatewayErrorRetryAfter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
		want   string
	}{
		{"absent", nil, ""}, {"zero", []string{"0"}, "0"}, {"delta", []string{"120"}, "120"},
		{"date", []string{"Wed, 21 Oct 2015 07:28:00 GMT"}, "Wed, 21 Oct 2015 07:28:00 GMT"},
		{"negative", []string{"-1"}, ""}, {"fraction", []string{"1.5"}, ""},
		{"overflow", []string{"99999999999999999999999999"}, ""},
		{"text", []string{"private-marker"}, ""}, {"list", []string{"1, 2"}, ""},
		{"duplicate", []string{"1", "2"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, err := NewAPI(apiConfig(t), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
				return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": tc.values}, Body: io.NopCloser(strings.NewReader("opaque upstream error"))}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"coding-default"}`))
			r.Header.Set("Authorization", "Bearer "+gatewayToken)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 429 || w.Body.String() != "opaque upstream error" || w.Header().Get("X-Infinity-Error-Source") != "upstream" || w.Header().Get("Retry-After") != tc.want {
				t.Fatal("Retry-After invalid diteruskan atau error upstream berubah")
			}
		})
	}
}

type brokenRequestBody struct{ closes int }

func (*brokenRequestBody) Read(p []byte) (int, error) {
	return copy(p, "{"), errors.New("private-marker " + gatewayToken)
}
func (b *brokenRequestBody) Close() error { b.closes++; return nil }

func TestGatewayErrorBodyRead(t *testing.T) {
	calls := 0
	h, err := NewAPI(apiConfig(t, "max_active_requests: 20", "max_active_requests: 1"), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	body := &brokenRequestBody{}
	r.Body = body
	r.ContentLength = -1
	r.Header.Set("Authorization", "Bearer "+gatewayToken)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertGatewayError(t, w, 400, "invalid_request")
	if calls != 0 || body.closes != 1 {
		t.Fatal("body error melakukan dispatch atau tidak ditutup sekali")
	}
	r = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"coding-default"}`))
	r.Header.Set("Authorization", "Bearer "+gatewayToken)
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || calls != 1 {
		t.Fatal("body error tidak melepas kapasitas")
	}
}
