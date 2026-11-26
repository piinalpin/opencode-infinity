package httpapi

import (
	"errors"
	"net/http"
	"opencode-infinity/internal/access"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
	"strings"
	"time"
)

// NewAPI merakit komponen HTTP; tidak membuka listener atau membuat transport default.
// shortcut: listener, batas koneksi inbound dan health runtime menunggu perakitan aplikasi.
func NewAPI(c *config.Config, d gateway.Dispatcher) (http.Handler, error) {
	if !c.Validated() {
		return nil, errors.New("invalid_api_component")
	}
	owned := *c
	coordinator, err := gateway.New(&owned, d)
	if err != nil {
		return nil, errors.New("invalid_api_component")
	}
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
