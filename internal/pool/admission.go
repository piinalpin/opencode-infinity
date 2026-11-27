// Package pool memiliki reservasi kapasitas request dan relay.
package pool

import (
	"errors"
)

var (
	ErrCapacityExceeded   = errors.New("gateway_capacity_exceeded")
	ErrNoEligibleRelay    = errors.New("no_eligible_relay")
	ErrInvalidReservation = errors.New("invalid_reservation")
)

// Admit memilih dari mapping server dan mereservasi seluruh kapasitas secara atomik.
// ID hasil adalah satu-satunya relay yang boleh dipakai caller untuk dispatch.
func (r *Reservation) Admit(upstream, alias string, streamBytes int64) (string, error) {
	if r == nil || r.owner == nil {
		return "", ErrInvalidReservation
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released || r.admitted || streamBytes < 0 {
		return "", ErrInvalidReservation
	}
	now := a.clock()
	var model *ModelHealthView
	for i := range a.view.Models {
		m := &a.view.Models[i]
		if m.Upstream == upstream && m.Alias == alias {
			model = m
			break
		}
	}
	if model == nil || a.upstreamCoolingLocked(upstream, alias, now) || (fresh(model.Availability, now) && model.Availability.Circuit == CircuitOpen) {
		return "", ErrNoEligibleRelay
	}
	candidates := []string{model.PrimaryRelay}
	if model.Routing == "least_inflight" {
		if !model.Stateless {
			return "", ErrNoEligibleRelay
		}
		candidates = a.candidates[upstream]
	}
	relay := ""
	eligible := false
	for _, id := range candidates {
		if !a.transportEligibleLocked(id, now) {
			continue
		}
		cap, ok := a.relays[id]
		if !ok {
			continue
		}
		eligible = true
		if a.used[id] >= cap {
			continue
		}
		if relay == "" || a.used[id] < a.used[relay] {
			relay = id
		}
	}
	if relay == "" {
		if eligible {
			return "", ErrCapacityExceeded
		}
		return "", ErrNoEligibleRelay
	}
	if a.active >= a.maxActive || a.outbound >= a.maxOutbound || streamBytes > a.maxStreamBytes-a.streamBytes {
		return "", ErrCapacityExceeded
	}
	a.pending--
	a.active++
	a.used[relay]++
	a.outbound++
	a.streamBytes += streamBytes
	r.admitted = true
	r.relay = relay
	r.streamBytes = streamBytes
	return relay, nil
}

// SwitchFallback atomically moves an admitted reservation to one configured eligible fallback relay.
func (r *Reservation) SwitchFallback(upstream, alias, relay string) error {
	if r == nil || r.owner == nil || upstream == "" || alias == "" || relay == "" {
		return ErrInvalidReservation
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released || !r.admitted || r.relay == "" {
		return ErrInvalidReservation
	}
	configured := false
	for _, model := range a.view.Models {
		if model.Upstream != upstream || model.Alias != alias {
			continue
		}
		for _, candidate := range model.fallbackRelays {
			if candidate == relay {
				configured = true
				break
			}
		}
		break
	}
	if !configured {
		return ErrNoEligibleRelay
	}
	if relay == r.relay || !a.transportEligibleLocked(relay, a.clock()) {
		return ErrNoEligibleRelay
	}
	cap, ok := a.relays[relay]
	if !ok {
		return ErrNoEligibleRelay
	}
	if a.used[relay] >= cap {
		return ErrCapacityExceeded
	}
	a.used[r.relay]--
	a.used[relay]++
	r.relay = relay
	return nil
}

type Reservation struct {
	owner       *Manager
	bytes       int64
	released    bool
	admitted    bool
	relay       string
	streamBytes int64
	binding     bindingKey
	hasBinding  bool
}

// AdmitLocal mengaktifkan pekerjaan katalog tanpa slot relay/outbound/stream.
func (r *Reservation) AdmitLocal() error {
	if r == nil || r.owner == nil {
		return ErrInvalidReservation
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released || r.admitted {
		return ErrInvalidReservation
	}
	if a.active >= a.maxActive {
		return ErrCapacityExceeded
	}
	a.pending--
	a.active++
	r.admitted = true
	return nil
}

// ReserveStream untuk SSE yang baru diketahui setelah header; caller wajib sudah di-admit.
// Panggil sebelum membuat buffer forwarding. Stream deklaratif memakai Admit.
func (r *Reservation) ReserveStream(bytes int64) bool {
	if r == nil || r.owner == nil {
		return false
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released || !r.admitted || r.relay == "" || r.streamBytes != 0 || bytes <= 0 || bytes > a.maxStreamBytes-a.streamBytes {
		return false
	}
	a.streamBytes += bytes
	r.streamBytes = bytes
	return true
}

// Begin membatasi persiapan pending dan mereservasi byte sebelum alokasi/read.
// Lease pending maupun admitted wajib Release; tidak ada antrean menunggu kapasitas.
func (a *Manager) Begin(bytes int64) (*Reservation, bool) {
	if a == nil {
		return nil, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if bytes <= 0 || bytes > a.maxBytes-a.bytes || a.pending+a.active >= a.maxActive {
		return nil, false
	}
	a.pending++
	a.bytes += bytes
	return &Reservation{owner: a, bytes: bytes}, true
}

func (r *Reservation) Release() {
	if r == nil || r.owner == nil {
		return
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released {
		return
	}
	r.released = true
	if r.admitted {
		a.active--
		if r.hasBinding {
			if binding, ok := a.bindings[r.binding]; ok && binding.inflight > 0 {
				binding.inflight--
				a.bindings[r.binding] = binding
			}
			r.hasBinding = false
		}
	} else {
		a.pending--
	}
	a.bytes -= r.bytes
	a.streamBytes -= r.streamBytes
	if r.relay != "" {
		a.used[r.relay]--
		a.outbound--
	}
}
