package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"opencode-infinity/internal/health"
	"opencode-infinity/internal/httpapi"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
)

// Adapter hanya untuk fixture: target berasal dari snapshot server, bukan request client.
type eligibilityProber struct{ h *modelHarness }

func (p eligibilityProber) Probe(ctx context.Context, relay string, layer pool.HealthLayer) (pool.HealthState, error) {
	var proxy, target string
	for _, r := range p.h.config.Relays() {
		if r.ID == relay {
			u, _ := url.Parse(r.ProxyURL)
			proxy = u.Host
		}
	}
	for _, u := range p.h.config.Upstreams() {
		for _, id := range u.Relays {
			if id == relay {
				target = u.BaseURL
			}
		}
	}
	if proxy == "" || target == "" {
		return pool.HealthUnhealthy, errors.New("fixture_unknown_target")
	}
	if layer == pool.HealthSOCKS5 {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", proxy)
		if err != nil {
			return pool.HealthUnhealthy, err
		}
		defer conn.Close()
		stop := context.AfterFunc(ctx, func() { conn.Close() })
		defer stop()
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
			return pool.HealthUnhealthy, err
		}
		var greeting [2]byte
		if _, err := io.ReadFull(conn, greeting[:]); err != nil {
			return pool.HealthUnhealthy, err
		}
		if greeting != [2]byte{5, 0} {
			return pool.HealthUnhealthy, errors.New("fixture_bad_socks5")
		}
		return pool.HealthHealthy, nil
	}
	request, err := http.NewRequestWithContext(ctx, "GET", target+"/fixture-health", nil)
	if err != nil {
		return pool.HealthUnhealthy, err
	}
	response, err := p.h.clients[relay].Do(request)
	if err != nil {
		return pool.HealthUnhealthy, err
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return pool.HealthUnhealthy, errors.New("fixture_https_unavailable")
	}
	return pool.HealthHealthy, nil
}

func waitEligibility(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for !condition() {
		select {
		case <-deadline.C:
			t.Fatal("state eligibility fixture melewati deadline")
		case <-tick.C:
		}
	}
}

func TestEligibilityScheduledWireAndDrain(t *testing.T) {
	var primaryPosts, secondaryPosts atomic.Int32
	var hold atomic.Bool
	entered := make(chan struct{}, 1)
	unblock := make(chan struct{})
	release := sync.OnceFunc(func() { close(unblock) })
	defer release()
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/fixture-health") {
			w.WriteHeader(204)
			return
		}
		primaryPosts.Add(1)
		if hold.Load() {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"held\":true}\n\n")
			_ = http.NewResponseController(w).Flush()
			entered <- struct{}{}
			select {
			case <-unblock:
			case <-r.Context().Done():
				return
			}
			_, _ = io.WriteString(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}
	h := newModelHarness(t, handler, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/fixture-health") {
			w.WriteHeader(204)
			return
		}
		secondaryPosts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})
	// Satu lease mewakili socket yang benar-benar ditutup, tanpa idle keepalive.
	for _, client := range h.clients {
		client.Transport.(*http.Transport).DisableKeepAlives = true
	}
	components, err := httpapi.NewComponents(h.config, &modelFixtureDispatcher{clients: h.clients}, status.Build{}, time.Now, pool.Policy{Failures: 3, Successes: 2})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(components.Models)
	defer server.Close()
	admin := httptest.NewServer(components.Health)
	defer admin.Close()
	call := func(path string) *http.Response {
		t.Helper()
		response, err := server.Client().Do(gatewayStreamRequest(t, server, path, `{"model":"coding-default","stream":true}`))
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	checkNoNetwork := func(action func()) {
		t.Helper()
		a, b, p, s := h.a.Snapshot(), h.b.Snapshot(), h.primary.Snapshot(), h.secondary.Snapshot()
		action()
		aa, bb := h.a.Snapshot(), h.b.Snapshot()
		if aa.Accepted != a.Accepted || bb.Accepted != b.Accepted || len(aa.Connects) != len(a.Connects) || len(bb.Connects) != len(b.Connects) || h.primary.Snapshot().Total != p.Total || h.secondary.Snapshot().Total != s.Total {
			t.Fatal("request ditolak/read menambah koneksi atau dispatch")
		}
	}
	checkNoNetwork(func() {
		for _, path := range []string{"/v1/chat/completions", "/providers/secondary/v1/chat/completions"} {
			assertWireGatewayError(t, call(path), 503, "no_eligible_relay")
		}
	})
	scheduler, err := health.New(components.Pool, health.Schedule{Interval: 20 * time.Millisecond, Timeout: time.Second, Freshness: 3 * time.Second, MaxBackoff: 100 * time.Millisecond}, eligibilityProber{h}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	waitEligibility(t, func() bool {
		v := components.Pool.View()
		for _, r := range v.Relays {
			if r.SOCKS5.Circuit != pool.CircuitClosed || r.HTTPS.Circuit != pool.CircuitClosed {
				return false
			}
		}
		return v.Ready
	})
	cancel()
	if err := await(t, done); err != nil {
		t.Fatal(err)
	}
	waitEligibility(t, func() bool { return h.a.Snapshot().Active == 0 && h.b.Snapshot().Active == 0 })
	if primaryPosts.Load() != 0 || secondaryPosts.Load() != 0 {
		t.Fatal("probe menjadwalkan generasi")
	}
	checkNoNetwork(func() {
		for range 5 {
			r, err := http.NewRequest("GET", admin.URL+"/health/ready", nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Authorization", "Bearer "+modelAdminToken)
			response, err := admin.Client().Do(r)
			if err != nil {
				t.Fatal(err)
			}
			var result struct{ Health status.HealthSnapshot }
			if json.NewDecoder(response.Body).Decode(&result) != nil {
				t.Fatal("snapshot invalid")
			}
			response.Body.Close()
			if response.StatusCode != 200 || !result.Health.Ready {
				t.Fatal("transport terjadwal belum ready")
			}
			for _, m := range result.Health.Models {
				if m.Availability.State != "unknown" {
					t.Fatal("probe transport mengarang availability model")
				}
			}
		}
	})
	for _, relay := range []string{"warp-1", "warp-2"} {
		if err := components.Pool.SetMode(relay, "disabled"); err != nil {
			t.Fatal(err)
		}
	}
	checkNoNetwork(func() {
		for _, path := range []string{"/v1/chat/completions", "/providers/secondary/v1/chat/completions"} {
			assertWireGatewayError(t, call(path), 503, "no_eligible_relay")
		}
	})
	// Probe sukses setelah disabled tetap tidak mengubah mode operator.
	for _, relay := range []string{"warp-1", "warp-2"} {
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			at := time.Now()
			if err := components.Pool.Observe(relay, layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
				t.Fatal(err)
			}
		}
	}
	checkNoNetwork(func() { assertWireGatewayError(t, call("/v1/chat/completions"), 503, "no_eligible_relay") })
	if err := components.Pool.SetMode("warp-1", "serving"); err != nil {
		t.Fatal(err)
	}
	hold.Store(true)
	first := call("/v1/chat/completions")
	defer first.Body.Close()
	await(t, entered)
	if err := components.Pool.SetMode("warp-1", "draining"); err != nil {
		t.Fatal(err)
	}
	checkNoNetwork(func() { assertWireGatewayError(t, call("/v1/chat/completions"), 503, "no_eligible_relay") })
	release()
	data, err := io.ReadAll(first.Body)
	first.Body.Close()
	if err != nil || !strings.Contains(string(data), "[DONE]") {
		t.Fatal("drain memutus stream admitted")
	}
	if primaryPosts.Load() != 1 || secondaryPosts.Load() != 0 {
		t.Fatal("jumlah generasi salah")
	}
	h.verifyPath(t)
}

func TestEligibilityProbeOutboundAndModelScope(t *testing.T) {
	var blocked atomic.Bool
	blocked.Store(true)
	entered := make(chan struct{}, 1)
	h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			entered <- struct{}{}
			if blocked.Load() {
				<-r.Context().Done()
				return
			}
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}, "outbound_connections: 32", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1")
	for _, client := range h.clients {
		client.Transport.(*http.Transport).DisableKeepAlives = true
	}
	components, err := httpapi.NewComponents(h.config, &modelFixtureDispatcher{clients: h.clients}, status.Build{}, time.Now, pool.Policy{Failures: 3, Successes: 2})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		for _, relay := range []string{"warp-1", "warp-2"} {
			for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
				at := time.Now()
				if err := components.Pool.Observe(relay, layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	scheduler, err := health.New(components.Pool, health.Schedule{Interval: 20 * time.Millisecond, Timeout: time.Second, Freshness: 3 * time.Second, MaxBackoff: 100 * time.Millisecond}, eligibilityProber{h}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	await(t, entered)
	server := httptest.NewServer(components.Models)
	defer server.Close()
	call := func(path string) *http.Response {
		t.Helper()
		response, err := server.Client().Do(gatewayStreamRequest(t, server, path, `{"model":"coding-default"}`))
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	beforeA, beforeB := h.a.Snapshot().Accepted, h.b.Snapshot().Accepted
	beforeP, beforeS := h.primary.Snapshot().Total, h.secondary.Snapshot().Total
	assertWireGatewayError(t, call("/v1/chat/completions"), 429, "gateway_capacity_exceeded")
	if h.a.Snapshot().Accepted != beforeA || h.b.Snapshot().Accepted != beforeB || h.primary.Snapshot().Total != beforeP || h.secondary.Snapshot().Total != beforeS {
		t.Fatal("probe tidak membatasi socket generasi melalui cap bersama")
	}
	cancel()
	if err := await(t, done); err != nil {
		t.Fatal(err)
	}
	waitEligibility(t, func() bool { return h.a.Snapshot().Active == 0 && h.b.Snapshot().Active == 0 })
	for range 3 {
		at := time.Now()
		if err := components.Pool.ObserveModel("primary", "coding-default", pool.ObservationView{State: pool.HealthUnhealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	assertWireGatewayError(t, call("/v1/chat/completions"), 503, "no_eligible_relay")
	response := call("/providers/secondary/v1/chat/completions")
	_, err = io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 {
		t.Fatal("kegagalan model primary menutup scope secondary atau slot probe bocor")
	}
	if !components.Pool.View().Ready {
		t.Fatal("availability model merusak readiness transport")
	}
	h.verifyPath(t)
}
