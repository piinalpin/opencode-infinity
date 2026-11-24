package integration_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"opencode-infinity/internal/access"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/httpapi"
	"opencode-infinity/internal/testutil"
	"opencode-infinity/internal/transport"
)

// Harness komponen saja: tidak mengimplementasikan katalog/JSON mapping atau restart gateway produk.
func TestCredentialBoundaryThroughSOCKS5(t *testing.T) {
	const gateway = "synthetic-gateway-credential-12345"
	const admin = "synthetic-admin-credential-67890"
	const firstKey = "synthetic-primary-provider-key"
	const secondKey = "synthetic-secondary-provider-key"
	const replacement = "synthetic-replaced-provider-key"
	var expected atomic.Value
	expected.Store(firstKey)
	var bad atomic.Int32
	check := func(key func() string, path string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+key() || len(r.Header.Values("Authorization")) != 1 || r.URL.Path != path || r.URL.RawQuery != "" {
				bad.Add(1)
			}
			for _, name := range []string{"Proxy-Authorization", "Cookie", "X-API-Key", "X-Relay-Target", "Forwarded", "Idempotency-Key"} {
				if r.Header.Get(name) != "" {
					bad.Add(1)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"ok":true}`)
		}
	}
	primary := ledgerUpstream(t, check(func() string { return expected.Load().(string) }, "/v1/chat/completions"))
	secondary := ledgerUpstream(t, check(func() string { return secondKey }, "/v2/chat/completions"))
	pa := newProxy(t, "relay-a", primary.Authority, primary.Addr(), false)
	pb := newProxy(t, "relay-b", secondary.Authority, secondary.Addr(), false)
	ca, da := controlClient(t, pa.Addr(), primary.Authority, primary.Addr(), primary.Roots, false)
	cb, db := controlClient(t, pb.Addr(), secondary.Authority, secondary.Addr(), secondary.Roots, false)
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), "https://api.provider.example/v1", "https://"+primary.Authority+"/v1", 1)
	block := text[strings.Index(text, "  - id: primary\n"):]
	block = strings.Replace(block, "id: primary", "id: secondary", 1)
	block = strings.Replace(block, "https://"+primary.Authority+"/v1", "https://"+secondary.Authority+"/v2", 1)
	block = strings.Replace(block, "api_key_env: INFINITY_UPSTREAM_KEY", "api_key_env: SECONDARY_KEY", 1)
	text += block
	load := func(key string) *config.Config {
		c, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
			v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": gateway, "INFINITY_ADMIN_TOKEN": admin, "INFINITY_UPSTREAM_KEY": key, "SECONDARY_KEY": secondKey}[name]
			return v, ok
		}})
		if err != nil {
			t.Fatal("fixture konfigurasi ditolak", err)
		}
		return c
	}
	var next, dispatch atomic.Int32
	build := func(c *config.Config) http.Handler {
		policy, err := access.New(c)
		if err != nil {
			t.Fatal(err)
		}
		boundary, err := transport.NewRequestBoundary(c)
		if err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		for _, pattern := range []string{"GET /v1/models", "GET /providers/{provider}/v1/models", "POST /v1/chat/completions", "POST /providers/{provider}/v1/chat/completions"} {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.Add(1)
				id := r.PathValue("provider")
				client := ca
				if id == "secondary" {
					client = cb
				} else if id != "" && id != "primary" {
					w.WriteHeader(404)
					return
				}
				if r.Method == "GET" {
					w.WriteHeader(204)
					return
				}
				out, err := boundary.ChatCompletions(r, id)
				if err != nil {
					w.WriteHeader(400)
					return
				}
				dispatch.Add(1)
				resp, err := client.Do(out)
				if err != nil {
					w.WriteHeader(502)
					return
				}
				defer resp.Body.Close()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				_, _ = io.Copy(w, resp.Body)
			})
			guard, err := httpapi.RequireGateway(policy, handler)
			if err != nil {
				t.Fatal(err)
			}
			mux.Handle(pattern, guard)
		}
		guard, err := httpapi.RequireAdmin(policy, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.Add(1); w.WriteHeader(204) }))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/admin/status", "/metrics", "/admin/relay/disable"} {
			mux.Handle(path, guard)
		}
		return mux
	}
	c := load(firstKey)
	a := build(c)
	paths := []struct{ method, path string }{{"GET", "/v1/models"}, {"GET", "/providers/secondary/v1/models"}, {"POST", "/v1/chat/completions"}, {"POST", "/providers/secondary/v1/chat/completions"}, {"GET", "/providers/missing/v1/models"}}
	for _, path := range paths {
		for i := 0; i < 5; i++ {
			r := httptest.NewRequest(path.method, path.path, strings.NewReader(`{"fixture":true}`))
			switch i {
			case 1:
				r.Header.Set("Authorization", "Bearer wrong")
			case 2:
				r.Header.Set("Authorization", "Bearer "+admin)
			case 3:
				r.Header.Set("Authorization", "Bearer "+firstKey)
			case 4:
				r.Header["Authorization"] = []string{"Bearer " + gateway, "Bearer " + gateway}
			}
			w := httptest.NewRecorder()
			a.ServeHTTP(w, r)
			if w.Code != 401 {
				t.Fatal("invalid model credential tidak401")
			}
		}
	}
	for _, path := range []string{"/admin/status", "/metrics", "/admin/relay/disable"} {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer "+gateway)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatal("gateway token membuka admin")
		}
	}
	if next.Load() != 0 || dispatch.Load() != 0 || pa.Snapshot().Accepted != 0 || pb.Snapshot().Accepted != 0 || primary.Snapshot().Total != 0 || secondary.Snapshot().Total != 0 {
		t.Fatal("auth invalid menghubungi upstream")
	}
	run := func(h http.Handler, method, path string) {
		r := httptest.NewRequest(method, "http://gateway.fixture"+path+"?token=ignored", strings.NewReader(`{"fixture":true}`))
		r.Header.Set("Authorization", "Bearer "+gateway)
		r.Header.Set("Cookie", "private")
		r.Header.Set("Proxy-Authorization", "private")
		r.Header.Set("X-API-Key", "private")
		r.Header.Set("Forwarded", "private")
		r.Header.Set("Idempotency-Key", "private")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 200
		if method == "GET" {
			want = 204
		}
		if w.Code != want {
			t.Fatal("request berotorisasi gagal")
		}
		for _, key := range []string{gateway, admin, firstKey, secondKey, replacement} {
			if strings.Contains(w.Body.String(), key) {
				t.Fatal("credential bocor ke response")
			}
		}
	}
	for _, path := range paths[:4] {
		run(a, path.method, path.path)
	}
	// Snapshot baru memakai key baru; tidak mengganti token/config client dan tidak mengklaim restart proses.
	b := build(load(replacement))
	expected.Store(replacement)
	run(b, "POST", "/v1/chat/completions")
	expected.Store(firstKey)
	run(a, "POST", "/v1/chat/completions")
	if bad.Load() != 0 || dispatch.Load() != 4 || primary.Snapshot().Total != 3 || secondary.Snapshot().Total != 1 {
		t.Fatal("credential/dispatch teramati salah")
	}
	for _, tc := range []struct {
		u *testutil.Upstream
		p *testutil.SOCKS5
	}{{primary, pa}, {secondary, pb}} {
		if err := testutil.VerifyProxyPath(tc.u.Snapshot().Dispatches, tc.p.Snapshot()); err != nil {
			t.Fatal("dispatch tidak melalui relay")
		}
	}
	for _, d := range []*dialLog{da, db} {
		d.mu.Lock()
		direct := d.direct
		d.mu.Unlock()
		if direct != 0 {
			t.Fatal("terjadi dial langsung")
		}
	}
}

func TestDuplicateAuthorizationOnWire(t *testing.T) {
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(strings.NewReader(string(data)), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": "synthetic-gateway-credential-12345", "INFINITY_ADMIN_TOKEN": "synthetic-admin-credential-67890", "INFINITY_UPSTREAM_KEY": "synthetic-provider-key"}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := access.New(c)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	h, err := httpapi.RequireGateway(p, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(testutil.FixtureTimeout))
	_, err = fmt.Fprintf(conn, "GET /v1/models HTTP/1.1\r\nHost: gateway.fixture\r\nAuthorization: Bearer synthetic-gateway-credential-12345\r\nauthorization: Bearer wrong\r\nConnection: close\r\n\r\n")
	if err != nil {
		t.Fatal("fixture wire write gagal")
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal("fixture wire response gagal")
	}
	defer response.Body.Close()
	if response.StatusCode != 401 || calls.Load() != 0 {
		t.Fatal("header wire ganda melewati guard")
	}
}
