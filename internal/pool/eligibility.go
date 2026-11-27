package pool

import (
	"errors"
)

// SetMode tidak mengubah hasil probe atau membatalkan lease yang sudah admitted.
func (a *Manager) SetMode(relay, mode string) error {
	if a == nil || a.clock == nil {
		return errors.New("health_invalid_component")
	}
	if mode != "serving" && mode != "draining" && mode != "disabled" {
		return errors.New("health_invalid_mode")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if !healthTime(now) {
		return errors.New("health_invalid_clock")
	}
	for i := range a.view.Relays {
		r := &a.view.Relays[i]
		if r.ID == relay {
			if r.Mode != mode {
				r.Mode = mode
				a.changed(now)
			}
			return nil
		}
	}
	return errors.New("health_unknown_relay")
}

// ObserveModel tidak menganggap transport sehat sebagai bukti availability model.
// shortcut: sumber availability provider runtime menunggu integrasi; unknown tidak memblokir transport sehat.
func (a *Manager) ObserveModel(upstream, alias string, observation ObservationView) error {
	if a == nil || a.clock == nil {
		return errors.New("health_invalid_component")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if !validObservation(observation, now) {
		return errors.New("health_invalid_observation")
	}
	for i := range a.view.Models {
		m := &a.view.Models[i]
		if m.Upstream == upstream && m.Alias == alias {
			return a.updateObservation(&m.Availability, observation, now)
		}
	}
	return errors.New("health_unknown_model")
}

// TryProbe membatasi probe dan generasi dengan ledger outbound yang sama.
// Caller wajib menutup jaringan sebelum release, termasuk sesudah timeout/cancel.
func (a *Manager) TryProbe(relay string, layer HealthLayer) (func(), bool) {
	if a == nil || layer > HealthHTTPS {
		return nil, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	target, ok := a.probes[relay]
	if !ok || target[layer] || a.probeCount >= a.maxProbes || a.outbound >= a.maxOutbound {
		return nil, false
	}
	target[layer] = true
	a.probes[relay] = target
	a.probeCount++
	a.outbound++
	released := false
	return func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if released {
			return
		}
		released = true
		target := a.probes[relay]
		target[layer] = false
		a.probes[relay] = target
		a.probeCount--
		a.outbound--
	}, true
}

func (a *Manager) ProbeConcurrency() int {
	if a == nil {
		return 0
	}
	return int(min(a.maxProbes, int64(2*len(a.relays))))
}
