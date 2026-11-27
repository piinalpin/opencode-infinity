package integration_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"opencode-infinity/internal/testutil"
)

func modelCall(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "http://gateway.fixture"+path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+modelGatewayToken)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Cookie", "private")
	r.Header.Set("Proxy-Authorization", "private")
	r.Header.Set("Idempotency-Key", "client-replay-key")
	r.Header.Set("X-Idempotency-Key", "client-replay-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestModelEncodedJSON(t *testing.T) {
	const payload = `{"usage":{"total_tokens":3},"extra":true}`
	var encoded bytes.Buffer
	writer := gzip.NewWriter(&encoded)
	if _, err := io.WriteString(writer, payload); err != nil {
		t.Fatal("fixture gzip gagal")
	}
	if err := writer.Close(); err != nil {
		t.Fatal("fixture gzip close gagal")
	}
	u := ledgerUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded.Bytes())
	})
	p := newProxy(t, "warp-1", u.Authority, u.Addr(), false)
	client, _ := controlClient(t, p.Addr(), u.Authority, u.Addr(), u.Roots, false)
	client.Transport.(*http.Transport).DisableCompression = true
	c := modelConfiguration(t, "https://api.provider.example/v1", "https://"+u.Authority+"/v1", "socks5h://warp-1:1080", "socks5h://"+p.Addr())
	h, err := fixtureAPI(c, &modelFixtureDispatcher{clients: map[string]*http.Client{"warp-1": client}})
	if err != nil {
		t.Fatal(err)
	}
	w := modelCall(t, h, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
	if w.Code != 200 || w.Header().Get("Content-Encoding") != "gzip" || !bytes.Equal(w.Body.Bytes(), encoded.Bytes()) {
		t.Fatal("body gzip diteruskan tanpa metadata encoding yang sama")
	}
	decoder, err := gzip.NewReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal("gzip client tidak dapat dibaca")
	}
	defer decoder.Close()
	decoded, err := io.ReadAll(decoder)
	if err != nil || string(decoded) != payload {
		t.Fatal("JSON/usage berubah setelah decoding client")
	}
	if err := testutil.VerifyProxyPath(u.Snapshot().Dispatches, p.Snapshot()); err != nil {
		t.Fatal("gzip melewati relay yang salah")
	}
}

func TestModelAliasJSON(t *testing.T) {
	input := ` {"model":"coding-default","stream":false,"messages":[{"model":"nested","content":"日本語"}],"tools":[{"type":"function"}],"extra":{"n":9007199254740993,"huge":1e1000,"a":1,"a":2},"target":"https://ignored.invalid"} `
	primaryJSON := `{"model":"provider-model-id","usage":{"total_tokens":7},"extra":{"n":9007199254740993}}`
	secondaryJSON := `{"model":"secondary-model-id","extra":null}`
	var bad atomic.Int32
	check := func(key, path, model, result string, status int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			want := strings.Replace(input, `"coding-default"`, `"`+model+`"`, 1)
			if err != nil || string(body) != want || r.Header.Get("Authorization") != "Bearer "+key || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" {
				bad.Add(1)
			}
			for _, name := range []string{"Cookie", "Proxy-Authorization", "Idempotency-Key", "X-Idempotency-Key", "X-API-Key", "Forwarded"} {
				if r.Header.Get(name) != "" {
					bad.Add(1)
				}
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, result)
		}
	}
	h := newModelHarness(t, check(modelPrimaryKey, "/v1/chat/completions", "provider-model-id", primaryJSON, 201), check(modelSecondaryKey, "/v2/chat/completions", "secondary-model-id", secondaryJSON, 202))
	for _, path := range []string{"/v1/models", "/providers/secondary/v1/models", "/v1/chat/completions", "/providers/secondary/v1/chat/completions"} {
		method := "POST"
		if strings.HasSuffix(path, "/models") {
			method = "GET"
		}
		for _, token := range []string{"", "wrong", modelAdminToken, modelPrimaryKey} {
			r := httptest.NewRequest(method, path, strings.NewReader(input))
			r.Header.Set("Content-Type", "application/json")
			if token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			h.handler.ServeHTTP(w, r)
			if w.Code != 401 {
				t.Fatal("auth invalid tidak ditolak sebelum jaringan")
			}
		}
	}
	if w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"missing"}`); w.Code != 404 {
		t.Fatal("alias missing tidak ditolak")
	}
	if h.a.Snapshot().Accepted != 0 || h.b.Snapshot().Accepted != 0 || h.primary.Snapshot().Total != 0 || h.secondary.Snapshot().Total != 0 {
		t.Fatal("auth/alias ditolak membuka jaringan")
	}
	for _, path := range []string{"/v1/models", "/providers/primary/v1/models", "/providers/secondary/v1/models"} {
		w := modelCall(t, h.handler, "GET", path, "")
		var list struct {
			Object string
			Data   []struct {
				ID      string
				OwnedBy string `json:"owned_by"`
			}
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &list) != nil || list.Object != "list" || len(list.Data) != 1 || list.Data[0].ID != "coding-default" {
			t.Fatal("katalog alias scope salah")
		}
		owner := "primary"
		if strings.Contains(path, "secondary") {
			owner = "secondary"
		}
		if list.Data[0].OwnedBy != owner {
			t.Fatal("owner katalog salah")
		}
		for _, hidden := range []string{"provider-model-id", "secondary-model-id", modelPrimaryKey, modelSecondaryKey, "upstream.invalid", "warp-1"} {
			if strings.Contains(w.Body.String(), hidden) {
				t.Fatal("detail server bocor di katalog")
			}
		}
	}
	if h.primary.Snapshot().Total != 0 || h.secondary.Snapshot().Total != 0 || h.a.Snapshot().Accepted != 0 || h.b.Snapshot().Accepted != 0 {
		t.Fatal("katalog melakukan jaringan")
	}
	for _, tc := range []struct {
		path, body string
		status     int
	}{{"/v1/chat/completions", primaryJSON, 201}, {"/providers/primary/v1/chat/completions", primaryJSON, 201}, {"/providers/secondary/v1/chat/completions", secondaryJSON, 202}} {
		w := modelCall(t, h.handler, "POST", tc.path+"?target=https://ignored.invalid", input)
		if w.Code != tc.status || w.Body.String() != tc.body || w.Header().Get("Content-Type") != "application/json; charset=utf-8" || w.Header().Get("X-Infinity-Error-Source") != "upstream" {
			t.Fatal("JSON/status/content type/usage berubah")
		}
	}
	before := h.primary.Snapshot().Total + h.secondary.Snapshot().Total
	for _, tc := range []struct{ method, path, body, code string }{{"POST", "/v1/chat/completions", `{"model":"missing"}`, "model_not_found"}, {"POST", "/providers/missing/v1/chat/completions", `{"model":"coding-default"}`, "upstream_not_found"}, {"GET", "/providers/missing/v1/models", "", "upstream_not_found"}} {
		w := modelCall(t, h.handler, tc.method, tc.path, tc.body)
		if w.Code != 404 || !strings.Contains(w.Body.String(), `"code":"`+tc.code+`"`) {
			t.Fatal("unknown alias/provider tidak ditolak")
		}
	}
	if before != h.primary.Snapshot().Total+h.secondary.Snapshot().Total || bad.Load() != 0 {
		t.Fatal("dispatch/remap/key/path salah")
	}
	h.verifyPath(t)
	if len(h.a.Snapshot().Connects) != 1 || len(h.b.Snapshot().Connects) != 1 {
		t.Fatal("koneksi relay tidak dipakai ulang")
	}
	for _, p := range []*testutil.SOCKS5{h.a, h.b} {
		if p.Snapshot().Connects[0].Hostname != testutil.Hostname {
			t.Fatal("hostname tidak diserahkan ke proxy")
		}
	}
}

func TestModelNetworkBoundaries(t *testing.T) {
	t.Run("dead-relay", func(t *testing.T) {
		h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{}`) }, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{}`) })
		if err := h.a.Close(); err != nil {
			t.Fatal(err)
		}
		w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
		if w.Code != 502 || h.primary.Snapshot().Total != 0 || h.secondary.Snapshot().Total != 0 {
			t.Fatal("relay mati melakukan fallback/dispatch")
		}
		h.verifyPath(t)
	})
	t.Run("tls-hostname", func(t *testing.T) {
		u := ledgerUpstream(t, func(http.ResponseWriter, *http.Request) { t.Error("TLS mismatch mencapai handler") })
		_, port, _ := net.SplitHostPort(u.Authority)
		wrong := net.JoinHostPort("wrong.invalid", port)
		p := newProxy(t, "warp-1", wrong, u.Addr(), false)
		client, log := controlClient(t, p.Addr(), wrong, u.Addr(), u.Roots, false)
		c := modelConfiguration(t, "https://api.provider.example/v1", "https://"+wrong+"/v1", "socks5h://warp-1:1080", "socks5h://"+p.Addr())
		h, err := fixtureAPI(c, &modelFixtureDispatcher{clients: map[string]*http.Client{"warp-1": client}})
		if err != nil {
			t.Fatal(err)
		}
		w := modelCall(t, h, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
		if w.Code != 502 || u.Snapshot().Total != 0 || len(p.Snapshot().Connects) != 1 {
			t.Fatal("TLS mismatch tidak ditolak")
		}
		log.mu.Lock()
		direct := log.direct
		log.mu.Unlock()
		if direct != 0 {
			t.Fatal("TLS mismatch melakukan direct fallback")
		}
	})
	t.Run("redirect", func(t *testing.T) {
		target := "https://ignored.invalid/other"
		h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", target)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(307)
			_, _ = io.WriteString(w, `{"redirect":true}`)
		}, func(http.ResponseWriter, *http.Request) { t.Error("redirect mencapai upstream lain") })
		target = "https://" + h.secondary.Authority + "/v2/chat/completions"
		w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
		if w.Code != 502 || !strings.Contains(w.Body.String(), `"code":"upstream_redirect_rejected"`) || w.Header().Get("X-Infinity-Error-Source") != "gateway" || w.Header().Get("X-Request-ID") == "" || w.Header().Get("Location") != "" || h.primary.Snapshot().Total != 1 || h.secondary.Snapshot().Total != 0 || h.b.Snapshot().Accepted != 0 {
			t.Fatal("redirect tidak ditolak terstruktur atau mencapai origin lain")
		}
		h.verifyPath(t)
	})
}

func TestModelAmbiguousPOST(t *testing.T) {
	for _, warm := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "reused"}[warm], func(t *testing.T) {
			var drop atomic.Bool
			var dropFailed atomic.Bool
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				if drop.Load() {
					if testutil.DropConnection(w) != nil {
						dropFailed.Store(true)
					}
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{}`)
			}, func(http.ResponseWriter, *http.Request) { t.Error("POST ambigu mengganti upstream") }, "max_active_requests: 20", "max_active_requests: 1")
			if warm {
				if w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`); w.Code != 200 {
					t.Fatal("warmup gagal")
				}
			}
			drop.Store(true)
			w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
			want := 1
			if warm {
				want = 2
			}
			if w.Code != 502 || h.primary.Snapshot().Total != want || len(h.a.Snapshot().Connects) != 1 || dropFailed.Load() {
				t.Fatal("POST ambigu diulang/disembunyikan")
			}
			drop.Store(false)
			if w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`); w.Code != 200 {
				t.Fatal("slot error tidak dilepas")
			}
			h.verifyPath(t)
		})
	}
}

func TestModelCancellation(t *testing.T) {
	for _, headers := range []bool{false, true} {
		t.Run(map[bool]string{false: "waiting-header", true: "reading-body"}[headers], func(t *testing.T) {
			started := make(chan struct{}, 1)
			cancelled := make(chan time.Time, 1)
			var wait atomic.Bool
			wait.Store(true)
			h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
				if !wait.Load() {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				if headers {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"partial":`)
					http.NewResponseController(w).Flush()
				}
				started <- struct{}{}
				<-r.Context().Done()
				cancelled <- time.Now()
			}, func(http.ResponseWriter, *http.Request) { t.Error("cancel mengganti upstream") }, "max_active_requests: 20", "max_active_requests: 1")
			server := httptest.NewServer(h.handler)
			defer server.Close()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/v1/chat/completions", strings.NewReader(`{"model":"coding-default"}`))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer "+modelGatewayToken)
			request.Header.Set("Content-Type", "application/json")
			done := make(chan struct{}, 1)
			go func() {
				response, err := server.Client().Do(request)
				if err == nil {
					_, _ = io.Copy(io.Discard, response.Body)
					response.Body.Close()
				}
				done <- struct{}{}
			}()
			await(t, started)
			at := time.Now()
			cancel()
			observed := await(t, cancelled)
			await(t, done)
			if observed.Sub(at) > 2*time.Second {
				t.Fatal("cancellation upstream melewati 2 detik")
			}
			wait.Store(false)
			// Sinkronkan return handler gateway sebelum mencoba admission lagi; tidak menunggu cleanup fixture.
			deadline := time.Now().Add(2 * time.Second)
			for {
				w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
				if w.Code == 200 {
					break
				}
				if w.Code != 429 || time.Now().After(deadline) {
					t.Fatal("slot cancellation tidak dilepas")
				}
				time.Sleep(time.Millisecond)
			}
			h.verifyPath(t)
		})
	}
}

func TestModelPartialJSONAborts(t *testing.T) {
	h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"partial":`)
		http.NewResponseController(w).Flush()
		if testutil.DropConnection(w) != nil {
			t.Error("fixture drop gagal")
		}
	}, func(http.ResponseWriter, *http.Request) { t.Error("partial JSON mengganti upstream") })
	server := httptest.NewServer(h.handler)
	defer server.Close()
	request, err := http.NewRequest("POST", server.URL+"/v1/chat/completions", strings.NewReader(`{"model":"coding-default"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+modelGatewayToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err == nil {
		defer response.Body.Close()
		body, readErr := io.ReadAll(response.Body)
		if readErr == nil || bytes.Contains(body, []byte(`"error"`)) {
			t.Fatal("JSON parsial diberi EOF sukses/envelope pengganti")
		}
	}
	if h.primary.Snapshot().Total != 1 || h.secondary.Snapshot().Total != 0 {
		t.Fatal("JSON parsial diulang")
	}
	h.verifyPath(t)
}

func TestModelTimeoutReleases(t *testing.T) {
	var wait atomic.Bool
	wait.Store(true)
	cancelled := make(chan struct{}, 1)
	h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if wait.Load() {
			<-r.Context().Done()
			cancelled <- struct{}{}
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}, func(http.ResponseWriter, *http.Request) { t.Error("timeout mengganti upstream") }, "total_request_seconds: 1800", "total_request_seconds: 1", "max_active_requests: 20", "max_active_requests: 1")
	w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
	if w.Code != 504 || !strings.Contains(w.Body.String(), `"code":"upstream_timeout"`) {
		t.Fatal("timeout fixture salah kategori")
	}
	await(t, cancelled)
	wait.Store(false)
	if w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`); w.Code != 200 {
		t.Fatal("slot timeout tidak dilepas")
	}
	h.verifyPath(t)
}
