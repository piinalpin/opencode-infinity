// Package catalog menyediakan lookup registry konfigurasi tervalidasi.
package catalog

import (
	"errors"
	"opencode-infinity/internal/config"
	"slices"
)

type Registry struct {
	defaultUpstream string
	upstreams       map[string]config.Upstream
	models          map[string]map[string]config.Model
	relays          map[string]config.Relay
}

func New(c *config.Config) (*Registry, error) {
	if c == nil || !c.Validated() {
		return nil, errors.New("catalog_unvalidated_config")
	}
	r := &Registry{defaultUpstream: c.Server().DefaultUpstream, upstreams: make(map[string]config.Upstream), models: make(map[string]map[string]config.Model), relays: make(map[string]config.Relay)}
	for _, u := range c.Upstreams() {
		r.upstreams[u.ID] = u
		r.models[u.ID] = make(map[string]config.Model, len(u.Models))
		for _, m := range u.Models {
			r.models[u.ID][m.Alias] = m
		}
	}
	for _, v := range c.Relays() {
		r.relays[v.ID] = v
	}
	return r, nil
}
func (r *Registry) DefaultUpstream() string { return r.defaultUpstream }
func (r *Registry) Resolve(upstream, alias string) (config.Upstream, config.Model, bool) {
	if upstream == "" {
		upstream = r.defaultUpstream
	}
	u, ok := r.upstreams[upstream]
	if !ok {
		return config.Upstream{}, config.Model{}, false
	}
	m, ok := r.models[upstream][alias]
	if !ok {
		return config.Upstream{}, config.Model{}, false
	}
	m.FallbackRelays = slices.Clone(m.FallbackRelays)
	return copyUpstream(u), m, true
}
func (r *Registry) Relay(id string) (config.Relay, bool) { v, ok := r.relays[id]; return v, ok }
func (r *Registry) Upstream(id string) (config.Upstream, bool) {
	u, ok := r.upstreams[id]
	return copyUpstream(u), ok
}
func copyUpstream(u config.Upstream) config.Upstream {
	u.Relays = slices.Clone(u.Relays)
	u.Models = slices.Clone(u.Models)
	for i := range u.Models {
		u.Models[i].FallbackRelays = slices.Clone(u.Models[i].FallbackRelays)
	}
	return u
}
