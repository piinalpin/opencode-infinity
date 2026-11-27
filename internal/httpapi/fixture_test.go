package httpapi

import (
	"errors"
	"net/http"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
	"time"
)

// Helper tes lama menyatakan transport fixture sehat; constructor produk tetap unknown.
func fixtureModelAPI(c *config.Config, d gateway.Dispatcher) (http.Handler, error) {
	components, err := NewComponents(c, d, status.Build{}, time.Now, pool.Policy{Failures: 1, Successes: 1})
	if err != nil {
		return nil, err
	}
	at := time.Now()
	for _, r := range c.Relays() {
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			if err := components.Pool.Observe(r.ID, layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Hour)}); err != nil {
				return nil, err
			}
		}
	}
	return components.Models, nil
}
func fixtureHealthAPI(c *config.Config, b status.Build, clock func() time.Time) (http.Handler, *pool.Manager, error) {
	components, err := NewComponents(c, modelDispatch(func(*http.Request, config.Relay) (*http.Response, error) {
		return nil, errors.New("fixture_no_generation")
	}), b, clock, pool.Policy{Failures: 1, Successes: 1})
	if err != nil {
		return nil, nil, err
	}
	return components.Health, components.Pool, nil
}
