package gateway

import (
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/pool"
	"time"
)

func newFixtureCoordinator(c *config.Config, d Dispatcher) (*Coordinator, error) {
	g, h, err := New(c, d, time.Now, pool.Policy{Failures: 1, Successes: 1})
	if err != nil {
		return nil, err
	}
	at := time.Now()
	for _, r := range c.Relays() {
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			if err := h.Observe(r.ID, layer, pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Hour)}); err != nil {
				return nil, err
			}
		}
	}
	return g, nil
}
