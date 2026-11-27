package pool

import (
	"errors"
	"opencode-infinity/internal/config"
	"sync"
	"time"
)

// Policy menetapkan ambang komponen secara eksplisit, tanpa default produk.
type Policy struct{ Failures, Successes uint32 }

type ModelHealthView struct {
	Upstream, Alias, PrimaryRelay string
	Availability                  ObservationView
}

// Manager memiliki keputusan eligibility dan seluruh reservasi di bawah satu mutex.
// shortcut: cooldown, fallback dan binding menunggu child S03–S05 serta EP-04.
type Manager struct {
	probes                             map[string][2]bool
	probeCount, maxProbes              int64
	mu                                 sync.Mutex
	clock                              func() time.Time
	policy                             Policy
	view                               HealthView
	active, bytes, maxActive, maxBytes int64
	pending                            int64
	outbound, maxOutbound              int64
	relays, used                       map[string]int64
	streamBytes, maxStreamBytes        int64
}

func New(c *config.Config, clock func() time.Time, policy Policy) (*Manager, error) {
	if !c.Validated() || clock == nil || policy.Failures == 0 || policy.Successes == 0 {
		return nil, errors.New("health_invalid_component")
	}
	now := clock()
	if !healthTime(now) {
		return nil, errors.New("health_invalid_clock")
	}
	relays := c.Relays()
	models := int64(0)
	for _, u := range c.Upstreams() {
		models += int64(len(u.Models))
	}
	// Bound konservatif owner, dua observasi/circuit dan job per relay, serta scope per model.
	// Bukan pengukuran RSS; registry sudah dibatasi loader sehingga perkalian tidak overflow.
	if c.Limits().StateBytes < 4096+1024*int64(len(relays))+1024*models {
		return nil, errors.New("health_state_budget_exceeded")
	}
	workers := min(c.Limits().ParallelProbes, int64(2*len(relays)))
	if workers > c.Limits().MonitoringBytes/4096 {
		return nil, errors.New("health_monitoring_budget_exceeded")
	}
	a := &Manager{probes: make(map[string][2]bool), maxProbes: c.Limits().ParallelProbes, clock: clock, policy: policy, maxActive: c.Server().MaxActiveRequests, maxBytes: c.Limits().BodyBufferBytes, maxStreamBytes: c.Limits().StreamBufferBytes, maxOutbound: c.Limits().OutboundConnections, relays: make(map[string]int64), used: make(map[string]int64)}
	a.view = HealthView{Accepting: true, Revision: 1, StateChangedAt: now, DefaultUpstream: c.Server().DefaultUpstream}
	primary := map[string]bool{}
	for _, u := range c.Upstreams() {
		for _, m := range u.Models {
			a.view.Models = append(a.view.Models, ModelHealthView{Upstream: u.ID, Alias: m.Alias, PrimaryRelay: m.PrimaryRelay})
			if u.ID == c.Server().DefaultUpstream {
				primary[m.PrimaryRelay] = true
			}
		}
	}
	for _, r := range relays {
		a.relays[r.ID] = r.MaxInflight
		a.probes[r.ID] = [2]bool{}
		a.view.Relays = append(a.view.Relays, RelayHealthView{ID: r.ID, Mode: "serving", DefaultPrimary: primary[r.ID]})
	}
	return a, nil
}

func (a *Manager) transportEligibleLocked(relay string, now time.Time) bool {
	if !a.view.Accepting || !healthTime(now) {
		return false
	}
	for _, r := range a.view.Relays {
		if r.ID == relay {
			return r.Mode == "serving" && fresh(r.SOCKS5, now) && fresh(r.HTTPS, now) && r.SOCKS5.Circuit == CircuitClosed && r.HTTPS.Circuit == CircuitClosed
		}
	}
	return false
}
func fresh(o ObservationView, now time.Time) bool {
	return healthTime(now) && !o.ObservedAt.IsZero() && !now.Before(o.ObservedAt) && now.Before(o.ValidUntil)
}
