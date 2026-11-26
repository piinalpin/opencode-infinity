package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"opencode-infinity/internal/httpapi"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
)

func TestHealthWireAndNoDispatch(t *testing.T) {
	var paid atomic.Int32
	var hold atomic.Bool
	entered, proceed := make(chan struct{}, 1), make(chan struct{})
	release := sync.OnceFunc(func() { close(proceed) })
	defer release()
	h := newModelHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/fixture-health" {
			w.WriteHeader(204)
			return
		}
		paid.Add(1)
		if hold.Load() {
			entered <- struct{}{}
			<-proceed
		}
		_, _ = io.WriteString(w, `{}`)
	}, func(http.ResponseWriter, *http.Request) { t.Error("health menghubungi provider sekunder") }, "max_active_requests: 20", "max_active_requests: 1")
	c := modelConfiguration(t, "https://api.provider.example/v1", "https://"+h.primary.Authority+"/v1", "socks5h://warp-1:1080", "socks5h://"+h.a.Addr())
	var instant atomic.Int64
	instant.Store(time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC).UnixNano())
	clock := func() time.Time { return time.Unix(0, instant.Load()).UTC() }
	api, owner, err := httpapi.NewHealthAPI(c, status.Build{}, clock)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api)
	defer server.Close()
	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	call := func(method, path, token string) *http.Response {
		t.Helper()
		r, err := http.NewRequestWithContext(t.Context(), method, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		r.Header.Set("X-Request-ID", "private-marker")
		r.Header.Set("Cookie", "token="+modelAdminToken)
		response, err := client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	read := func(method, path string, code int, ready bool) status.HealthSnapshot {
		t.Helper()
		response := call(method, path, modelAdminToken)
		data := readErrorResponse(t, response)
		if response.StatusCode != code || len(response.Header.Get("X-Request-ID")) != 36 || response.Header.Get("X-Infinity-Error-Source") != "gateway" || response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("health wire status/ID/source/cache salah")
		}
		if method == "HEAD" {
			if len(data) != 0 {
				t.Fatal("HEAD health mengirim body")
			}
			return status.HealthSnapshot{}
		}
		var envelope struct {
			Health status.HealthSnapshot  `json:"health"`
			ID     string                 `json:"request_id"`
			Error  *struct{ Code string } `json:"error"`
		}
		if json.Unmarshal(data, &envelope) != nil || !envelope.Health.Live || envelope.Health.Ready != ready || envelope.Health.Scope != "default_primary_transport" || envelope.Health.Application.Version != "unknown" || envelope.Health.UpstreamModel != "unknown" || envelope.ID != response.Header.Get("X-Request-ID") {
			t.Fatal("snapshot health wire salah")
		}
		if code == 503 && (envelope.Error == nil || envelope.Error.Code != "gateway_not_ready") {
			t.Fatal("not-ready kehilangan envelope error")
		}
		for _, secret := range []string{modelGatewayToken, modelAdminToken, modelPrimaryKey, modelSecondaryKey, "upstream.invalid", "private-marker", "socks5h://"} {
			if bytes.Contains(data, []byte(secret)) {
				t.Fatal("metadata health membocorkan secret/target")
			}
		}
		return envelope.Health
	}
	for _, path := range []string{"/health/live", "/health/ready"} {
		for _, method := range []string{"GET", "HEAD"} {
			for _, token := range []string{"", "wrong", modelGatewayToken, modelPrimaryKey} {
				assertWireGatewayError(t, call(method, path+"?token="+modelAdminToken, token), 401, "invalid_admin_token")
			}
		}
	}
	for _, tc := range []struct {
		method, path, code string
		status             int
	}{
		{"POST", "/health/live", "method_not_allowed", 405},
		{"GET", "/missing?token=private-marker", "endpoint_not_found", 404},
		{"GET", "/health//ready?token=private-marker", "invalid_request", 400},
	} {
		assertWireGatewayError(t, call(tc.method, tc.path, modelAdminToken), tc.status, tc.code)
	}
	read("GET", "/health/live", 200, false)
	read("GET", "/health/ready", 503, false)
	read("HEAD", "/health/live", 200, false)
	read("HEAD", "/health/ready", 503, false)
	if h.a.Snapshot().Accepted != 0 || h.b.Snapshot().Accepted != 0 || h.primary.Snapshot().Total != 0 || paid.Load() != 0 {
		t.Fatal("constructor/health read memicu jaringan atau generasi")
	}
	// Observasi dikumpulkan eksplisit oleh fixture; endpoint health tidak menjalankannya.
	conn, err := net.DialTimeout("tcp", h.a.Addr(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	_, err = conn.Write([]byte{5, 1, 0})
	var greeting [2]byte
	_, readErr := io.ReadFull(conn, greeting[:])
	conn.Close()
	if err != nil || readErr != nil || greeting != [2]byte{5, 0} {
		t.Fatal("observasi SOCKS5 fixture gagal")
	}
	at := clock()
	if err := owner.Observe("warp-1", pool.HealthSOCKS5, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	read("GET", "/health/ready", 503, false)
	probe, err := h.clients["warp-1"].Get("https://" + h.primary.Authority + "/fixture-health")
	if err != nil {
		t.Fatal(err)
	}
	probe.Body.Close()
	if probe.StatusCode != 204 {
		t.Fatal("observasi HTTPS fixture gagal")
	}
	if err := owner.Observe("warp-1", pool.HealthHTTPS, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	ready := read("GET", "/health/ready", 200, true)
	beforeConnect, beforeUpstream := h.a.Snapshot().Accepted, h.primary.Snapshot().Total
	for range 10 {
		read("GET", "/health/live", 200, true)
		read("GET", "/health/%72eady", 200, true)
		read("HEAD", "/health/ready", 200, true)
	}
	if h.a.Snapshot().Accepted != beforeConnect || h.primary.Snapshot().Total != beforeUpstream || paid.Load() != 0 {
		t.Fatal("health read mengulang probe/generasi")
	}
	// Request generasi memegang satu-satunya lease; read health tidak mengambil/melepasnya.
	hold.Store(true)
	done := make(chan int, 1)
	go func() {
		done <- modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`).Code
	}()
	await(t, entered)
	read("GET", "/health/live", 200, true)
	read("GET", "/health/ready", 200, true)
	if w := modelCall(t, h.handler, "POST", "/v1/chat/completions", `{"model":"coding-default"}`); w.Code != 429 {
		t.Fatal("health melepas lease generation aktif")
	}
	release()
	if await(t, done) != 200 || paid.Load() != 1 {
		t.Fatal("health mengubah dispatch aktif")
	}
	h.verifyPath(t)
	if err := h.a.Close(); err != nil {
		t.Fatal(err)
	}
	if conn, err := net.DialTimeout("tcp", h.a.Addr(), time.Second); err == nil {
		conn.Close()
		t.Fatal("relay fixture masih tersedia")
	}
	instant.Add(int64(time.Second))
	at = clock()
	if err := owner.Observe("warp-1", pool.HealthSOCKS5, pool.ObservationView{State: pool.HealthUnhealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	failed := read("GET", "/health/ready", 503, false)
	read("GET", "/health/live", 200, false)
	if failed.Revision <= ready.Revision || failed.Relays[0].HTTPS.State != "healthy" {
		t.Fatal("failure tidak terlacak atau mengubah lapisan lain")
	}
	instant.Add(int64(2 * time.Minute))
	expired := read("GET", "/health/ready", 503, false)
	if expired.Revision != failed.Revision || !expired.StateChangedAt.Equal(failed.StateChangedAt) || expired.Relays[0].HTTPS.State != "stale" || !expired.SnapshotAt.After(failed.SnapshotAt) {
		t.Fatal("expiry mengganti revision/observasi atau tidak terlacak")
	}
}
