package httpapi

import (
	"errors"
	"net/http"
	"opencode-infinity/internal/access"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
	"strings"
	"time"
)

// Components berbagi owner dari satu snapshot server; constructor tidak memulai probe.
type Components struct {
	Models, Health http.Handler
	Pool           *pool.Manager
}

func NewComponents(c *config.Config, d gateway.Dispatcher, build status.Build, clock func() time.Time, healthPolicy pool.Policy) (*Components, error) {
	if !c.Validated() {
		return nil, errors.New("invalid_api_component")
	}
	owned := *c
	coordinator, owner, err := gateway.New(&owned, d, clock, healthPolicy)
	if err != nil {
		return nil, errors.New("invalid_api_component")
	}
	health, err := newHealthAPI(&owned, build, owner)
	if err != nil {
		return nil, errors.New("invalid_api_component")
	}
	models, err := newModelAPI(&owned, coordinator)
	if err != nil {
		return nil, err
	}
	return &Components{Models: models, Health: health, Pool: owner}, nil
}

func newModelAPI(c *config.Config, coordinator *gateway.Coordinator) (http.Handler, error) {
	owned := *c
	policy, err := access.New(&owned)
	if err != nil {
		return nil, errors.New("invalid_api_component")
	}
	mux := http.NewServeMux()
	handler := modelHandler{coordinator: coordinator, idle: time.Duration(owned.Timeouts().StreamIdleSeconds) * time.Second, total: time.Duration(owned.Timeouts().TotalRequestSeconds) * time.Second}
	for _, pattern := range []string{"GET /v1/models", "GET /providers/{provider}/v1/models", "POST /v1/chat/completions", "POST /providers/{provider}/v1/chat/completions"} {
		mux.Handle(pattern, handler)
		method, route, _ := strings.Cut(pattern, " ")
		allow := method
		if method == http.MethodGet {
			allow = "GET, HEAD"
		}
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Allow", allow)
			writeModelError(w, newRequestID(), &gateway.Failure{Code: "method_not_allowed", Status: http.StatusMethodNotAllowed})
		})
	}
	for _, route := range []string{"/v1/responses", "/providers/{provider}/v1/responses"} {
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			writeModelError(w, newRequestID(), &gateway.Failure{Code: "unsupported_api", Status: http.StatusNotImplemented})
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeModelError(w, newRequestID(), &gateway.Failure{Code: "endpoint_not_found", Status: http.StatusNotFound})
	})
	return RequireGateway(policy, canonicalRoutes(mux))
}
