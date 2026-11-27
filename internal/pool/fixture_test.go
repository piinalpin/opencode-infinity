package pool

import (
	"opencode-infinity/internal/config"
	"strings"
	"testing"
	"time"
)

func newDiagnosticPool(c *config.Config, clock func() time.Time) (*Manager, error) {
	return New(c, clock, Policy{Failures: 1, Successes: 1})
}
func newReadyPool(c *config.Config) (*Manager, error) {
	h, err := New(c, time.Now, Policy{Failures: 1, Successes: 1})
	if err != nil {
		return nil, err
	}
	at := time.Now()
	for _, r := range c.Relays() {
		for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
			if err := h.Observe(r.ID, layer, ObservationView{State: HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Hour)}); err != nil {
				return nil, err
			}
		}
	}
	return h, nil
}
func capacityConfig(t *testing.T, edits ...string) *config.Config {
	// Fixture kapasitas dua relay memerlukan dua mapping server yang sah.
	if strings.Contains(strings.Join(edits, ""), "id: warp-2") {
		edits = append(edits, "relays: [warp-1]", "relays: [warp-1, warp-2]", "    models:\n", "    models:\n      - alias: coding-warp-2\n        upstream_model: fixture-secondary\n        primary_relay: warp-2\n        fallback_relays: []\n")
	}
	return admissionConfig(t, edits...)
}
func admitFixtureRelay(r *Reservation, relay string, bytes int64) error {
	alias := "missing"
	if relay == "warp-1" {
		alias = "coding-default"
	}
	if relay == "warp-2" {
		alias = "coding-warp-2"
	}
	return r.AdmitFixed("primary", alias, bytes)
}
