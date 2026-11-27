package integration_test

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
	"opencode-infinity/internal/httpapi"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
	"opencode-infinity/internal/testutil"
)

type fixtureNoSendError struct{}

func (fixtureNoSendError) Error() string        { return "fixture_proven_no_send" }
func (fixtureNoSendError) RequestNotSent() bool { return true }

type refusalEvidenceDispatcher struct {
	clients map[string]*http.Client
	primary *testutil.SOCKS5
}

type failoverFixture struct {
	components                  *httpapi.Components
	upstream                    *testutil.Upstream
	primaryProxy, fallbackProxy *testutil.SOCKS5
	primaryDial, fallbackDial   *dialLog
}

func newFailoverFixture(t *testing.T, handler http.HandlerFunc, refusePrimary bool) *failoverFixture {
	t.Helper()
	upstream := ledgerUpstream(t, handler)
	primaryProxy := newProxy(t, "warp-1", upstream.Authority, upstream.Addr(), refusePrimary)
	fallbackProxy := newProxy(t, "warp-2", upstream.Authority, upstream.Addr(), false)
	primaryClient, primaryDial := controlClient(t, primaryProxy.Addr(), upstream.Authority, upstream.Addr(), upstream.Roots, false)
	fallbackClient, fallbackDial := controlClient(t, fallbackProxy.Addr(), upstream.Authority, upstream.Addr(), upstream.Roots, false)

	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.NewReplacer(
		"https://api.provider.example/v1", "https://"+upstream.Authority+"/v1",
		"socks5h://warp-1:1080", "socks5h://"+primaryProxy.Addr(),
		"  - id: warp-1\n    proxy_url: socks5h://warp-1:1080\n    max_inflight: 5\nupstreams:",
		"  - id: warp-1\n    proxy_url: socks5h://"+primaryProxy.Addr()+"\n    max_inflight: 5\n  - id: warp-2\n    proxy_url: socks5h://"+fallbackProxy.Addr()+"\n    max_inflight: 5\nupstreams:",
		"relays: [warp-1]",
		"relays: [warp-1, warp-2]",
		"fallback_relays: []",
		"fallback_relays: [warp-2]",
	).Replace(string(data))
	cfg, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{
			"INFINITY_GATEWAY_TOKEN": modelGatewayToken,
			"INFINITY_ADMIN_TOKEN":   modelAdminToken,
			"INFINITY_UPSTREAM_KEY":  modelPrimaryKey,
		}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal("config fallback fixture invalid", err)
	}
	dispatcher := &refusalEvidenceDispatcher{
		clients: map[string]*http.Client{"warp-1": primaryClient, "warp-2": fallbackClient},
		primary: primaryProxy,
	}
	components, err := httpapi.NewComponents(cfg, dispatcher, status.Build{}, time.Now, pool.Policy{Failures: 1, Successes: 1})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	for _, relay := range cfg.Relays() {
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			if err := components.Pool.Observe(relay.ID, layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
		}
	}
	return &failoverFixture{components, upstream, primaryProxy, fallbackProxy, primaryDial, fallbackDial}
}

func (d *refusalEvidenceDispatcher) Dispatch(r *http.Request, relay config.Relay) (*http.Response, error) {
	client := d.clients[relay.ID]
	if client == nil {
		return nil, errors.New("fixture_unknown_relay")
	}
	if r.GetBody != nil {
		return nil, errors.New("fixture_replay_enabled")
	}
	response, err := client.Do(r)
	if relay.ID == "warp-1" && err != nil {
		trace := d.primary.Snapshot()
		if len(trace.Connects) == 1 && trace.Connects[0].Refused && trace.Connects[0].TargetPeer == "" {
			return nil, fixtureNoSendError{}
		}
	}
	return response, err
}

func TestGatewayFallbackRequiresRefusalProof(t *testing.T) {
	fixture := newFailoverFixture(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"model":"provider-model-id"}` {
			t.Errorf("fallback body berubah: %s", body)
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}, true)
	response := modelCall(t, fixture.components.Models, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
	if response.Code != http.StatusOK || response.Body.String() != `{"ok":true}` {
		t.Fatalf("fallback response salah: status=%d body=%s", response.Code, response.Body.String())
	}
	trace := fixture.primaryProxy.Snapshot()
	if len(trace.Connects) != 1 || !trace.Connects[0].Refused || trace.Connects[0].TargetPeer != "" {
		t.Fatalf("primary tidak memberi bukti refusal sebelum tunnel: %+v", trace)
	}
	if len(fixture.fallbackProxy.Snapshot().Connects) != 1 || fixture.upstream.Snapshot().Total != 1 || len(fixture.upstream.Snapshot().Dispatches) != 1 {
		t.Fatal("fallback tidak menghasilkan tepat satu dispatch upstream")
	}
	if err := testutil.VerifyProxyPath(fixture.upstream.Snapshot().Dispatches, fixture.fallbackProxy.Snapshot()); err != nil {
		t.Fatal("dispatch fallback melewati relay lain")
	}
	for _, log := range []*dialLog{fixture.primaryDial, fixture.fallbackDial} {
		log.mu.Lock()
		direct := log.direct
		log.mu.Unlock()
		if direct != 0 {
			t.Fatal("fallback memakai direct dial")
		}
	}
}

func TestGatewayAmbiguousDispatchDoesNotFallback(t *testing.T) {
	dropped := make(chan error, 1)
	fixture := newFailoverFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		dropped <- testutil.DropConnection(w)
	}, false)
	response := modelCall(t, fixture.components.Models, "POST", "/v1/chat/completions", `{"model":"coding-default"}`)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("dispatch ambigu harus menjadi 502, mendapat %d", response.Code)
	}
	if err := <-dropped; err != nil {
		t.Fatal("fixture gagal memutus response")
	}
	ledger := fixture.upstream.Snapshot()
	if ledger.Total != 1 || len(ledger.Dispatches) != 1 || !ledger.Dispatches[0].BodyRead || len(fixture.primaryProxy.Snapshot().Connects) != 1 || len(fixture.fallbackProxy.Snapshot().Connects) != 0 {
		t.Fatal("dispatch ambigu memicu fallback atau dispatch kedua")
	}
	for _, log := range []*dialLog{fixture.primaryDial, fixture.fallbackDial} {
		log.mu.Lock()
		direct := log.direct
		log.mu.Unlock()
		if direct != 0 {
			t.Fatal("dispatch ambigu memakai direct dial")
		}
	}
}

var _ gateway.Dispatcher = (*refusalEvidenceDispatcher)(nil)
