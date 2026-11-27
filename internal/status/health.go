// Package status membentuk proyeksi teredaksi dari salinan state server.
package status

import (
	"encoding/hex"
	"errors"
	"opencode-infinity/internal/pool"
	"regexp"
	"time"
)

type Build struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
}

type Health struct{ build Build }

func NewHealth(build Build) (*Health, error) {
	if build.Version == "" {
		build.Version = "unknown"
	}
	if build.Revision == "" {
		build.Revision = "unknown"
	}
	if len(build.Version) > 64 {
		return nil, errors.New("health_invalid_build")
	}
	validVersion, _ := regexp.MatchString(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`, build.Version)
	if !validVersion && build.Version != "unknown" && build.Version != "devel" {
		return nil, errors.New("health_invalid_build")
	}
	if build.Revision != "unknown" {
		if len(build.Revision) != 40 && len(build.Revision) != 64 {
			return nil, errors.New("health_invalid_build")
		}
		if _, err := hex.DecodeString(build.Revision); err != nil {
			return nil, errors.New("health_invalid_build")
		}
	}
	return &Health{build: build}, nil
}

type HealthSnapshot struct {
	Scope           string        `json:"scope"`
	Live            bool          `json:"live"`
	Ready           bool          `json:"ready"`
	Accepting       bool          `json:"accepting"`
	Application     Build         `json:"application"`
	Revision        uint64        `json:"revision"`
	StateChangedAt  time.Time     `json:"state_changed_at"`
	SnapshotAt      time.Time     `json:"snapshot_at"`
	DefaultUpstream string        `json:"default_upstream"`
	UpstreamModel   string        `json:"upstream_model"`
	Relays          []RelayHealth `json:"relays"`
	Models          []ModelHealth `json:"models"`
}

type ModelHealth struct {
	Upstream     string      `json:"upstream"`
	Alias        string      `json:"alias"`
	PrimaryRelay string      `json:"primary_relay"`
	Availability Observation `json:"availability"`
}
type RelayHealth struct {
	Mode           string      `json:"mode"`
	ID             string      `json:"id"`
	DefaultPrimary bool        `json:"default_primary"`
	SOCKS5         Observation `json:"socks5"`
	HTTPS          Observation `json:"https"`
}

type Observation struct {
	Circuit    string     `json:"circuit"`
	State      string     `json:"state"`
	ObservedAt *time.Time `json:"observed_at"`
	ValidUntil *time.Time `json:"valid_until"`
}

func (h *Health) Snapshot(view pool.HealthView) HealthSnapshot {
	result := HealthSnapshot{Scope: "default_primary_transport", Live: view.Revision != 0, Ready: view.Ready, Accepting: view.Accepting, Application: h.build, Revision: view.Revision, StateChangedAt: view.StateChangedAt.UTC(), SnapshotAt: view.SnapshotAt.UTC(), DefaultUpstream: view.DefaultUpstream, UpstreamModel: "unknown", Relays: make([]RelayHealth, 0, len(view.Relays))}
	for _, r := range view.Relays {
		result.Relays = append(result.Relays, RelayHealth{ID: r.ID, Mode: r.Mode, DefaultPrimary: r.DefaultPrimary, SOCKS5: observation(r.SOCKS5), HTTPS: observation(r.HTTPS)})
	}
	for _, m := range view.Models {
		result.Models = append(result.Models, ModelHealth{Upstream: m.Upstream, Alias: m.Alias, PrimaryRelay: m.PrimaryRelay, Availability: observation(m.Availability)})
	}
	return result
}

func observation(source pool.ObservationView) Observation {
	result := Observation{State: "unknown", Circuit: "checking"}
	switch source.Circuit {
	case pool.CircuitClosed:
		result.Circuit = "closed"
	case pool.CircuitOpen:
		result.Circuit = "open"
	}
	switch source.State {
	case pool.HealthHealthy:
		result.State = "healthy"
	case pool.HealthUnhealthy:
		result.State = "unhealthy"
	case pool.HealthStale:
		result.State = "stale"
	}
	if !source.ObservedAt.IsZero() {
		at := source.ObservedAt.UTC()
		result.ObservedAt = &at
	}
	if !source.ValidUntil.IsZero() {
		until := source.ValidUntil.UTC()
		result.ValidUntil = &until
	}
	return result
}
