package pool

import (
	"errors"
	"time"
)

// ExtendRelayCooldown hanya menerima deadline relay-wide dari sumber server tepercaya.
// shortcut: interpretasi 429/Retry-After dan scope upstream/credential menunggu S05.
func (a *Manager) ExtendRelayCooldown(relay string, until time.Time) error {
	if a == nil || a.clock == nil {
		return errors.New("health_invalid_component")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if !healthTime(now) {
		return errors.New("health_invalid_clock")
	}
	if !healthTime(until) || !until.After(now) {
		return errors.New("health_invalid_cooldown")
	}
	for i := range a.view.Relays {
		r := &a.view.Relays[i]
		if r.ID == relay {
			if until.After(r.CooldownUntil) {
				r.CooldownUntil = until
				a.changed(now)
			}
			return nil
		}
	}
	return errors.New("health_unknown_relay")
}
