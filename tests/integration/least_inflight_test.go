package integration_test

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"opencode-infinity/internal/httpapi"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
	"opencode-infinity/internal/testutil"
)

type leastWire struct {
	server     *httptest.Server
	components *httpapi.Components
	upstream   *testutil.Upstream
	a, b       *testutil.SOCKS5
	logs       []*dialLog
}

func newLeastWire(t *testing.T, handler http.HandlerFunc) *leastWire {
	t.Helper()
	upstream := ledgerUpstream(t, handler)
	a := newProxy(t, "warp-1", upstream.Authority, upstream.Addr(), false)
	b := newProxy(t, "warp-2", upstream.Authority, upstream.Addr(), false)
	ca, la := controlClient(t, a.Addr(), upstream.Authority, upstream.Addr(), upstream.Roots, false)
	cb, lb := controlClient(t, b.Addr(), upstream.Authority, upstream.Addr(), upstream.Roots, false)
	for _, client := range []*http.Client{ca, cb} {
		client.Timeout = 0
		client.Transport.(*http.Transport).DisableKeepAlives = true
	}
	c := modelConfiguration(t, "https://api.provider.example/v1", "https://"+upstream.Authority+"/v1", "socks5h://warp-1:1080", "socks5h://"+a.Addr(), "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://"+b.Addr()+"\n    max_inflight: 1\nupstreams:", "max_inflight: 5", "max_inflight: 1", "relays: [warp-1]", "relays: [warp-2, warp-1]", "routing: fixed", "routing: least_inflight\n        stateless: true")
	components, err := httpapi.NewComponents(c, &modelFixtureDispatcher{clients: map[string]*http.Client{"warp-1": ca, "warp-2": cb}}, status.Build{}, time.Now, pool.Policy{Failures: 1, Successes: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range c.Relays() {
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			at := time.Now()
			if err := components.Pool.Observe(r.ID, layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
		}
	}
	server := httptest.NewServer(components.Models)
	server.Client().Timeout = 3 * time.Second
	t.Cleanup(server.Close)
	return &leastWire{server: server, components: components, upstream: upstream, a: a, b: b, logs: []*dialLog{la, lb}}
}
func (h *leastWire) call(t *testing.T, stream bool) *http.Response {
	t.Helper()
	body := `{"model":"coding-default","stream":false,"extra":{"value":1}}`
	if stream {
		body = `{"model":"coding-default","stream":true,"extra":{"value":1}}`
	}
	response, err := h.server.Client().Do(gatewayStreamRequest(t, h.server, "/v1/chat/completions", body))
	if err != nil {
		t.Fatal(err)
	}
	return response
}
func (h *leastWire) verify(t *testing.T) {
	t.Helper()
	combined := h.a.Snapshot()
	combined.Connects = append(combined.Connects, h.b.Snapshot().Connects...)
	if err := testutil.VerifyProxyPath(h.upstream.Snapshot().Dispatches, combined); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*testutil.SOCKS5{h.a, h.b} {
		for _, c := range p.Snapshot().Connects {
			if c.Hostname != testutil.Hostname || c.Refused {
				t.Fatal("DNS/CONNECT relay salah")
			}
		}
	}
	for _, log := range h.logs {
		log.mu.Lock()
		direct := log.direct
		log.mu.Unlock()
		if direct != 0 {
			t.Fatal("dial langsung teramati")
		}
	}
}
func (h *leastWire) reject(t *testing.T, status int, code string) {
	t.Helper()
	a, b, u := h.a.Snapshot(), h.b.Snapshot(), h.upstream.Snapshot()
	assertWireGatewayError(t, h.call(t, true), status, code)
	aa, bb := h.a.Snapshot(), h.b.Snapshot()
	if aa.Accepted != a.Accepted || bb.Accepted != b.Accepted || len(aa.Connects) != len(a.Connects) || len(bb.Connects) != len(b.Connects) || h.upstream.Snapshot().Total != u.Total {
		t.Fatal("penolakan membuka koneksi/CONNECT/dispatch")
	}
}

func TestLeastInflightWireStreamCooldownAndCapacity(t *testing.T) {
	proceed := make(chan struct{})
	release := sync.OnceFunc(func() { close(proceed) })
	defer release()
	var hold atomic.Bool
	hold.Store(true)
	var posts atomic.Int32
	h := newLeastWire(t, func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), `"model":"provider-model-id"`) || !strings.Contains(string(body), `"extra":{"value":1}`) || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer "+modelPrimaryKey {
			t.Error("least mengganti target/key/semantik body")
		}
		if hold.Load() {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"held\":true}\n\n")
			_ = http.NewResponseController(w).Flush()
			select {
			case <-proceed:
			case <-r.Context().Done():
				return
			}
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	first := h.call(t, true)
	defer first.Body.Close()
	firstReader := bufio.NewReader(first.Body)
	if event, err := readEvent(firstReader); err != nil || event != `{"held":true}` {
		t.Fatal("event pertama tidak di-flush")
	}
	if h.b.Snapshot().Accepted != 1 || h.a.Snapshot().Accepted != 0 {
		t.Fatal("tie tidak memakai urutan allowlist, atau dispatch masih primary")
	}
	second := h.call(t, true)
	defer second.Body.Close()
	secondReader := bufio.NewReader(second.Body)
	if _, err := readEvent(secondReader); err != nil {
		t.Fatal(err)
	}
	if h.a.Snapshot().Accepted != 1 || h.b.Snapshot().Accepted != 1 || posts.Load() != 2 {
		t.Fatal("stream pertama tidak menahan slot pilihan")
	}
	h.reject(t, 429, "gateway_capacity_exceeded")
	if err := h.components.Pool.ExtendRelayCooldown("warp-2", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	h.reject(t, 429, "gateway_capacity_exceeded") // warp-2 cooldown, warp-1 masih penuh.
	release()
	hold.Store(false)
	for _, pair := range []struct {
		reader   *bufio.Reader
		response *http.Response
	}{{firstReader, first}, {secondReader, second}} {
		tail, err := io.ReadAll(pair.reader)
		pair.response.Body.Close()
		if err != nil || !strings.Contains(string(tail), "[DONE]") {
			t.Fatal("cooldown memutus stream admitted")
		}
	}
	// Body klien selesai dibaca dapat mendahului defer release handler beberapa instruksi.
	waitEligibility(t, func() bool { return h.a.Snapshot().Active == 0 && h.b.Snapshot().Active == 0 })
	next := h.call(t, false)
	data, err := io.ReadAll(next.Body)
	next.Body.Close()
	if err != nil || next.StatusCode != 200 || string(data) != `{"ok":true}` {
		t.Fatal("kapasitas/stream tidak pulih")
	}
	if h.a.Snapshot().Accepted != 2 || h.b.Snapshot().Accepted != 1 || posts.Load() != 3 {
		t.Fatal("kandidat cooldown tidak dilewati")
	}
	if err := h.components.Pool.ExtendRelayCooldown("warp-1", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	h.reject(t, 503, "no_eligible_relay")
	h.verify(t)
}

func TestLeastInflightWireStableSerialTie(t *testing.T) {
	h := newLeastWire(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})
	for range 4 {
		response := h.call(t, false)
		_, err := io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != 200 {
			t.Fatal("generasi serial gagal")
		}
	}
	if h.a.Snapshot().Accepted != 0 || h.b.Snapshot().Accepted != 4 || h.upstream.Snapshot().Total != 4 {
		t.Fatal("tie serial tidak stabil atau ada dispatch tambahan")
	}
	h.verify(t)
}
