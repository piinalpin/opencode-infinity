// Package config memuat satu snapshot konfigurasi sebelum dipakai aplikasi.
package config

import (
	"io"
	"slices"
)

type Sources struct {
	LookupEnv func(string) (string, bool)
	OpenFile  func(string) (io.ReadCloser, error)
}
type Server struct {
	Listen            string `yaml:"listen"`
	AdminListen       string `yaml:"admin_listen"`
	DefaultUpstream   string `yaml:"default_upstream"`
	MaxBodyBytes      int64  `yaml:"max_body_bytes"`
	MaxActiveRequests int64  `yaml:"max_active_requests"`
}
type Limits struct {
	BodyBufferBytes     int64 `yaml:"body_buffer_bytes"`
	StreamBufferBytes   int64 `yaml:"stream_buffer_bytes"`
	StateBytes          int64 `yaml:"state_bytes"`
	MonitoringBytes     int64 `yaml:"monitoring_bytes"`
	InboundConnections  int64 `yaml:"inbound_connections"`
	OutboundConnections int64 `yaml:"outbound_connections"`
	Bindings            int64 `yaml:"bindings"`
	AdminSessions       int64 `yaml:"admin_sessions"`
	ParallelProbes      int64 `yaml:"parallel_probes"`
	ParallelStatus      int64 `yaml:"parallel_status"`
	SnapshotBytes       int64 `yaml:"snapshot_bytes"`
}
type Timeouts struct {
	ConnectSeconds         int64 `yaml:"connect_seconds"`
	ResponseHeadersSeconds int64 `yaml:"response_headers_seconds"`
	StreamIdleSeconds      int64 `yaml:"stream_idle_seconds"`
	TotalRequestSeconds    int64 `yaml:"total_request_seconds"`
}
type Relay struct {
	ID          string `yaml:"id"`
	ProxyURL    string `yaml:"proxy_url"`
	MaxInflight int64  `yaml:"max_inflight"`
}
type Model struct {
	Stateless      bool     `yaml:"stateless"`
	Alias          string   `yaml:"alias"`
	UpstreamModel  string   `yaml:"upstream_model"`
	Routing        string   `yaml:"routing"`
	PrimaryRelay   string   `yaml:"primary_relay"`
	FallbackRelays []string `yaml:"fallback_relays"`
	Affinity       string   `yaml:"affinity"`
}
type Upstream struct {
	ID      string   `yaml:"id"`
	BaseURL string   `yaml:"base_url"`
	Relays  []string `yaml:"relays"`
	Models  []Model  `yaml:"models"`
}
type Config struct {
	validated      bool
	gateway, admin Secret
	keys           map[string]Secret
	server         Server
	limits         Limits
	timeouts       Timeouts
	relays         []Relay
	upstreams      []Upstream
}

func (c *Config) Server() Server     { return c.server }
func (c *Config) Limits() Limits     { return c.limits }
func (c *Config) Timeouts() Timeouts { return c.timeouts }
func (c *Config) Relays() []Relay    { return slices.Clone(c.relays) }
func (c *Config) Upstreams() []Upstream {
	out := slices.Clone(c.upstreams)
	for i := range out {
		out[i].Relays = slices.Clone(out[i].Relays)
		out[i].Models = slices.Clone(out[i].Models)
		for j := range out[i].Models {
			out[i].Models[j].FallbackRelays = slices.Clone(out[i].Models[j].FallbackRelays)
		}
	}
	return out
}

func (c *Config) Validated() bool { return c != nil && c.validated }
