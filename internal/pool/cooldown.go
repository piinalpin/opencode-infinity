package pool

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

// ExtendRelayCooldown hanya menerima deadline relay-wide dari sumber server tepercaya.
// Cooldown upstream ditangani terpisah lewat RecordUpstreamResponse.
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

// RecordUpstreamResponse stores redacted classification and applies only scoped
// upstream cooldown. scope must come from trusted server-side provider evidence;
// unknown scope is conservatively upstream-wide.
func (a *Manager) RecordUpstreamResponse(upstream, alias string, status int, retryAfter, scope string) error {
	if a == nil || a.clock == nil || upstream == "" || alias == "" || status < 400 || status > 599 {
		return errors.New("health_invalid_upstream_response")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.clock()
	if !healthTime(now) {
		return errors.New("health_invalid_clock")
	}
	knownModel := false
	for _, model := range a.view.Models {
		if model.Upstream == upstream && model.Alias == alias {
			knownModel = true
			break
		}
	}
	if !knownModel {
		return errors.New("health_unknown_model")
	}
	a.lastUpstreamErrors[modelKey(upstream, alias)] = upstreamError{Status: status, Category: upstreamErrorCategory(status)}
	if status != http.StatusTooManyRequests {
		return nil
	}
	until, valid, active := retryAfterDeadline(retryAfter, now)
	if !valid || !active {
		return nil
	}
	key := modelKey(upstream, alias)
	if scope == "model" {
		if until.After(a.modelCooldowns[key]) {
			a.modelCooldowns[key] = until
			a.changed(now)
		}
		return nil
	}
	if until.After(a.upstreamCooldowns[upstream]) {
		a.upstreamCooldowns[upstream] = until
		a.changed(now)
	}
	return nil
}

func modelKey(upstream, alias string) string { return upstream + "\x00" + alias }

func upstreamErrorCategory(status int) string {
	switch status {
	case 401:
		return "authentication"
	case 403:
		return "access"
	case 429:
		return "rate_limit"
	case 400, 404, 422:
		return "request"
	default:
		return "upstream"
	}
}

func retryAfterDeadline(value string, now time.Time) (time.Time, bool, bool) {
	if value == "" || len(value) > 128 {
		return time.Time{}, false, false
	}
	if at, err := http.ParseTime(value); err == nil {
		if !healthTime(at) {
			return time.Time{}, false, false
		}
		return at, true, at.After(now)
	}
	for _, b := range []byte(value) {
		if b < '0' || b > '9' {
			return time.Time{}, false, false
		}
	}
	seconds, err := strconv.ParseUint(value, 10, 63)
	if err != nil || seconds > uint64((time.Duration(1<<63-1))/time.Second) {
		return time.Time{}, false, false
	}
	until := now.Add(time.Duration(seconds) * time.Second)
	if !healthTime(until) {
		return time.Time{}, false, false
	}
	return until, true, until.After(now)
}

func (a *Manager) upstreamCoolingLocked(upstream, alias string, now time.Time) bool {
	return now.Before(a.upstreamCooldowns[upstream]) || now.Before(a.modelCooldowns[modelKey(upstream, alias)])
}
