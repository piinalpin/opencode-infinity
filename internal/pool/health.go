package pool

import (
	"errors"
	"opencode-infinity/internal/config"
	"slices"
	"strings"
	"sync"
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

type ObservationView struct {
	State                  HealthState
	ObservedAt, ValidUntil time.Time
}

type RelayHealthView struct {
	ID             string
	DefaultPrimary bool
	SOCKS5, HTTPS  ObservationView
}

type HealthView struct {
	Ready, Accepting           bool
	Revision                   uint64
	StateChangedAt, SnapshotAt time.Time
	DefaultUpstream            string
	Relays                     []RelayHealthView
}

// Health memiliki observasi diagnostik, bukan reservasi atau keputusan dispatch.
// shortcut: integrasi eligibility dengan Admission menunggu EP-03-S01/S02.
type Health struct {
	mu    sync.Mutex
	clock func() time.Time
	view  HealthView
}

func NewHealth(c *config.Config, clock func() time.Time) (*Health, error) {
	if !c.Validated() || clock == nil {
		return nil, errors.New("health_invalid_component")
	}
	owned := *c
	now := clock()
	if !healthTime(now) {
		return nil, errors.New("health_invalid_clock")
	}
	relays := owned.Relays()
	// Accounting konservatif untuk owner, ID dan dua observasi tetap per relay.
	// Ini batas alokasi state komponen, bukan pengukuran RSS atau default konfigurasi.
	if owned.Limits().StateBytes < 2048+512*int64(len(relays)) {
		return nil, errors.New("health_state_budget_exceeded")
	}
	primary := make(map[string]bool)
	for _, u := range owned.Upstreams() {
		if u.ID == owned.Server().DefaultUpstream {
			for _, m := range u.Models {
				primary[m.PrimaryRelay] = true
			}
		}
	}
	view := HealthView{Accepting: true, Revision: 1, StateChangedAt: now, DefaultUpstream: strings.Clone(owned.Server().DefaultUpstream)}
	for _, r := range relays {
		view.Relays = append(view.Relays, RelayHealthView{ID: strings.Clone(r.ID), DefaultPrimary: primary[r.ID]})
	}
	return &Health{clock: clock, view: view}, nil
}

func (h *Health) View() HealthView {
	if h == nil || h.clock == nil {
		return HealthView{}
	}
	h.mu.Lock()
	view := h.view
	view.Relays = slices.Clone(view.Relays)
	h.mu.Unlock()
	view.SnapshotAt = h.clock()
	for i := range view.Relays {
		relay := &view.Relays[i]
		for _, observation := range []*ObservationView{&relay.SOCKS5, &relay.HTTPS} {
			if observation.State != HealthUnknown && (!healthTime(view.SnapshotAt) || view.SnapshotAt.Before(observation.ObservedAt) || !view.SnapshotAt.Before(observation.ValidUntil)) {
				observation.State = HealthStale
			}
		}
		if view.Accepting && relay.DefaultPrimary && relay.SOCKS5.State == HealthHealthy && relay.HTTPS.State == HealthHealthy {
			view.Ready = true
		}
	}
	return view
}

func (h *Health) Observe(relay string, layer HealthLayer, observation ObservationView) error {
	if h == nil || h.clock == nil || layer > HealthHTTPS || (observation.State != HealthHealthy && observation.State != HealthUnhealthy) {
		return errors.New("health_invalid_observation")
	}
	now := h.clock()
	if !healthTime(now) || !healthTime(observation.ObservedAt) || !healthTime(observation.ValidUntil) || observation.ObservedAt.After(now) || !observation.ValidUntil.After(now) || !observation.ValidUntil.After(observation.ObservedAt) {
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
		if target.State == observation.State && target.ObservedAt.Equal(observation.ObservedAt) && target.ValidUntil.Equal(observation.ValidUntil) {
			return nil
		}
		if !observation.ObservedAt.After(target.ObservedAt) {
			return errors.New("health_observation_order")
		}
		*target = observation
		h.view.Revision++
		if now.After(h.view.StateChangedAt) {
			h.view.StateChangedAt = now
		}
		return nil
	}
	return errors.New("health_unknown_relay")
}

func healthTime(t time.Time) bool { return !t.IsZero() && t.Year() >= 1 && t.Year() <= 9999 }

// SetAccepting mencatat kondisi proses; tidak menjalankan drain atau mengubah Admission.
func (h *Health) SetAccepting(accepting bool) error {
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
