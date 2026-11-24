package config

import (
	"errors"
	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/loaderlimits"
	"io"
)

type rawServer struct {
	Server      `yaml:",inline"`
	GatewayEnv  string `yaml:"gateway_token_env"`
	GatewayFile string `yaml:"gateway_token_file"`
	AdminEnv    string `yaml:"admin_token_env"`
	AdminFile   string `yaml:"admin_token_file"`
}
type rawUpstream struct {
	Upstream `yaml:",inline"`
	KeyEnv   string `yaml:"api_key_env"`
	KeyFile  string `yaml:"api_key_file"`
}
type document struct {
	Version   int           `yaml:"version"`
	Server    rawServer     `yaml:"server"`
	Limits    Limits        `yaml:"limits"`
	Timeouts  Timeouts      `yaml:"timeouts"`
	Relays    []Relay       `yaml:"relays"`
	Upstreams []rawUpstream `yaml:"upstreams"`
}

// shortcut: loader offline; pasang sebelum startup listener pada task perakitan aplikasi.
func Load(r io.Reader, s Sources) (*Config, error) {
	if r == nil {
		return nil, errors.New("config_read_failed")
	}
	data, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil {
		return nil, errors.New("config_read_failed")
	}
	if len(data) > 1<<20 {
		return nil, errors.New("config_too_large")
	}
	var d document
	var root yaml.Node
	if err := yaml.Load(data, &root, yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthValue(32)))); err != nil || root.Kind != yaml.DocumentNode || len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode || !schemaNodes(root.Content[0], 0, new(int)) {
		return nil, errors.New("config_invalid_yaml")
	}
	if err := yaml.Load(data, &d, yaml.WithKnownFields(true), yaml.WithUniqueKeys(true), yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthValue(32)))); err != nil {
		return nil, errors.New("config_invalid_yaml")
	}
	if d.Version != 1 {
		return nil, errors.New("config_invalid_version")
	}
	if err := validateDocument(&d); err != nil {
		return nil, err
	}
	if err := validateTargets(&d); err != nil {
		return nil, err
	}
	if err := validateLimits(&d); err != nil {
		return nil, err
	}
	c := &Config{server: d.Server.Server, limits: d.Limits, timeouts: d.Timeouts, relays: d.Relays}
	for _, u := range d.Upstreams {
		c.upstreams = append(c.upstreams, u.Upstream)
	}
	if err := loadSecrets(&d, c, s); err != nil {
		return nil, err
	}
	c.validated = true
	return c, nil
}

func schemaNodes(n *yaml.Node, depth int, count *int) bool {
	*count++
	if depth > 32 || *count > 20000 || n.Anchor != "" || n.Kind == yaml.AliasNode {
		return false
	}
	switch n.Kind {
	case yaml.ScalarNode:
		if (n.Tag != "!!str" && n.Tag != "!!int") || n.Value == "" || len(n.Value) > 4096 {
			return false
		}
	case yaml.MappingNode:
		if n.Tag != "!!map" {
			return false
		}
		for i := 0; i < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" {
				return false
			}
			if v.Kind == yaml.ScalarNode {
				expected := "!!str"
				if numericField(k.Value) {
					expected = "!!int"
				}
				if v.Tag != expected {
					return false
				}
			}
		}
	case yaml.SequenceNode:
		if n.Tag != "!!seq" {
			return false
		}
		for _, v := range n.Content {
			if v.Kind == yaml.ScalarNode && v.Tag != "!!str" {
				return false
			}
		}
	default:
		return false
	}
	for _, v := range n.Content {
		if !schemaNodes(v, depth+1, count) {
			return false
		}
	}
	return true
}

func numericField(key string) bool {
	switch key {
	case "version", "max_body_bytes", "max_active_requests", "body_buffer_bytes", "stream_buffer_bytes", "state_bytes", "monitoring_bytes", "inbound_connections", "outbound_connections", "bindings", "admin_sessions", "parallel_probes", "parallel_status", "snapshot_bytes", "connect_seconds", "response_headers_seconds", "stream_idle_seconds", "total_request_seconds", "max_inflight":
		return true
	}
	return false
}
