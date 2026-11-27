package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"opencode-infinity/internal/access"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
	"sync"
)

// NewHealthAPI merakit handler operasional terpisah, tanpa dispatcher atau probe.
// shortcut: pasang pada listener privat saat runtime dirakit; guard bukan bukti binding jaringan.
func newHealthAPI(c *config.Config, build status.Build, state *pool.Manager) (http.Handler, error) {
	if !c.Validated() {
		return nil, errors.New("health_invalid_component")
	}
	owned := *c
	limits := owned.Limits()
	// Batas konservatif schema tetap: envelope/build/time dan tiap relay (ID <=64).
	// Tiga kali bound mencakup salinan view/proyeksi serta buffer encode/body.
	bound := int64(4096 + 2048*len(owned.Relays()) + 1024*len(state.View().Models))
	// Worker probe memakai scratch bounded; perkalian view aman karena registry finite.
	workerBytes := int64(state.ProbeConcurrency()) * 4096
	if limits.SnapshotBytes < bound || limits.ParallelStatus > (limits.MonitoringBytes-workerBytes)/(3*bound) {
		return nil, errors.New("health_invalid_component")
	}
	projection, err := status.NewHealth(build)
	if err != nil {
		return nil, errors.New("health_invalid_component")
	}
	policy, _ := access.New(&owned)
	h := &healthHandler{state: state, projection: projection, maxSnapshot: limits.SnapshotBytes, maxParallel: limits.ParallelStatus}
	mux := http.NewServeMux()
	for _, route := range []string{"/health/live", "/health/ready"} {
		mux.Handle("GET "+route, h)
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Allow", "GET, HEAD")
			writeModelError(w, newRequestID(), &gateway.Failure{Code: "method_not_allowed", Status: 405})
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeModelError(w, newRequestID(), &gateway.Failure{Code: "endpoint_not_found", Status: 404})
	})
	guard, _ := RequireAdmin(policy, canonicalRoutes(mux))
	return guard, nil
}

type healthHandler struct {
	state                            *pool.Manager
	projection                       *status.Health
	mu                               sync.Mutex
	active, maxParallel, maxSnapshot int64
}

type healthError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if h.active >= h.maxParallel {
		h.mu.Unlock()
		writeModelError(w, newRequestID(), &gateway.Failure{Code: "gateway_capacity_exceeded", Status: 429})
		return
	}
	h.active++
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.active--
		h.mu.Unlock()
	}()
	id := newRequestID()
	snapshot := h.projection.Snapshot(h.state.View())
	result := struct {
		Health    status.HealthSnapshot `json:"health"`
		Error     *healthError          `json:"error,omitempty"`
		RequestID string                `json:"request_id"`
	}{Health: snapshot, RequestID: id}
	code := http.StatusOK
	if r.URL.Path == "/health/ready" && !snapshot.Ready {
		code = http.StatusServiceUnavailable
		result.Error = &healthError{Message: "Gateway belum siap.", Type: "gateway_unavailable", Code: "gateway_not_ready"}
	}
	data, err := json.Marshal(result)
	if err != nil || int64(len(data)) > h.maxSnapshot {
		writeModelError(w, id, &gateway.Failure{Status: 503, Code: "health_snapshot_unavailable"})
		return
	}
	w.Header().Set("X-Request-ID", id)
	w.Header().Set("X-Infinity-Error-Source", "gateway")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	if r.Method != http.MethodHead {
		_, _ = w.Write(data)
	}
}
