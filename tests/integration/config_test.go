package integration_test

import (
	"opencode-infinity/internal/catalog"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"testing"
)

// Membuktikan pemuatan config → catalog offline, bukan startup/restart server.
func TestConfigurationCatalogSnapshots(t *testing.T) {
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	text = strings.Replace(text, "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 5\nupstreams:", 1)
	text = strings.ReplaceAll(text, "relays: [warp-1]", "relays: [warp-1, warp-2]")
	text = strings.Replace(text, "fallback_relays: []", "fallback_relays: [warp-2]", 1)
	second := text[strings.Index(text, "  - id: primary\n"):]
	second = strings.Replace(second, "id: primary", "id: secondary", 1)
	second = strings.Replace(second, "provider-model-id", "secondary-model-id", 1)
	text += second
	values := map[string]string{"INFINITY_GATEWAY_TOKEN": "synthetic-gateway-credential-12345", "INFINITY_ADMIN_TOKEN": "synthetic-admin-credential-67890", "INFINITY_UPSTREAM_KEY": "synthetic-key-version-one"}
	src := config.Sources{LookupEnv: func(name string) (string, bool) { v, ok := values[name]; return v, ok }}
	a, err := config.Load(strings.NewReader(text), src)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := catalog.New(a)
	if err != nil {
		t.Fatal(err)
	}
	_, m, ok := reg.Resolve("", "coding-default")
	if !ok || m.UpstreamModel != "provider-model-id" || len(m.FallbackRelays) != 1 {
		t.Fatal("default alias salah")
	}
	_, other, ok := reg.Resolve("secondary", "coding-default")
	if !ok || other.UpstreamModel != "secondary-model-id" {
		t.Fatal("namespace alias lintas upstream salah")
	}
	m.FallbackRelays[0] = "changed"
	u, _, _ := reg.Resolve("primary", "coding-default")
	u.Models[0].FallbackRelays[0] = "changed"
	u.Relays[0] = "changed"
	copy := a.Upstreams()
	copy[0].Models[0].FallbackRelays[0] = "changed"
	_, next, _ := reg.Resolve("primary", "coding-default")
	if next.FallbackRelays[0] != "warp-2" || a.Upstreams()[0].Models[0].FallbackRelays[0] != "warp-2" {
		t.Fatal("nested copy memutasi snapshot")
	}
	values["INFINITY_UPSTREAM_KEY"] = "synthetic-key-version-two"
	b, err := config.Load(strings.NewReader(text), src)
	if err != nil {
		t.Fatal(err)
	}
	oldKey, _ := a.APIKey("primary")
	newKey, _ := b.APIKey("primary")
	if oldKey.Reveal() != "synthetic-key-version-one" || newKey.Reveal() != "synthetic-key-version-two" {
		t.Fatal("secret snapshot berubah sebagian")
	}
	bad, err := config.Load(strings.NewReader(strings.Replace(text, "id: secondary", "id: primary", 1)), src)
	if err == nil || bad != nil {
		t.Fatal("reload invalid menghasilkan hasil parsial")
	}
	lateFailure := strings.Replace(text, "api_key_env: INFINITY_UPSTREAM_KEY", "api_key_env: ABSENT", 1)
	bad, err = config.Load(strings.NewReader(lateFailure), src)
	if err == nil || bad != nil {
		t.Fatal("kegagalan secret menghasilkan snapshot parsial")
	}
	_, next, _ = reg.Resolve("primary", "coding-default")
	if next.FallbackRelays[0] != "warp-2" {
		t.Fatal("load gagal memutasi catalog lama")
	}
}
