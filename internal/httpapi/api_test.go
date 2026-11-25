package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"testing"
)

type modelDispatch func(*http.Request, config.Relay) (*http.Response, error)

func (f modelDispatch) Dispatch(r *http.Request, v config.Relay) (*http.Response, error) {
	return f(r, v)
}

func TestModelAPITransportPrivacyAndSnapshot(t *testing.T) {
	c := apiConfig(t)
	h, err := NewAPI(c, modelDispatch(func(r *http.Request, _ config.Relay) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer "+upstreamKey || r.URL.Host != "api.provider.example" {
			t.Fatal("snapshot berubah di luar constructor")
		}
		return nil, errors.New(upstreamKey + " private request content " + gatewayToken)
	}))
	if err != nil {
		t.Fatal(err)
	}
	*c = *apiConfig(t, "https://api.provider.example/v1", "https://replacement.example/v2")
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"coding-default","private":"private request content"}`))
	r.Header.Set("Authorization", "Bearer "+gatewayToken)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 502 {
		t.Fatal("error transport tidak dikenali")
	}
	for _, secret := range []string{upstreamKey, gatewayToken, "private request content", "replacement.example"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("error/input sensitif bocor")
		}
	}
}

func TestModelAPIAuthBeforeRead(t *testing.T) {
	h, err := NewAPI(apiConfig(t), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		t.Fatal("auth invalid melakukan dispatch")
		return nil, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/models", "/providers/missing/v1/models", "/v1/chat/completions", "/providers/missing/v1/chat/completions"} {
		for _, token := range []string{"", "wrong", adminToken, upstreamKey, "duplicate"} {
			method := "POST"
			if strings.HasSuffix(path, "/models") {
				method = "GET"
			}
			r := httptest.NewRequest(method, path, nil)
			body := &unreadBody{}
			r.Body = body
			if token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			if token == "duplicate" {
				r.Header["Authorization"] = []string{"Bearer " + gatewayToken, "Bearer " + gatewayToken}
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 401 || body.reads != 0 || body.closes != 0 {
				t.Fatal("guard tidak mendahului body/lookup")
			}
		}
	}
}

func TestModelAPIRejectsBeforeDispatch(t *testing.T) {
	calls := 0
	h, err := NewAPI(apiConfig(t), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, body, content, encoding, header, code string
		status                                      int
	}{
		{"/v1/chat/completions", `{"model":"missing"}`, "application/json", "", "", "model_not_found", 404},
		{"/providers/missing/v1/chat/completions", `{"model":"coding-default"}`, "application/json", "", "", "upstream_not_found", 404},
		{"/v1/chat/completions", `{"model":"coding-default","stream":true,"str\u0065am":false}`, "application/json", "", "", "invalid_json", 400},
		{"/v1/chat/completions", `{"model":"coding-default","model":"other"}`, "application/json", "", "", "invalid_json", 400},
		{"/v1/chat/completions", `{"model":"coding-default"}`, "text/plain", "", "", "unsupported_media_type", 415},
		{"/v1/chat/completions", `{"model":"coding-default"}`, "", "", "", "unsupported_media_type", 415},
		{"/v1/chat/completions", `{"model":"coding-default"}`, "application/json", "gzip", "", "unsupported_content_encoding", 415},
		{"/v1/chat/completions", `{"model":"coding-default"}`, "application/json", "", "X-Relay-Target", "forbidden_target_override", 400},
		{"/v1/chat/completions", `{"model":"coding-default"}`, "application/json", "", "X-Relay-Path", "forbidden_target_override", 400},
		{"/v1/chat/completions", `{"model":"coding-default","private":"` + gatewayToken + `"} {}`, "application/json", "", "", "invalid_json", 400},
		{"/v1/chat/completions", `{"model":"coding-default","x":"` + strings.Repeat("x", 4096) + `"}`, "application/json", "", "", "request_too_large", 413},
	} {
		r := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
		r.ContentLength = -1
		r.Header.Set("Authorization", "Bearer "+gatewayToken)
		r.Header.Set("Content-Type", tc.content)
		r.Header.Set("X-Request-ID", gatewayToken)
		if tc.encoding != "" {
			r.Header.Set("Content-Encoding", tc.encoding)
		}
		if tc.header != "" {
			r.Header.Set(tc.header, "https://ignored.invalid")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		var envelope struct {
			Error     struct{ Code string }
			RequestID string `json:"request_id"`
		}
		if json.Unmarshal(w.Body.Bytes(), &envelope) != nil || w.Code != tc.status || envelope.Error.Code != tc.code || envelope.RequestID != w.Header().Get("X-Request-ID") || envelope.RequestID == gatewayToken || w.Header().Get("X-Infinity-Error-Source") != "gateway" {
			t.Fatal("penolakan/error envelope salah")
		}
		for _, secret := range []string{gatewayToken, adminToken, upstreamKey} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatal("input/secret bocor dalam error")
			}
		}
	}
	if calls != 0 {
		t.Fatal("request invalid melakukan dispatch")
	}
}

func TestModelAPIBodyLimitAndAdmission(t *testing.T) {
	calls := 0
	dispatch := modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	h, err := NewAPI(apiConfig(t), dispatch)
	if err != nil {
		t.Fatal(err)
	}
	prefix := `{"model":"coding-default","x":"`
	suffix := `"}`
	for _, size := range []int{4093, 4094, 4096, 4097} {
		body := prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix
		r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
		r.ContentLength = 0
		r.Header.Set("Authorization", "Bearer "+gatewayToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 413
		if size == 4093 {
			want = 200
		}
		if w.Code != want {
			t.Fatal("batas aktual/remap tidak ditegakkan")
		}
	}
	if calls != 1 {
		t.Fatal("body melewati batas melakukan dispatch")
	}
	bounded, err := NewAPI(apiConfig(t, "body_buffer_bytes: 67108864", "body_buffer_bytes: 4096"), dispatch)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	body := &unreadBody{}
	r.Body = body
	r.Header.Set("Authorization", "Bearer "+gatewayToken)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	bounded.ServeHTTP(w, r)
	if w.Code != 429 || body.reads != 0 {
		t.Fatal("admission tidak mendahului alokasi/read")
	}
	if h, err := NewAPI(apiConfig(t), modelDispatch(nil)); err == nil || h != nil {
		t.Fatal("typed nil dispatcher diterima")
	}
	if h, err := NewAPI(nil, dispatch); err == nil || h != nil {
		t.Fatal("snapshot nil diterima")
	}
}
func apiConfig(t *testing.T, edits ...string) *config.Config {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.NewReplacer("max_body_bytes: 10485760", "max_body_bytes: 4096").Replace(string(data))
	text = strings.NewReplacer(edits...).Replace(text)
	c, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": gatewayToken, "INFINITY_ADMIN_TOKEN": adminToken, "INFINITY_UPSTREAM_KEY": upstreamKey}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestModelAPIEndpoints(t *testing.T) {
	calls := 0
	h, err := NewAPI(apiConfig(t), modelDispatch(func(r *http.Request, _ config.Relay) (*http.Response, error) {
		calls++
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"model":"provider-model-id"`) {
			t.Fatal("model alias belum dipetakan")
		}
		return &http.Response{StatusCode: 201, Header: http.Header{"Content-Type": {"application/json; charset=utf-8"}}, Body: io.NopCloser(strings.NewReader(`{"extra":true}`))}, nil
	}))
	if err != nil || h == nil {
		t.Fatal("API valid ditolak")
	}
	for _, path := range []string{"/v1/models", "/providers/primary/v1/models", "/v1/chat/completions", "/providers/primary/v1/chat/completions"} {
		method := "POST"
		want := 201
		if strings.HasSuffix(path, "/models") {
			method = "GET"
			want = 200
		}
		r := httptest.NewRequest(method, path, strings.NewReader(`{"model":"coding-default"}`))
		r.Header.Set("Content-Type", "application/json; charset=utf-8")
		r.Header.Set("Authorization", "Bearer "+gatewayToken)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want || w.Header().Get("X-Request-ID") == "" {
			t.Fatal("status/request ID endpoint salah")
		}
		if method == "POST" && (w.Body.String() != `{"extra":true}` || w.Header().Get("Content-Type") != "application/json; charset=utf-8") {
			t.Fatal("respons upstream berubah")
		}
	}
	if calls != 2 {
		t.Fatal("katalog melakukan dispatch")
	}
}

func TestModelAPIRequiredContentType(t *testing.T) {
	h, err := NewAPI(apiConfig(t), modelDispatch(func(r *http.Request, _ config.Relay) (*http.Response, error) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("header JSON wajib hilang karena nominasi hop-by-hop")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"coding-default"}`))
	r.Header.Set("Authorization", "Bearer "+gatewayToken)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Connection", "Content-Type")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal("generasi JSON valid ditolak")
	}
}

func TestModelAPIHeadCatalogue(t *testing.T) {
	h, err := NewAPI(apiConfig(t), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		t.Fatal("HEAD katalog melakukan dispatch")
		return nil, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/models", "/providers/primary/v1/models"} {
		r := httptest.NewRequest("HEAD", path, nil)
		r.Header.Set("Authorization", "Bearer "+gatewayToken)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || w.Header().Get("Content-Type") != "application/json" || w.Body.Len() != 0 || w.Header().Get("X-Infinity-Error-Source") != "" {
			t.Fatal("HEAD katalog salah status/body/source")
		}
	}
	for _, tc := range []struct {
		path, token string
		status      int
	}{{"/providers/missing/v1/models", gatewayToken, 404}, {"/v1/models", "", 401}, {"/providers/primary/v1/models", adminToken, 401}} {
		r := httptest.NewRequest("HEAD", tc.path, nil)
		body := &unreadBody{}
		r.Body = body
		if tc.token != "" {
			r.Header.Set("Authorization", "Bearer "+tc.token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status || body.reads != 0 {
			t.Fatal("HEAD provider/auth tidak ditolak sebelum body/dispatch")
		}
	}
}
