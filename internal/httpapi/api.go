package httpapi

import (
	"errors"
	"net/http"
	"opencode-infinity/internal/access"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
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
	}
	return RequireGateway(policy, mux)
}
