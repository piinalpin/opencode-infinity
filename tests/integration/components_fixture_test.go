package integration_test

import (
	"errors"
	"net/http"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/gateway"
	"opencode-infinity/internal/httpapi"
	"opencode-infinity/internal/pool"
	"opencode-infinity/internal/status"
	"time"
)

func fixtureAPI(c *config.Config, d gateway.Dispatcher) (http.Handler, error) {
	components, err := httpapi.NewComponents(c, d, status.Build{}, time.Now, pool.Policy{Failures: 1, Successes: 1})
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

type noGeneration struct{}

func (noGeneration) Dispatch(*http.Request, config.Relay) (*http.Response, error) {
	return nil, errors.New("fixture_no_generation")
}
func fixtureHealthAPI(c *config.Config, b status.Build, clock func() time.Time) (http.Handler, *pool.Manager, error) {
	components, err := httpapi.NewComponents(c, noGeneration{}, b, clock, pool.Policy{Failures: 1, Successes: 1})
	if err != nil {
		return nil, nil, err
	}
	return components.Health, components.Pool, nil
}
