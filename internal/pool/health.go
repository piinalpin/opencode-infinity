package pool

import (
	"errors"
	"slices"
	"time"
)

type HealthState uint8

const (
	HealthUnknown HealthState = iota
	HealthHealthy
	HealthUnhealthy
	HealthStale
)

type HealthLayer uint8

const (
	HealthSOCKS5 HealthLayer = iota
	HealthHTTPS
)

type CircuitState uint8

const (
	CircuitChecking CircuitState = iota
	CircuitClosed
	CircuitOpen
)

type ObservationView struct {
	Circuit                CircuitState
	successes, failures    uint32
	State                  HealthState
	ObservedAt, ValidUntil time.Time
}

type RelayHealthView struct {
	CooldownUntil  time.Time
	CooldownActive bool
	ID             string
	Mode           string
	DefaultPrimary bool
	SOCKS5, HTTPS  ObservationView
}

type HealthView struct {
	Ready, Accepting           bool
	Revision                   uint64
	StateChangedAt, SnapshotAt time.Time
	DefaultUpstream            string
	Relays                     []RelayHealthView
	Models                     []ModelHealthView
}

func (h *Manager) View() HealthView {
	if h == nil || h.clock == nil {
		return HealthView{}
	}
	h.mu.Lock()
	view := h.view
	view.Relays = slices.Clone(view.Relays)
	view.Models = slices.Clone(view.Models)
	h.mu.Unlock()
	view.SnapshotAt = h.clock()
	for i := range view.Relays {
		relay := &view.Relays[i]
		relay.CooldownActive = healthTime(view.SnapshotAt) && view.SnapshotAt.Before(relay.CooldownUntil)
		for _, observation := range []*ObservationView{&relay.SOCKS5, &relay.HTTPS} {
			if observation.State != HealthUnknown && (!healthTime(view.SnapshotAt) || view.SnapshotAt.Before(observation.ObservedAt) || !view.SnapshotAt.Before(observation.ValidUntil)) {
				observation.State = HealthStale
			}
		}
		if view.Accepting && relay.Mode == "serving" && relay.DefaultPrimary && relay.SOCKS5.Circuit == CircuitClosed && relay.HTTPS.Circuit == CircuitClosed && relay.SOCKS5.State != HealthStale && relay.HTTPS.State != HealthStale {
			view.Ready = true
		}
	}
	for i := range view.Models {
		if !fresh(view.Models[i].Availability, view.SnapshotAt) {
			view.Models[i].Availability.State = HealthUnknown
			view.Models[i].Availability.Circuit = CircuitChecking
		}
	}
	return view
}

func (h *Manager) Observe(relay string, layer HealthLayer, observation ObservationView) error {
	if h == nil || h.clock == nil || layer > HealthHTTPS || (observation.State != HealthHealthy && observation.State != HealthUnhealthy) {
		return errors.New("health_invalid_observation")
	}
	now := h.clock()
	if !validObservation(observation, now) {
		return errors.New("health_invalid_observation")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.view.Relays {
		if h.view.Relays[i].ID != relay {
			continue
		}
		target := &h.view.Relays[i].SOCKS5
		if layer == HealthHTTPS {
			target = &h.view.Relays[i].HTTPS
		}
		return h.updateObservation(target, observation, now)
	}
	return errors.New("health_unknown_relay")
}

func healthTime(t time.Time) bool { return !t.IsZero() && t.Year() >= 1 && t.Year() <= 9999 }

// SetAccepting mengendalikan admission baru tanpa membatalkan request aktif.
func (h *Manager) SetAccepting(accepting bool) error {
	if h == nil || h.clock == nil {
		return errors.New("health_invalid_component")
	}
	now := h.clock()
	if !healthTime(now) {
		return errors.New("health_invalid_clock")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.view.Accepting != accepting {
		h.view.Accepting = accepting
		h.view.Revision++
		if now.After(h.view.StateChangedAt) {
			h.view.StateChangedAt = now
		}
	}
	return nil
}

func advanceObservation(target *ObservationView, observation ObservationView, now time.Time, policy Policy) {
	circuit, successes, failures := target.Circuit, target.successes, target.failures
	if !fresh(*target, now) {
		circuit = CircuitChecking
		successes = 0
		if target.State != HealthUnhealthy {
			failures = 0
		}
	}
	if observation.State == HealthHealthy {
		failures = 0
		if successes < policy.Successes {
			successes++
		}
		if successes >= policy.Successes {
			circuit = CircuitClosed
		}
	} else {
		successes = 0
		if failures < policy.Failures {
			failures++
		}
		if failures >= policy.Failures {
			circuit = CircuitOpen
		}
	}
	*target = ObservationView{State: observation.State, ObservedAt: observation.ObservedAt, ValidUntil: observation.ValidUntil, Circuit: circuit, successes: successes, failures: failures}
}

func (a *Manager) changed(now time.Time) {
	a.view.Revision++
	if now.After(a.view.StateChangedAt) {
		a.view.StateChangedAt = now
	}
}

func validObservation(o ObservationView, now time.Time) bool {
	return (o.State == HealthHealthy || o.State == HealthUnhealthy) && healthTime(now) && healthTime(o.ObservedAt) && healthTime(o.ValidUntil) && !o.ObservedAt.After(now) && o.ValidUntil.After(now) && o.ValidUntil.After(o.ObservedAt)
}
func (a *Manager) updateObservation(target *ObservationView, o ObservationView, now time.Time) error {
	if target.State == o.State && target.ObservedAt.Equal(o.ObservedAt) && target.ValidUntil.Equal(o.ValidUntil) {
		return nil
	}
	if !o.ObservedAt.After(target.ObservedAt) {
		return errors.New("health_observation_order")
	}
	advanceObservation(target, o, now, a.policy)
	a.changed(now)
	return nil
}
