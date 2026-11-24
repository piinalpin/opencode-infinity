package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"sync"
	"testing"
)

const gatewayToken = "synthetic-gateway-credential-12345"
const adminToken = "synthetic-admin-credential-67890"
const primaryKey = "synthetic-primary-upstream-key"
const secondaryKey = "synthetic-secondary-upstream-key"

func requestSnapshot(t *testing.T, key string) *config.Config {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), "https://api.provider.example/v1", "https://primary.example/prefix%20space/v1/", 1)
	block := text[strings.Index(text, "  - id: primary\n"):]
	block = strings.Replace(block, "id: primary", "id: secondary", 1)
	block = strings.Replace(block, "primary.example/prefix%20space/v1/", "secondary.example/api", 1)
	block = strings.Replace(block, "api_key_env: INFINITY_UPSTREAM_KEY", "api_key_env: SECONDARY_KEY", 1)
	c, err := config.Load(strings.NewReader(text+block), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": gatewayToken, "INFINITY_ADMIN_TOKEN": adminToken, "INFINITY_UPSTREAM_KEY": key, "SECONDARY_KEY": secondaryKey}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal("fixture konfigurasi ditolak", err)
	}
	return c
}
func TestServerCredentialRequest(t *testing.T) {
	boundary, err := NewRequestBoundary(requestSnapshot(t, primaryKey))
	if err != nil {
		t.Fatal("boundary valid ditolak", err)
	}
	for _, tc := range []struct{ id, host, path, key string }{{"", "primary.example", "/prefix%20space/v1/chat/completions", primaryKey}, {"primary", "primary.example", "/prefix%20space/v1/chat/completions", primaryKey}, {"secondary", "secondary.example", "/api/chat/completions", secondaryKey}} {
		in := httptest.NewRequest(http.MethodPost, "https://attacker.example/free?access_token=ignored", strings.NewReader(`{"opaque":true}`))
		in.Header.Set("Authorization", "Bearer "+gatewayToken)
		in.Host = "spoofed.example"
		out, err := boundary.ChatCompletions(in, tc.id)
		if err != nil || out == nil {
			t.Fatal("request valid ditolak", err)
		}
		if out.Method != "POST" || out.URL.Scheme != "https" || out.URL.Host != tc.host || out.Host != tc.host || out.URL.EscapedPath() != tc.path || out.URL.RawQuery != "" || out.RequestURI != "" {
			t.Fatal("target server tidak dipakai")
		}
		if values := out.Header.Values("Authorization"); len(values) != 1 || values[0] != "Bearer "+tc.key {
			t.Fatal("credential server tidak tepat")
		}
		if in.Header.Get("Authorization") != "Bearer "+gatewayToken || in.Host != "spoofed.example" || in.URL.Host != "attacker.example" {
			t.Fatal("input dimutasi")
		}
	}
	if out, err := boundary.ChatCompletions(httptest.NewRequest("POST", "/", nil), "unknown"); err == nil || out != nil {
		t.Fatal("upstream unknown memakai fallback")
	}
	for _, c := range []*config.Config{nil, {}} {
		if b, err := NewRequestBoundary(c); err == nil || b != nil {
			t.Fatal("snapshot invalid diterima")
		}
	}
}

func TestForwardHeaderPolicy(t *testing.T) {
	b, err := NewRequestBoundary(requestSnapshot(t, primaryKey))
	if err != nil {
		t.Fatal(err)
	}
	in := httptest.NewRequest("POST", "https://attacker.example/ignored?token=secret", nil)
	in.Header = http.Header{"Authorization": {"Bearer " + gatewayToken}, "authorization": {"Bearer bad"}, "Content-Type": {"application/json"}, "Accept": {"application/json", "text/event-stream"}, "Proxy-Authorization": {"private"}, "Cookie": {"private"}, "X-API-Key": {"private"}, "Forwarded": {"private"}, "X-Forwarded-Host": {"attacker.example"}, "Idempotency-Key": {"private"}, "Connection": {"keep-alive"}, "Content-Encoding": {"identity"}}
	out, err := b.ChatCompletions(in, "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Header.Get("Content-Type") != "application/json" || len(out.Header.Values("Accept")) != 2 {
		t.Fatal("header representasi hilang")
	}
	if len(out.Header) != 3 {
		t.Fatal("header client di luar allowlist diteruskan")
	}
	out.Header["Accept"][0] = "changed"
	if in.Header["Accept"][0] != "application/json" {
		t.Fatal("header input dimutasi")
	}
	for _, header := range []http.Header{
		{"X-Relay-Target": {"https://attacker.example"}}, {"x-relay-path": {"private"}},
		{"Content-Encoding": {"gzip"}}, {"Content-Encoding": {"identity", "identity"}},
		{"Content-Type": {"application/json", "text/plain"}}, {"Content-Type": {"application/json"}, "content-type": {"text/plain"}},
		{"Accept": {"text/plain\r\nInjected: private"}}, {"Accept": {"private\x00"}},
		{"Accept": {strings.Repeat("x", 8183)}}, {"Connection": {strings.Repeat("x", 8193)}},
	} {
		in.Header = header
		if out, err := b.ChatCompletions(in, ""); err == nil || out != nil {
			t.Fatal("header tidak aman diterima")
		}
	}
	in.Header = http.Header{"Accept": {strings.Repeat("x", 8182)}}
	if _, err := b.ChatCompletions(in, ""); err != nil {
		t.Fatal("header tepat 8 KiB ditolak", err)
	}
	in.Header = http.Header{"Content-Type": {"application/json"}, "Accept": {"text/event-stream"}, "Connection": {"Content-Type, Authorization"}}
	out, err = b.ChatCompletions(in, "")
	if err != nil || out.Header.Get("Content-Type") != "" || out.Header.Get("Authorization") != "Bearer "+primaryKey {
		t.Fatal("header hop-by-hop nominasi client mengubah auth atau diteruskan")
	}
}

type opaqueBody struct{ reads, closes int }

func (b *opaqueBody) Read([]byte) (int, error) { b.reads++; panic("preparation membaca body") }
func (b *opaqueBody) Close() error             { b.closes++; return nil }
func TestRequestOwnership(t *testing.T) {
	b, err := NewRequestBoundary(requestSnapshot(t, primaryKey))
	if err != nil {
		t.Fatal(err)
	}
	body := &opaqueBody{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := httptest.NewRequest("POST", "https://attacker.example/ignored", nil).WithContext(ctx)
	in.Body = body
	in.ContentLength = 27
	in.GetBody = func() (io.ReadCloser, error) { t.Fatal("GetBody tidak boleh dipakai"); return nil, nil }
	in.Trailer = http.Header{"Authorization": {"private"}}
	in.TransferEncoding = []string{"chunked"}
	in.Header.Set("Idempotency-Key", "private")
	in.Header.Set("X-Idempotency-Key", "private")
	out, err := b.ChatCompletions(in, "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Body != body || out.ContentLength != 27 || body.reads != 0 || body.closes != 0 || out.Context() != ctx {
		t.Fatal("ownership/metadata body berubah")
	}
	if out.GetBody != nil || len(out.Trailer) != 0 || len(out.TransferEncoding) != 0 || out.Header.Get("Idempotency-Key") != "" || out.Header.Get("X-Idempotency-Key") != "" {
		t.Fatal("metadata replay/trailer client diteruskan")
	}
	if in.GetBody == nil || len(in.Trailer) != 1 || len(in.TransferEncoding) != 1 {
		t.Fatal("input request dimutasi")
	}
	cancel()
	if out.Context().Err() != context.Canceled {
		t.Fatal("cancellation tidak dipertahankan")
	}
}
func TestInvalidRequestFailsClosed(t *testing.T) {
	b, err := NewRequestBoundary(requestSnapshot(t, primaryKey))
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "DELETE", ""} {
		in := httptest.NewRequest("POST", "/", nil)
		in.Method = method
		if out, err := b.ChatCompletions(in, ""); err == nil || out != nil {
			t.Fatal("method invalid diterima")
		}
	}
	in := httptest.NewRequest("POST", "/", nil)
	in.ContentLength = -2
	if out, err := b.ChatCompletions(in, ""); err == nil || out != nil {
		t.Fatal("panjang body invalid diterima")
	}
	for _, empty := range []*RequestBoundary{nil, {}} {
		func() {
			defer func() {
				if recover() != nil {
					t.Error("boundary kosong panic")
				}
			}()
			if out, err := empty.ChatCompletions(in, ""); err == nil || out != nil {
				t.Error("boundary kosong diterima")
			}
		}()
	}
	func() {
		defer func() {
			if recover() != nil {
				t.Error("request nil panic")
			}
		}()
		if out, err := b.ChatCompletions(nil, ""); err == nil || out != nil {
			t.Error("request nil diterima")
		}
	}()
}

func TestCredentialSnapshotsAndPrivacy(t *testing.T) {
	a, err := NewRequestBoundary(requestSnapshot(t, primaryKey))
	if err != nil {
		t.Fatal(err)
	}
	changed := "synthetic-replaced-upstream-key"
	b, err := NewRequestBoundary(requestSnapshot(t, changed))
	if err != nil {
		t.Fatal(err)
	}
	in := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	in.Header.Set("Authorization", "Bearer "+gatewayToken)
	for _, tc := range []struct {
		boundary *RequestBoundary
		key      string
	}{{a, primaryKey}, {b, changed}, {a, primaryKey}} {
		out, err := tc.boundary.ChatCompletions(in, "")
		if err != nil {
			t.Fatal(err)
		}
		if out.Header.Get("Authorization") != "Bearer "+tc.key || in.Header.Get("Authorization") != "Bearer "+gatewayToken {
			t.Fatal("credential snapshot tercampur")
		}
		out.Header.Set("Authorization", "changed")
	}
	for _, object := range []any{a, *a, struct{ boundary RequestBoundary }{*a}} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			for _, secret := range []string{gatewayToken, adminToken, primaryKey, secondaryKey} {
				if strings.Contains(fmt.Sprintf(format, object), secret) {
					t.Fatal("diagnostik boundary membocorkan credential")
				}
			}
		}
		data, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), primaryKey) {
			t.Fatal("JSON boundary membocorkan credential")
		}
	}
	_, err = a.ChatCompletions(in, "private-sentinel")
	if err == nil {
		t.Fatal("unknown diterima")
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		for _, format := range []string{"%v", "%+v", "%#v"} {
			if strings.Contains(fmt.Sprintf(format, e), "private-sentinel") {
				t.Fatal("input bocor lewat error")
			}
		}
	}
}
func TestConcurrentRequestPreparation(t *testing.T) {
	b, err := NewRequestBoundary(requestSnapshot(t, primaryKey))
	if err != nil {
		t.Fatal(err)
	}
	in := httptest.NewRequest("POST", "/", nil)
	in.Header.Set("Authorization", "Bearer "+gatewayToken)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				out, err := b.ChatCompletions(in, "secondary")
				if err != nil || out.Header.Get("Authorization") != "Bearer "+secondaryKey {
					t.Error("preparation concurrent berubah")
					return
				}
				out.Header.Set("Authorization", "changed")
			}
		}()
	}
	wg.Wait()
}

func TestBoundaryOwnsSnapshotValue(t *testing.T) {
	source := requestSnapshot(t, primaryKey)
	boundary, err := NewRequestBoundary(source)
	if err != nil {
		t.Fatal(err)
	}
	replacement := requestSnapshot(t, "synthetic-new-provider-key")
	*source = *replacement
	out, err := boundary.ChatCompletions(httptest.NewRequest("POST", "/", nil), "")
	if err != nil || out.Header.Get("Authorization") != "Bearer "+primaryKey {
		t.Fatal("penggantian object caller mencampur snapshot target/key")
	}
}
