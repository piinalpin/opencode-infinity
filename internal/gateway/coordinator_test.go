package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

type dispatchFunc func(*http.Request, config.Relay) (*http.Response, error)

func (f dispatchFunc) Dispatch(r *http.Request, relay config.Relay) (*http.Response, error) {
	return f(r, relay)
}

type countedBody struct {
	io.Reader
	closes atomic.Int32
}

type modelTimeoutError struct{}

func (modelTimeoutError) Error() string   { return "synthetic-private-timeout" }
func (modelTimeoutError) Timeout() bool   { return true }
func (modelTimeoutError) Temporary() bool { return true }
func TestModelCoordinatorTimeoutCategory(t *testing.T) {
	c, err := New(modelConfig(t), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) { return nil, modelTimeoutError{} }))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Chat(chatRequest(`{"model":"coding-default"}`), "")
	assertFailure(t, err, "upstream_timeout")
}

func (b *countedBody) Close() error { b.closes.Add(1); return nil }
func assertFailure(t *testing.T, err error, code string) {
	t.Helper()
	var f *Failure
	if !errors.As(err, &f) || f.Code != code {
		t.Fatal("kategori error salah")
	}
}

func TestModelCoordinatorLifecycle(t *testing.T) {
	body := &countedBody{Reader: strings.NewReader(`{"ok":true}`)}
	c, err := New(modelConfig(t, "max_active_requests: 20", "max_active_requests: 1"), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	response, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(response.Body); err != nil {
		t.Fatal("read respons gagal")
	}
	extra, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
	if extra != nil {
		extra.Body.Close()
		t.Fatal("slot dilepas sebelum forwarding/caller selesai")
	}
	assertFailure(t, err, "gateway_capacity_exceeded")
	response.Body.Close()
	response.Body.Close()
	if body.closes.Load() != 1 {
		t.Fatal("body ditutup lebih dari sekali")
	}
	next, err := c.Chat(chatRequest(`{"model":"coding-default"}`), "")
	if err != nil {
		t.Fatal("slot tidak dilepas")
	}
	next.Body.Close()
}

func TestModelCoordinatorRejectsAndReleases(t *testing.T) {
	calls := 0
	dispatcher := dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
		calls++
		return nil, errors.New("synthetic-private-network-error")
	})
	c, err := New(modelConfig(t, "max_active_requests: 20", "max_active_requests: 1"), dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ body, id, code string }{
		{`{"model":"missing"}`, "", "model_not_found"}, {`{"model":"coding-default"}`, "missing", "upstream_not_found"},
		{`{"model":"coding-default","stream":true}`, "", "unsupported_api"}, {`{`, "", "invalid_json"},
		{`{"model":"coding-default","x":"` + strings.Repeat("x", 4096) + `"}`, "", "request_too_large"},
		{`{"model":"coding-default"}`, "", "upstream_transport_error"},
	} {
		result, err := c.Chat(chatRequest(tc.body), tc.id)
		if result != nil {
			result.Body.Close()
			t.Fatal("request invalid menghasilkan respons")
		}
		assertFailure(t, err, tc.code)
	}
	if calls != 1 {
		t.Fatal("request ditolak melakukan dispatch")
	}
	for _, d := range []Dispatcher{nil, dispatchFunc(nil)} {
		if c, err := New(modelConfig(t), d); err == nil || c != nil {
			t.Fatal("dispatcher nil diterima")
		}
	}
	if c, err := New(nil, dispatcher); err == nil || c != nil {
		t.Fatal("snapshot nil diterima")
	}
	// Context dipertahankan sampai adapter; error mentah tidak menjadi error komponen.
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	c, err = New(modelConfig(t), dispatchFunc(func(r *http.Request, _ config.Relay) (*http.Response, error) {
		if r.Context().Err() == nil {
			t.Fatal("context cancellation hilang")
		}
		return nil, r.Context().Err()
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Chat(chatRequest(`{"model":"coding-default"}`).WithContext(cancelled), "")
	assertFailure(t, err, "upstream_transport_error")
}

func modelConfig(t *testing.T, edits ...string) *config.Config {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.NewReplacer("max_body_bytes: 10485760", "max_body_bytes: 4096").Replace(string(data))
	text = strings.NewReplacer(edits...).Replace(text)
	c, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": "synthetic-gateway-credential-12345", "INFINITY_ADMIN_TOKEN": "synthetic-admin-credential-67890", "INFINITY_UPSTREAM_KEY": "synthetic-provider-key"}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestModelCatalog(t *testing.T) {
	c, err := New(modelConfig(t), dispatchFunc(func(*http.Request, config.Relay) (*http.Response, error) {
		t.Fatal("katalog melakukan dispatch")
		return nil, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "primary"} {
		r, err := c.Models(id)
		if err != nil || r == nil {
			t.Fatal("katalog valid ditolak")
		}
		b, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil || string(b) != `{"object":"list","data":[{"id":"coding-default","object":"model","owned_by":"primary"}]}` {
			t.Fatal("katalog tidak berisi alias scope")
		}
	}
	if r, err := c.Models("missing"); err == nil || r != nil {
		t.Fatal("provider unknown memakai default")
	}
}
func chatRequest(body string) *http.Request {
	r := httptest.NewRequest("POST", "http://ignored.invalid/client?target=ignored", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}
func TestModelCoordinatorDispatch(t *testing.T) {
	calls := 0
	c, err := New(modelConfig(t), dispatchFunc(func(r *http.Request, relay config.Relay) (*http.Response, error) {
		calls++
		b, err := io.ReadAll(r.Body)
		if err != nil || string(b) != `{"model":"provider-model-id","n":9007199254740993}` || r.URL.String() != "https://api.provider.example/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer synthetic-provider-key" || r.GetBody != nil || relay.ID != "warp-1" {
			t.Fatal("dispatch model/target/key/relay/body salah")
		}
		return &http.Response{StatusCode: 201, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"usage":{"total_tokens":3},"extra":true}`))}, nil
	}))
	if err != nil || c == nil {
		t.Fatal("coordinator valid ditolak")
	}
	response, err := c.Chat(chatRequest(`{"model":"coding-default","n":9007199254740993}`), "")
	if err != nil || response == nil {
		t.Fatal("generasi non-stream gagal")
	}
	defer response.Body.Close()
	b, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 201 || string(b) != `{"usage":{"total_tokens":3},"extra":true}` || calls != 1 {
		t.Fatal("payload/usage/status berubah")
	}
}
