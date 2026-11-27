package pool

import (
	"errors"
	"opencode-infinity/internal/config"
	"sync"
	"time"
)

// Policy menetapkan ambang komponen secara eksplisit, tanpa default produk.
type Policy struct {
	Failures, Successes uint32
	BindingTTL          time.Duration
}

type ModelHealthView struct {
	Routing                       string
	Stateless                     bool
	Affinity                      string
	Upstream, Alias, PrimaryRelay string
	Availability                  ObservationView
	fallbackRelays                []string
}

// Manager memiliki keputusan eligibility dan seluruh reservasi di bawah satu mutex.
// shortcut: generation invalidation menunggu EP-04-S03; provider-specific scope evidence menunggu integrasi provider.
type Manager struct {
	candidates                         map[string][]string
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
	upstreamCooldowns                  map[string]time.Time
	modelCooldowns                     map[string]time.Time
	lastUpstreamErrors                 map[string]upstreamError
	bindings                           map[bindingKey]affinityBinding
	generations                        map[string]uint64
	maxBindings                        int64
	bindingTTL                         time.Duration
}

// upstreamError contains bounded, redacted classification metadata only.
type upstreamError struct {
	Status   int
	Category string
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
	models, references := int64(0), int64(0)
	for _, u := range c.Upstreams() {
		models += int64(len(u.Models))
		references += int64(len(u.Relays))
		for _, model := range u.Models {
			if model.Affinity == "strict" && policy.BindingTTL <= 0 {
				return nil, errors.New("affinity_policy_invalid")
			}
			references += int64(len(model.FallbackRelays))
		}
	}
	// Bound konservatif owner, cooldown/error metadata per upstream/model, observasi/circuit per relay, dan candidate references.
	// Bukan pengukuran RSS; registry sudah dibatasi loader sehingga perkalian tidak overflow.
	bindingState := int64(0)
	for _, upstream := range c.Upstreams() {
		for _, model := range upstream.Models {
			if model.Affinity == "strict" {
				bindingState = 512 * c.Limits().Bindings
				break
			}
		}
	}
	if c.Limits().StateBytes < 4096+1024*int64(len(relays))+128*int64(len(c.Upstreams()))+1024*models+64*references+bindingState {
		return nil, errors.New("health_state_budget_exceeded")
	}
	workers := min(c.Limits().ParallelProbes, int64(2*len(relays)))
	if workers > c.Limits().MonitoringBytes/4096 {
		return nil, errors.New("health_monitoring_budget_exceeded")
	}
	a := &Manager{candidates: make(map[string][]string), probes: make(map[string][2]bool), maxProbes: c.Limits().ParallelProbes, clock: clock, policy: policy, maxActive: c.Server().MaxActiveRequests, maxBytes: c.Limits().BodyBufferBytes, maxStreamBytes: c.Limits().StreamBufferBytes, maxOutbound: c.Limits().OutboundConnections, relays: make(map[string]int64), used: make(map[string]int64), upstreamCooldowns: make(map[string]time.Time), modelCooldowns: make(map[string]time.Time), lastUpstreamErrors: make(map[string]upstreamError), bindings: make(map[bindingKey]affinityBinding), generations: make(map[string]uint64), maxBindings: c.Limits().Bindings, bindingTTL: policy.BindingTTL}
	a.view = HealthView{Accepting: true, Revision: 1, StateChangedAt: now, DefaultUpstream: c.Server().DefaultUpstream}
	primary := map[string]bool{}
	for _, u := range c.Upstreams() {
		a.candidates[u.ID] = u.Relays
		for _, m := range u.Models {
			a.view.Models = append(a.view.Models, ModelHealthView{Routing: m.Routing, Stateless: m.Stateless, Affinity: m.Affinity, Upstream: u.ID, Alias: m.Alias, PrimaryRelay: m.PrimaryRelay, fallbackRelays: append([]string(nil), m.FallbackRelays...)})
			if u.ID == c.Server().DefaultUpstream {
				primary[m.PrimaryRelay] = true
			}
		}
	}
	for _, r := range relays {
		a.relays[r.ID] = r.MaxInflight
		a.generations[r.ID] = 1
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
			return r.Mode == "serving" && !now.Before(r.CooldownUntil) && fresh(r.SOCKS5, now) && fresh(r.HTTPS, now) && r.SOCKS5.Circuit == CircuitClosed && r.HTTPS.Circuit == CircuitClosed
		}
	}
	return false
}
func fresh(o ObservationView, now time.Time) bool {
	return healthTime(now) && !o.ObservedAt.IsZero() && !now.Before(o.ObservedAt) && now.Before(o.ValidUntil)
}
