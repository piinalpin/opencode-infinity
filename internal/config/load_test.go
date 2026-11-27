package config

import (
	"errors"
	"go.yaml.in/yaml/v4"
	"io"
	"strings"
	"testing"
)

const validYAML = `version: 1
server:
  listen: 127.0.0.1:8787
  admin_listen: 127.0.0.1:8788
  gateway_token_env: GATEWAY_TOKEN
  admin_token_env: ADMIN_TOKEN
  default_upstream: primary
  max_body_bytes: 10485760
  max_active_requests: 20
limits:
  body_buffer_bytes: 67108864
  stream_buffer_bytes: 8388608
  state_bytes: 16777216
  monitoring_bytes: 8388608
  inbound_connections: 64
  outbound_connections: 32
  bindings: 1024
  admin_sessions: 32
  parallel_probes: 2
  parallel_status: 4
  snapshot_bytes: 262144
timeouts:
  connect_seconds: 10
  response_headers_seconds: 120
  stream_idle_seconds: 120
  total_request_seconds: 1800
relays:
  - id: warp-1
    proxy_url: socks5h://warp-1:1080
    max_inflight: 5
upstreams:
  - id: primary
    base_url: https://api.provider.example/v1
    api_key_env: UPSTREAM_KEY
    relays: [warp-1]
    models:
      - alias: coding-default
        upstream_model: provider/model-id
        primary_relay: warp-1
`

func testSources() Sources {
	return Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"GATEWAY_TOKEN": "synthetic-gateway-credential-12345", "ADMIN_TOKEN": "synthetic-admin-credential-67890", "UPSTREAM_KEY": "synthetic-provider-credential"}[name]
		return v, ok
	}}
}
func loadTest(s string) (*Config, error) { return Load(strings.NewReader(s), testSources()) }

func TestLoadValidSnapshot(t *testing.T) {
	c, err := loadTest(validYAML)
	if err != nil || c == nil {
		t.Fatalf("konfigurasi valid ditolak: %v", err)
	}
	if c.Server().DefaultUpstream != "primary" || c.Server().MaxActiveRequests != 20 || c.Server().MaxBodyBytes != 10485760 {
		t.Fatal("server/default/batas tidak dimuat")
	}
	if len(c.Relays()) != 1 || c.Relays()[0].ProxyURL != "socks5h://warp-1:1080" || c.Relays()[0].MaxInflight != 5 {
		t.Fatal("relay tidak dimuat")
	}
	u := c.Upstreams()
	if len(u) != 1 || u[0].ID != "primary" || u[0].Models[0].Alias != "coding-default" || u[0].Models[0].Routing != "fixed" || u[0].Models[0].Affinity != "none" {
		t.Fatal("registry/default mode tidak dimuat")
	}
	if c.Limits().BodyBufferBytes != 67108864 || c.Timeouts().ConnectSeconds != 10 {
		t.Fatal("anggaran/timeout tidak dimuat")
	}
}

func rejectConfig(t *testing.T, s string) {
	t.Helper()
	c, err := loadTest(s)
	if err == nil || c != nil {
		t.Fatal("konfigurasi invalid diterima atau hasil parsial dikembalikan")
	}
}
func TestStrictSchema(t *testing.T) {
	cases := map[string]string{
		"unknown_root":        validYAML + "unknown: sentinel-private\n",
		"unknown_server":      strings.Replace(validYAML, "  listen:", "  unknown: sentinel-private\n  listen:", 1),
		"unknown_model":       validYAML + "        unknown: sentinel-private\n",
		"duplicate":           strings.Replace(validYAML, "version: 1", "version: 1\nversion: 1", 1),
		"version":             strings.Replace(validYAML, "version: 1", "version: 2", 1),
		"version_missing":     strings.Replace(validYAML, "version: 1\n", "", 1),
		"null":                strings.Replace(validYAML, "primary_relay: warp-1", "primary_relay: null", 1),
		"empty_explicit_mode": validYAML + "        routing: ''\n",
		"numeric_string":      strings.Replace(validYAML, "max_active_requests: 20", "max_active_requests: '20'", 1),
		"missing_server":      strings.Replace(validYAML, "server:", "server_unused:", 1),
		"second_doc":          validYAML + "---\nversion: 1\n",
		"empty_second_doc":    validYAML + "---\n",
		"trailing":            validYAML + "...\nmalformed: [\n",
		"root_sequence":       "- version: 1\n", "empty": "", "syntax": "server: [",
	}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) { rejectConfig(t, s) })
	}
	if _, err := loadTest(validYAML + "\n# komentar akhir\n"); err != nil {
		t.Fatal("komentar ditolak")
	}
}

func TestBoundedYAML(t *testing.T) {
	for name, s := range map[string]string{
		"anchor": strings.Replace(validYAML, "id: warp-1", "id: &r warp-1", 1),
		"alias":  strings.Replace(validYAML, "[warp-1]", "[&r warp-1, *r]", 1),
		"tag":    strings.Replace(validYAML, "provider/model-id", "!private provider/model-id", 1),
		"scalar": strings.Replace(validYAML, "provider/model-id", strings.Repeat("x", 4097), 1),
		"nodes":  validYAML + "        fallback_relays: [" + strings.Repeat("warp-1, ", 20001) + "]\n",
	} {
		t.Run(name, func(t *testing.T) { rejectConfig(t, s) })
	}
	exact := validYAML + "#" + strings.Repeat("p", (1<<20)-len(validYAML)-1)
	if _, err := loadTest(exact); err != nil {
		t.Fatal("input tepat 1 MiB ditolak")
	}
	r := strings.NewReader(exact + strings.Repeat("p", 100))
	c, err := Load(r, testSources())
	if err == nil || c != nil {
		t.Fatal("input melewati batas diterima")
	}
	if r.Len() != 99 {
		t.Fatal("reader tidak dibatasi cap+1")
	}
	if c, err := Load(nil, testSources()); err == nil || c != nil {
		t.Fatal("reader nil diterima")
	}
}

func TestSnapshotIsolation(t *testing.T) {
	a, err := loadTest(validYAML)
	if err != nil {
		t.Fatal(err)
	}
	rs := a.Relays()
	rs[0].ID = "changed"
	us := a.Upstreams()
	us[0].ID = "changed"
	us[0].Relays[0] = "changed"
	us[0].Models[0].Alias = "changed"
	us[0].Models[0].FallbackRelays = append(us[0].Models[0].FallbackRelays, "changed")
	if a.Relays()[0].ID != "warp-1" || a.Upstreams()[0].ID != "primary" || a.Upstreams()[0].Relays[0] != "warp-1" || a.Upstreams()[0].Models[0].Alias != "coding-default" || len(a.Upstreams()[0].Models[0].FallbackRelays) != 0 {
		t.Fatal("konsumen memutasi snapshot")
	}
	b, err := loadTest(strings.Replace(validYAML, "api_key_env: UPSTREAM_KEY", "api_key_env: ABSENT", 1))
	if b != nil || err == nil {
		t.Fatal("load gagal menghasilkan snapshot parsial")
	}
	c, err := loadTest(strings.Replace(validYAML, "provider/model-id", "provider/new-model", 1))
	if err != nil {
		t.Fatal(err)
	}
	if c.Upstreams()[0].Models[0].UpstreamModel != "provider/new-model" || a.Upstreams()[0].Models[0].UpstreamModel != "provider/model-id" {
		t.Fatal("versi snapshot bercampur")
	}
}

func TestSchemaScalarTypes(t *testing.T) {
	for name, pair := range map[string][2]string{
		"numeric_id":    {"id: primary", "id: 123"},
		"numeric_model": {"provider/model-id", "123"},
		"numeric_alias": {"coding-default", "123"},
		"numeric_env":   {"api_key_env: UPSTREAM_KEY", "api_key_env: 123"},
		"numeric_key":   {"version: 1", "1: 1"},
	} {
		t.Run(name, func(t *testing.T) { rejectConfig(t, strings.Replace(validYAML, pair[0], pair[1], 1)) })
	}
}

// Pemeriksaan batas kontrak parser, termasuk node valid yang tidak mengekspansi alias.
func TestParserStructureEdges(t *testing.T) {
	rejectConfig(t, strings.Replace(validYAML, "    models:\n      - alias: coding-default\n        upstream_model: provider/model-id\n        primary_relay: warp-1\n", "    models: []\n", 1))
	rejectConfig(t, strings.Replace(validYAML, "server:\n", "", 1))
	rejectConfig(t, strings.Replace(validYAML, "[warp-1]", "[123]", 1))
	rejectConfig(t, strings.Replace(validYAML, "primary_relay: warp-1", "<<: {primary_relay: warp-1}", 1))
	if c, err := Load(errorReader{}, testSources()); err == nil || c != nil || strings.Contains(err.Error(), "sentinel") {
		t.Fatal("read config failure tidak teredaksi")
	}
	// Bentuk nested input tetap dibatasi sebelum konversi struct.
	rejectConfig(t, "version: 1\nunknown: "+strings.Repeat("[", 40)+"x"+strings.Repeat("]", 40))
}

func FuzzLoad(f *testing.F) {
	for _, s := range []string{validYAML, "", "version: 1\n---\n", "version: &x [*x]"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		src := testSources()
		src.OpenFile = func(string) (io.ReadCloser, error) { return nil, errors.New("synthetic-file-disabled") }
		c, err := Load(strings.NewReader(s), src)
		if (err == nil) != (c != nil) {
			t.Fatal("hasil parsial/tanpa error")
		}
		if c != nil && !c.Validated() {
			t.Fatal("snapshot belum tervalidasi dipublikasikan")
		}
	})
}
func FuzzTargetsAndLimits(f *testing.F) {
	f.Add("https://api.example/v1", int64(16777216))
	f.Add("socks5h://relay:1080", int64(-1))
	f.Add("https://a/%252e", int64(9223372036854775807))
	var base document
	if err := yaml.Load([]byte(validYAML), &base); err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, s string, n int64) {
		_ = targetURL(s, false)
		_ = targetURL(s, true)
		d := base
		d.Limits.StateBytes = n
		err := validateLimits(&d)
		if n <= 0 && err == nil {
			t.Fatal("anggaran nonpositif diterima")
		}
	})
}

func TestLeastInflightConfiguration(t *testing.T) {
	for _, affinity := range []string{"", "        affinity: none\n"} {
		if _, err := loadTest(validYAML + "        routing: least_inflight\n        stateless: true\n" + affinity); err != nil {
			t.Fatalf("least-inflight stateless eksplisit ditolak: %v", err)
		}
	}
	for _, value := range []string{"false", "'true'", "'false'", "yes", "no", "on", "off", "TRUE", "False", "1", "null", "''", "[]", "{}", "!custom true"} {
		t.Run(value, func(t *testing.T) {
			rejectConfig(t, validYAML+"        routing: least_inflight\n        stateless: "+value+"\n")
		})
	}
	rejectConfig(t, validYAML+"        routing: least_inflight\n")
	rejectConfig(t, validYAML+"        routing: least_inflight\n        stateless: true\n        affinity: strict\n")
	rejectConfig(t, validYAML+"        routing: fixed\n        stateless: true\n        affinity: strict\n")
	rejectConfig(t, validYAML+"        stateless: true\n        stateless: true\n")
	for _, field := range []string{"routing", "alias", "upstream_model", "primary_relay"} {
		rejectConfig(t, validYAML+"        "+field+": true\n")
	}
	for _, value := range []string{"true", "false"} {
		if _, err := loadTest(validYAML + "        stateless: " + value + "\n"); err != nil {
			t.Fatalf("fixed dengan boolean canonical %s ditolak", value)
		}
	}
}
