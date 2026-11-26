package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHealthAdminAndReadiness(t *testing.T) {
	now := time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)
	reads := 0
	h, state, err := NewHealthAPI(apiConfig(t), status.Build{}, func() time.Time { reads++; return now })
	if err != nil || h == nil || state == nil {
		t.Fatal("handler health privat tidak tersedia")
	}
	for _, path := range []string{"/health/live", "/health/ready"} {
		for _, token := range []string{"", "wrong", gatewayToken, upstreamKey, "duplicate"} {
			r := httptest.NewRequest("GET", path, nil)
			r.Header.Set("Authorization", "Bearer "+token)
			if token == "duplicate" {
				r.Header.Add("Authorization", "Bearer "+adminToken)
			}
			body := &unreadBody{}
			r.Body = body
			before := reads
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			assertGatewayError(t, w, 401, "invalid_admin_token")
			if body.reads != 0 || reads != before || w.Header().Get("WWW-Authenticate") != `Bearer realm="admin"` {
				t.Fatal("guard admin tidak mendahului body/state")
			}
		}
	}
	check := func(path string, want int, ready bool) {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer "+adminToken)
		r.Header.Set("X-Request-ID", "private-marker")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		var envelope struct {
			Health status.HealthSnapshot  `json:"health"`
			ID     string                 `json:"request_id"`
			Error  *struct{ Code string } `json:"error"`
		}
		if w.Code != want || json.Unmarshal(w.Body.Bytes(), &envelope) != nil || !envelope.Health.Live || envelope.Health.Ready != ready || envelope.ID != w.Header().Get("X-Request-ID") || len(envelope.ID) != 36 || w.Header().Get("X-Infinity-Error-Source") != "gateway" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("health status/snapshot/metadata salah")
		}
		if want == 503 && (envelope.Error == nil || envelope.Error.Code != "gateway_not_ready") || want == 200 && envelope.Error != nil {
			t.Fatal("not-ready tidak mempunyai envelope gateway yang tepat")
		}
	}
	check("/health/live", 200, false)
	check("/health/ready", 503, false)
	for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
		if err := state.Observe("warp-1", layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	check("/health/ready", 200, true)
	if err := state.SetAccepting(false); err != nil {
		t.Fatal(err)
	}
	check("/health/ready", 503, false)
	check("/health/live", 200, false)
}

func TestHealthRouteErrors(t *testing.T) {
	h, _, err := NewHealthAPI(apiConfig(t), status.Build{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path, code string
		status             int
	}{
		{"POST", "/health/live", "method_not_allowed", 405},
		{"OPTIONS", "/health/ready", "method_not_allowed", 405},
		{"GET", "/private-marker?token=private-marker", "endpoint_not_found", 404},
		{"GET", "/health//live?token=private-marker", "invalid_request", 400},
		{"GET", "/health/live/", "endpoint_not_found", 404},
	} {
		for _, token := range []string{adminToken, gatewayToken} {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want, code := tc.status, tc.code
			if token != adminToken {
				want, code = 401, "invalid_admin_token"
			}
			assertGatewayError(t, w, want, code)
			if want == 405 && w.Header().Get("Allow") != "GET, HEAD" {
				t.Fatal("health 405 tanpa Allow GET, HEAD")
			}
		}
	}
}

func TestHealthHTTPBudgets(t *testing.T) {
	for _, edits := range [][]string{
		{"snapshot_bytes: 262144", "snapshot_bytes: 1"},
		{"snapshot_bytes: 262144", "snapshot_bytes: 4096", "monitoring_bytes: 8388608", "monitoring_bytes: 16384"},
	} {
		if h, owner, err := NewHealthAPI(apiConfig(t, edits...), status.Build{}, time.Now); err == nil || h != nil || owner != nil {
			t.Fatal("health menerima budget snapshot/monitoring tidak cukup")
		}
	}
}

type healthBlockedWriter struct {
	header           http.Header
	entered, proceed chan struct{}
}

func (w *healthBlockedWriter) Header() http.Header { return w.header }
func (*healthBlockedWriter) WriteHeader(int)       {}
func (w *healthBlockedWriter) Write(p []byte) (int, error) {
	close(w.entered)
	<-w.proceed
	return 0, errors.New("synthetic-private-write-error")
}

func TestHealthParallelAndSlowWriter(t *testing.T) {
	h, owner, err := NewHealthAPI(apiConfig(t, "parallel_status: 4", "parallel_status: 1"), status.Build{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	request := func() *http.Request {
		r := httptest.NewRequest("GET", "/health/live", nil)
		r.Header.Set("Authorization", "Bearer "+adminToken)
		return r
	}
	w := &healthBlockedWriter{header: make(http.Header), entered: make(chan struct{}), proceed: make(chan struct{})}
	release := sync.OnceFunc(func() { close(w.proceed) })
	defer release()
	done := make(chan struct{})
	go func() { defer close(done); h.ServeHTTP(w, request()) }()
	select {
	case <-w.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("writer barrier tidak tercapai")
	}
	updated := make(chan error, 1)
	go func() { updated <- owner.SetAccepting(false) }()
	select {
	case err := <-updated:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("slow writer menahan lock owner")
	}
	second := httptest.NewRecorder()
	h.ServeHTTP(second, request())
	assertGatewayError(t, second, 429, "gateway_capacity_exceeded")
	release()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler tidak selesai setelah write error")
	}
	third := httptest.NewRecorder()
	h.ServeHTTP(third, request())
	if third.Code != 200 {
		t.Fatal("slot monitoring tidak pulih sesudah write error")
	}
}

func TestHealthSerializationFailureReleases(t *testing.T) {
	now := time.Now()
	h, _, err := NewHealthAPI(apiConfig(t, "parallel_status: 4", "parallel_status: 1"), status.Build{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/health/live", nil)
	r.Header.Set("Authorization", "Bearer "+adminToken)
	now = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assertGatewayError(t, w, 503, "health_snapshot_unavailable")
	now = time.Now()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal("serialize failure membocorkan slot monitoring")
	}
}

func TestHealthHandlerIsolation(t *testing.T) {
	health, _, err := NewHealthAPI(apiConfig(t), status.Build{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	model, err := NewAPI(apiConfig(t), modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		t.Fatal("route health melakukan dispatch model")
		return nil, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		handler           http.Handler
		path, token, code string
		status            int
	}{
		{model, "/health/live", gatewayToken, "endpoint_not_found", 404},
		{model, "/health/ready", adminToken, "invalid_gateway_token", 401},
		{health, "/v1/models", adminToken, "endpoint_not_found", 404},
		{health, "/v1/models", gatewayToken, "invalid_admin_token", 401},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Authorization", "Bearer "+tc.token)
		w := httptest.NewRecorder()
		tc.handler.ServeHTTP(w, r)
		assertGatewayError(t, w, tc.status, tc.code)
	}
}

func TestHealthMaximumRegistrySnapshot(t *testing.T) {
	var extra strings.Builder
	extra.WriteString("relays:\n")
	for i := 1; i < 128; i++ {
		fmt.Fprintf(&extra, "  - id: relay-%03d-%s\n    proxy_url: socks5h://fixture:1080\n    max_inflight: 1\n", i, strings.Repeat("x", 54))
	}
	c := apiConfig(t, "relays:\n", extra.String())
	now := time.Now()
	h, owner, err := NewHealthAPI(c, status.Build{Version: "v1.2.3+" + strings.Repeat("a", 57), Revision: strings.Repeat("a", 64)}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	for _, relay := range c.Relays() {
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			if err := owner.Observe(relay.ID, layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}); err != nil {
				t.Fatal(err)
			}
		}
	}
	r := httptest.NewRequest("GET", "/health/ready", nil)
	r.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var envelope struct {
		Health status.HealthSnapshot `json:"health"`
	}
	if w.Code != 200 || int64(w.Body.Len()) > c.Limits().SnapshotBytes || json.Unmarshal(w.Body.Bytes(), &envelope) != nil || len(envelope.Health.Relays) != 128 {
		t.Fatal("snapshot registry maksimum tidak bounded/utuh")
	}
}
