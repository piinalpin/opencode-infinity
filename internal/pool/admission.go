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

// AdmitFixed menerima primary dari mapping server tervalidasi, tanpa fallback.
// Seluruh pemeriksaan mendahului satu commit gateway/relay/outbound/stream.
func (r *Reservation) AdmitFixed(upstream, alias string, streamBytes int64) error {
	if r == nil || r.owner == nil {
		return ErrInvalidReservation
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released || r.admitted || streamBytes < 0 {
		return ErrInvalidReservation
	}
	relay := ""
	for _, m := range a.view.Models {
		if m.Upstream == upstream && m.Alias == alias {
			relay = m.PrimaryRelay
			if fresh(m.Availability, a.clock()) && m.Availability.Circuit == CircuitOpen {
				return ErrNoEligibleRelay
			}
			break
		}
	}
	if !a.transportEligibleLocked(relay, a.clock()) {
		return ErrNoEligibleRelay
	}
	cap, ok := a.relays[relay]
	if !ok {
		return ErrNoEligibleRelay
	}
	if a.active >= a.maxActive || a.outbound >= a.maxOutbound || a.used[relay] >= cap || streamBytes > a.maxStreamBytes-a.streamBytes {
		return ErrCapacityExceeded
	}
	a.pending--
	a.active++
	a.used[relay]++
	a.outbound++
	a.streamBytes += streamBytes
	r.admitted = true
	r.relay = relay
	r.streamBytes = streamBytes
	return nil
}

type Reservation struct {
	owner       *Manager
	bytes       int64
	released    bool
	admitted    bool
	relay       string
	streamBytes int64
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
// Panggil sebelum membuat buffer forwarding. Stream deklaratif memakai AdmitFixed.
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
