package pool

import (
	"errors"
	"opencode-infinity/internal/access"
	"time"
)

var ErrAffinityLost = errors.New("affinity_lost")

type bindingKey struct {
	token               access.GatewayIdentity
	upstream, alias, id string
}

type affinityBinding struct {
	relay      string
	generation uint64
	lastUsed   time.Time
	inflight   int64
}

// InvalidateRelayGeneration invalidates existing bindings without interrupting
// requests that have already been admitted.
// shortcut: controlled identity-maintenance callers must invoke this explicitly; lifecycle wiring is outside S03 and belongs with maintenance integration.
func (a *Manager) InvalidateRelayGeneration(relay string) error {
	if a == nil || a.clock == nil {
		return errors.New("pool_invalid_component")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	generation, ok := a.generations[relay]
	if !ok {
		return errors.New("health_unknown_relay")
	}
	if generation == ^uint64(0) {
		return errors.New("pool_generation_exhausted")
	}
	a.generations[relay] = generation + 1
	return nil
}

// AdmitAffinity binds strict sessions and reserves their relay under the same owner lock.
func (r *Reservation) AdmitAffinity(upstream, alias string, streamBytes int64, token access.GatewayIdentity, sessionID string, start bool) (string, uint64, error) {
	if r == nil || r.owner == nil || token == (access.GatewayIdentity{}) || sessionID == "" || len(sessionID) > 256 {
		return "", 0, ErrInvalidReservation
	}
	a := r.owner
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.released || r.admitted || streamBytes < 0 {
		return "", 0, ErrInvalidReservation
	}
	now := a.clock()
	var model *ModelHealthView
	for i := range a.view.Models {
		candidate := &a.view.Models[i]
		if candidate.Upstream == upstream && candidate.Alias == alias {
			model = candidate
			break
		}
	}
	if model == nil || model.Affinity != "strict" || model.Routing != "fixed" ||
		a.upstreamCoolingLocked(upstream, alias, now) ||
		fresh(model.Availability, now) && model.Availability.Circuit == CircuitOpen {
		return "", 0, ErrInvalidReservation
	}
	a.cleanupExpiredBindingsLocked(now)
	key := bindingKey{token: token, upstream: upstream, alias: alias, id: sessionID}
	binding, exists := a.bindings[key]
	if start {
		if exists {
			return "", 0, ErrAffinityLost
		}
		if int64(len(a.bindings)) >= a.maxBindings {
			return "", 0, ErrCapacityExceeded
		}
		relay := model.PrimaryRelay
		generation := a.generations[relay]
		if generation == 0 {
			return "", 0, ErrNoEligibleRelay
		}
		if err := r.admitRelayLocked(relay, streamBytes, now); err != nil {
			return "", 0, err
		}
		a.bindings[key] = affinityBinding{relay: relay, generation: generation, lastUsed: now, inflight: 1}
		r.binding = key
		r.hasBinding = true
		return relay, generation, nil
	}
	if !exists || !now.Before(binding.lastUsed.Add(a.bindingTTL)) || binding.generation != a.generations[binding.relay] || !a.transportEligibleLocked(binding.relay, now) {
		return "", 0, ErrAffinityLost
	}
	if err := r.admitRelayLocked(binding.relay, streamBytes, now); err != nil {
		return "", 0, err
	}
	binding.lastUsed = now
	binding.inflight++
	a.bindings[key] = binding
	r.binding = key
	r.hasBinding = true
	return binding.relay, binding.generation, nil
}

func (r *Reservation) admitRelayLocked(relay string, streamBytes int64, now time.Time) error {
	a := r.owner
	if r.released || r.admitted || streamBytes < 0 || !a.transportEligibleLocked(relay, now) {
		return ErrNoEligibleRelay
	}
	capacity, ok := a.relays[relay]
	if !ok {
		return ErrNoEligibleRelay
	}
	if a.used[relay] >= capacity || a.active >= a.maxActive || a.outbound >= a.maxOutbound || streamBytes > a.maxStreamBytes-a.streamBytes {
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

func (a *Manager) cleanupExpiredBindingsLocked(now time.Time) {
	for key, binding := range a.bindings {
		if binding.inflight == 0 && !now.Before(binding.lastUsed.Add(a.bindingTTL)) {
			delete(a.bindings, key)
		}
	}
}
