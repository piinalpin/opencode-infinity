// Package pool memiliki reservasi kapasitas request dan relay.
package pool

import (
	"errors"
	"opencode-infinity/internal/config"
	"sync"
)

// shortcut: kapasitas statis saja; eligibility health/cooldown/binding menunggu task pool/lifecycle.
type Admission struct {
	mu                                 sync.Mutex
	active, bytes, maxActive, maxBytes int64
	outbound, maxOutbound              int64
	relays                             map[string]int64
	used                               map[string]int64
	streamBytes, maxStreamBytes        int64
}

func (r *Reservation) Assign(relay string) bool {
	if r == nil || r.owner == nil {
		return false
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	cap, ok := a.relays[relay]
	if !ok || r.released || r.relay != "" || a.outbound >= a.maxOutbound || a.used[relay] >= cap {
		return false
	}
	a.used[relay]++
	a.outbound++
	r.relay = relay
	return true
}

type Reservation struct {
	owner       *Admission
	bytes       int64
	released    bool
	relay       string
	streamBytes int64
}

// ReserveStream dimiliki lease yang sama; panggil sebelum membuat buffer forwarding.
func (r *Reservation) ReserveStream(bytes int64) bool {
	if r == nil || r.owner == nil {
		return false
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released || r.streamBytes != 0 || bytes <= 0 || bytes > a.maxStreamBytes-a.streamBytes {
		return false
	}
	a.streamBytes += bytes
	r.streamBytes = bytes
	return true
}

func NewAdmission(c *config.Config) (*Admission, error) {
	if !c.Validated() {
		return nil, errors.New("admission_unvalidated_config")
	}
	a := &Admission{maxActive: c.Server().MaxActiveRequests, maxBytes: c.Limits().BodyBufferBytes, maxStreamBytes: c.Limits().StreamBufferBytes, maxOutbound: c.Limits().OutboundConnections, relays: make(map[string]int64), used: make(map[string]int64)}
	for _, r := range c.Relays() {
		a.relays[r.ID] = r.MaxInflight
	}
	return a, nil
}
func (a *Admission) Begin(bytes int64) (*Reservation, bool) {
	if a == nil {
		return nil, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if bytes <= 0 || bytes > a.maxBytes-a.bytes || a.active >= a.maxActive {
		return nil, false
	}
	a.active++
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
	a.active--
	a.bytes -= r.bytes
	a.streamBytes -= r.streamBytes
	if r.relay != "" {
		a.used[r.relay]--
		a.outbound--
	}
}
