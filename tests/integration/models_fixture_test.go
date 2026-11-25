package integration_test

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"opencode-infinity/internal/config"
	"opencode-infinity/internal/httpapi"
	"opencode-infinity/internal/testutil"
)

const modelGatewayToken = "synthetic-gateway-credential-12345"
const modelAdminToken = "synthetic-admin-credential-67890"
const modelPrimaryKey = "synthetic-primary-provider-key"
const modelSecondaryKey = "synthetic-secondary-provider-key"

// Adapter efek fixture tepercaya, bukan transport runtime. Tidak memilih fallback/default client.
type modelFixtureDispatcher struct{ clients map[string]*http.Client }

func (d *modelFixtureDispatcher) Dispatch(r *http.Request, relay config.Relay) (*http.Response, error) {
	c, ok := d.clients[relay.ID]
	if !ok {
		return nil, errors.New("fixture_unknown_relay")
	}
	if r.GetBody != nil || r.Header.Get("Idempotency-Key") != "" || r.Header.Get("X-Idempotency-Key") != "" {
		return nil, errors.New("fixture_replay_metadata")
	}
	return c.Do(r)
}

type modelHarness struct {
	handler            http.Handler
	primary, secondary *testutil.Upstream
	a, b               *testutil.SOCKS5
	logs               []*dialLog
	clients            map[string]*http.Client
}

func modelConfiguration(t *testing.T, edits ...string) *config.Config {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.NewReplacer("max_body_bytes: 10485760", "max_body_bytes: 4096").Replace(string(data))
	text = strings.NewReplacer(edits...).Replace(text)
	c, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": modelGatewayToken, "INFINITY_ADMIN_TOKEN": modelAdminToken, "INFINITY_UPSTREAM_KEY": modelPrimaryKey, "SECONDARY_KEY": modelSecondaryKey}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal("konfigurasi model fixture ditolak", err)
	}
	return c
}
func newModelHarness(t *testing.T, primary, secondary http.HandlerFunc, edits ...string) *modelHarness {
	t.Helper()
	p := ledgerUpstream(t, primary)
	s := ledgerUpstream(t, secondary)
	a := newProxy(t, "warp-1", p.Authority, p.Addr(), false)
	b := newProxy(t, "warp-2", s.Authority, s.Addr(), false)
	ca, da := controlClient(t, a.Addr(), p.Authority, p.Addr(), p.Roots, false)
	cb, db := controlClient(t, b.Addr(), s.Authority, s.Addr(), s.Roots, false)
	// Ambil contoh upstream yang sama agar shape config fixture mengikuti loader produk.
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	block := text[strings.Index(text, "  - id: primary\n"):]
	block = strings.NewReplacer("id: primary", "id: secondary", "https://api.provider.example/v1", "https://"+s.Authority+"/v2", "api_key_env: INFINITY_UPSTREAM_KEY", "api_key_env: SECONDARY_KEY", "[warp-1]", "[warp-2]", "primary_relay: warp-1", "primary_relay: warp-2", "provider-model-id", "secondary-model-id").Replace(block)
	text = strings.NewReplacer("https://api.provider.example/v1", "https://"+p.Authority+"/v1", "socks5h://warp-1:1080", "socks5h://"+a.Addr(), "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://"+b.Addr()+"\n    max_inflight: 5\nupstreams:").Replace(text) + block
	text = strings.NewReplacer("max_body_bytes: 10485760", "max_body_bytes: 4096").Replace(text)
	text = strings.NewReplacer(edits...).Replace(text)
	c, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": modelGatewayToken, "INFINITY_ADMIN_TOKEN": modelAdminToken, "INFINITY_UPSTREAM_KEY": modelPrimaryKey, "SECONDARY_KEY": modelSecondaryKey}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal("konfigurasi dua upstream ditolak", err)
	}
	// Timeout header berasal dari snapshot; total dikendalikan context coordinator, bukan cap client tersembunyi.
	for _, client := range []*http.Client{ca, cb} {
		client.Timeout = 0
		client.Transport.(*http.Transport).ResponseHeaderTimeout = time.Duration(c.Timeouts().ResponseHeadersSeconds) * time.Second
	}
	clients := map[string]*http.Client{"warp-1": ca, "warp-2": cb}
	h, err := httpapi.NewAPI(c, &modelFixtureDispatcher{clients: clients})
	if err != nil {
		t.Fatal(err)
	}
	return &modelHarness{handler: h, primary: p, secondary: s, a: a, b: b, logs: []*dialLog{da, db}, clients: clients}
}

func (h *modelHarness) verifyPath(t *testing.T) {
	t.Helper()
	for _, pair := range []struct {
		u *testutil.Upstream
		p *testutil.SOCKS5
	}{{h.primary, h.a}, {h.secondary, h.b}} {
		if err := testutil.VerifyProxyPath(pair.u.Snapshot().Dispatches, pair.p.Snapshot()); err != nil {
			t.Fatal("dispatch melewati relay yang salah")
		}
	}
	for _, log := range h.logs {
		log.mu.Lock()
		direct := log.direct
		log.mu.Unlock()
		if direct != 0 {
			t.Fatal("dispatch langsung teramati")
		}
	}
}
