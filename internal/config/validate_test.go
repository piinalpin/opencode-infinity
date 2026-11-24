package config

import (
	"fmt"
	"strings"
	"testing"
)

func withSecondUpstream(s, id string) string {
	start := strings.Index(s, "  - id: primary\n")
	return s + strings.Replace(s[start:], "id: primary", "id: "+id, 1)
}
func TestRegistryValidation(t *testing.T) {
	cases := map[string]string{
		"relay_missing":             strings.Replace(validYAML, "[warp-1]", "[missing]", 1),
		"relay_duplicate":           strings.Replace(validYAML, "upstreams:", "  - id: warp-1\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 5\nupstreams:", 1),
		"upstream_duplicate":        withSecondUpstream(validYAML, "primary"),
		"alias_duplicate":           validYAML + "      - alias: coding-default\n        upstream_model: another\n        primary_relay: warp-1\n",
		"primary_missing":           strings.Replace(validYAML, "primary_relay: warp-1", "primary_relay: missing", 1),
		"relay_reference_duplicate": strings.Replace(validYAML, "[warp-1]", "[warp-1, warp-1]", 1),
		"default_missing_multi":     strings.Replace(withSecondUpstream(validYAML, "secondary"), "  default_upstream: primary\n", "", 1),
		"default_unknown":           strings.Replace(validYAML, "default_upstream: primary", "default_upstream: missing", 1),
		"empty_allowlist":           strings.Replace(validYAML, "[warp-1]", "[]", 1),
		"empty_models":              strings.Replace(validYAML, "    models:", "    models: []\nunused_models:", 1),
		"invalid_id":                strings.Replace(validYAML, "id: primary", "id: 'wrong/id'", 1),
		"long_id":                   strings.Replace(validYAML, "id: primary", "id: "+strings.Repeat("a", 65), 1),
		"invalid_alias":             strings.Replace(validYAML, "alias: coding-default", "alias: 'wrong/alias'", 1),
		"long_alias":                strings.Replace(validYAML, "alias: coding-default", "alias: "+strings.Repeat("a", 129), 1),
		"model_space":               strings.Replace(validYAML, "provider/model-id", "'provider model'", 1),
		"model_long":                strings.Replace(validYAML, "provider/model-id", strings.Repeat("a", 257), 1),
		"fallback_missing":          validYAML + "        fallback_relays: [missing]\n",
		"fallback_primary":          validYAML + "        fallback_relays: [warp-1]\n",
		"least_inflight":            validYAML + "        routing: least_inflight\n",
		"strict":                    validYAML + "        affinity: strict\n",
		"strict_least":              validYAML + "        routing: least_inflight\n        affinity: strict\n",
		"mode_unknown":              validYAML + "        routing: FIXED\n",
	}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) { rejectConfig(t, s) })
	}
	for _, s := range []string{withSecondUpstream(validYAML, "secondary"), strings.Replace(validYAML, "  default_upstream: primary\n", "", 1)} {
		c, err := loadTest(s)
		if err != nil || c.Server().DefaultUpstream != "primary" {
			t.Fatal("default/namespace alias valid ditolak", err)
		}
	}
}

func TestTargetURLs(t *testing.T) {
	for _, target := range []string{"http://api.example/v1", "/v1", "https://", "https:opaque", "https://user:pass@api.example/v1", "https://api.example/v1?token=sentinel", "https://api.example/v1?", "https://api.example/v1#", "https://api.example/v1#x", "https://api.example:0/v1", "https://api.example:65536/v1", "https://api.example:/v1", "https://bad_host/v1", "https://é.example/v1", "https://api.example/a/../v1", "https://api.example/./v1", "https://api.example/%2e%2e/v1", "https://api.example/%252e/v1", "https://api.example/%2f/v1", "https://api.example/%5c/v1", "https://api.example/%00/v1", "https://api.example/a\\b", "https://api.example/a b"} {
		t.Run(target, func(t *testing.T) {
			rejectConfig(t, strings.Replace(validYAML, "https://api.provider.example/v1", target, 1))
		})
	}
	for _, target := range []string{"https://api.example", "https://api.example:443/v1/", "https://[::1]:8443/v1", "https://127.0.0.1/v1", "https://api.example/v1/hello%20there"} {
		if _, err := loadTest(strings.Replace(validYAML, "https://api.provider.example/v1", target, 1)); err != nil {
			t.Fatal("URL HTTPS valid ditolak", target, err)
		}
	}
	for _, target := range []string{"socks5://warp-1:1080", "http://warp-1:1080", "socks5h://warp-1", "socks5h://warp-1:0", "socks5h://warp-1:65536", "socks5h://user@warp-1:1080", "socks5h://warp-1:1080/", "socks5h://warp-1:1080?", "socks5h://warp-1:1080#", "socks5h://bad_host:1080"} {
		t.Run(target, func(t *testing.T) { rejectConfig(t, strings.Replace(validYAML, "socks5h://warp-1:1080", target, 1)) })
	}
	for _, target := range []string{"socks5h://relay.internal:1080", "socks5h://127.0.0.1:1080", "socks5h://[::1]:1080"} {
		if _, err := loadTest(strings.Replace(validYAML, "socks5h://warp-1:1080", target, 1)); err != nil {
			t.Fatal("relay valid ditolak", err)
		}
	}
}
func TestListeners(t *testing.T) {
	for _, addr := range []string{"localhost:8787", ":8787", "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1", "127.0.0.1:8788", "0.0.0.0:8788", "[::]:8788"} {
		t.Run(addr, func(t *testing.T) {
			rejectConfig(t, strings.Replace(validYAML, "listen: 127.0.0.1:8787", "listen: '"+addr+"'", 1))
		})
	}
	for _, addr := range []string{"0.0.0.0:8787", "[::1]:8787"} {
		if _, err := loadTest(strings.Replace(validYAML, "listen: 127.0.0.1:8787", "listen: '"+addr+"'", 1)); err != nil {
			t.Fatal("listener valid ditolak", err)
		}
	}
}

func TestCollectionCaps(t *testing.T) {
	for _, n := range []int{128, 129} {
		var extra strings.Builder
		for i := 1; i < n; i++ {
			fmt.Fprintf(&extra, "  - id: relay-%d\n    proxy_url: socks5h://relay-%d:1080\n    max_inflight: 5\n", i, i)
		}
		s := strings.Replace(validYAML, "upstreams:", extra.String()+"upstreams:", 1)
		_, err := loadTest(s)
		if (err == nil) != (n == 128) {
			t.Fatal("batas jumlah relay salah", n, err)
		}
	}
	for _, n := range []int{64, 65} {
		s := validYAML
		block := validYAML[strings.Index(validYAML, "  - id: primary\n"):]
		for i := 1; i < n; i++ {
			s += strings.Replace(block, "id: primary", fmt.Sprintf("id: provider-%d", i), 1)
		}
		_, err := loadTest(s)
		if (err == nil) != (n == 64) {
			t.Fatal("batas jumlah upstream salah", n, err)
		}
	}
	for _, n := range []int{256, 257} {
		s := validYAML
		for i := 1; i < n; i++ {
			s += fmt.Sprintf("      - alias: model-%d\n        upstream_model: provider/model-id\n        primary_relay: warp-1\n", i)
		}
		_, err := loadTest(s)
		if (err == nil) != (n == 256) {
			t.Fatal("batas model per upstream salah", n, err)
		}
	}
	for _, n := range []int{1024, 1025} {
		s := validYAML[:strings.Index(validYAML, "upstreams:")] + "upstreams:\n"
		for i, left := 0, n; left > 0; i++ {
			count := min(left, 256)
			left -= count
			id := fmt.Sprintf("provider-%d", i)
			if i == 0 {
				id = "primary"
			}
			s += fmt.Sprintf("  - id: %s\n    base_url: https://api.provider.example/v1\n    api_key_env: UPSTREAM_KEY\n    relays: [warp-1]\n    models:\n", id)
			for j := 0; j < count; j++ {
				s += fmt.Sprintf("      - alias: model-%d\n        upstream_model: provider/model-id\n        primary_relay: warp-1\n", j)
			}
		}
		_, err := loadTest(s)
		if (err == nil) != (n == 1024) {
			t.Fatal("batas total model salah", n, err)
		}
	}
}

func TestRelayAllowlistMembership(t *testing.T) {
	two := strings.Replace(validYAML, "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 5\nupstreams:", 1)
	rejectConfig(t, strings.Replace(two, "primary_relay: warp-1", "primary_relay: warp-2", 1))
	rejectConfig(t, two+"        fallback_relays: [warp-2]\n")
	both := strings.Replace(two, "relays: [warp-1]", "relays: [warp-1, warp-2]", 1)
	rejectConfig(t, both+"        fallback_relays: [warp-2, warp-2]\n")
	if _, err := loadTest(both + "        fallback_relays: [warp-2]\n"); err != nil {
		t.Fatal("fallback valid ditolak", err)
	}
}
