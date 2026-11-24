package catalog

import (
	"fmt"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"sync"
	"testing"
)

func fixture(t *testing.T) *config.Config {
	t.Helper()
	b, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(strings.NewReader(string(b)), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": "synthetic-gateway-credential-12345", "INFINITY_ADMIN_TOKEN": "synthetic-admin-credential-67890", "INFINITY_UPSTREAM_KEY": "synthetic-provider-credential"}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestCatalogLookup(t *testing.T) {
	cfg := fixture(t)
	r, err := New(cfg)
	if err != nil || r == nil {
		t.Fatal("catalog valid ditolak", err)
	}
	if r.DefaultUpstream() != "primary" {
		t.Fatal("default upstream salah")
	}
	u, m, ok := r.Resolve("", "coding-default")
	if !ok || u.ID != "primary" || m.UpstreamModel != "provider-model-id" || m.PrimaryRelay != "warp-1" {
		t.Fatal("lookup alias/default salah")
	}
	if _, _, ok := r.Resolve("unknown", "coding-default"); ok {
		t.Fatal("upstream unknown memakai fallback")
	}
	if _, _, ok := r.Resolve("primary", "unknown"); ok {
		t.Fatal("alias unknown memakai fallback")
	}
	if relay, ok := r.Relay("warp-1"); !ok || relay.MaxInflight != 5 {
		t.Fatal("relay lookup salah")
	}
	if _, ok := r.Relay("missing"); ok {
		t.Fatal("relay unknown ditemukan")
	}
	if r, err := New(nil); err == nil || r != nil {
		t.Fatal("config nil diterima")
	}
	if r, err := New(&config.Config{}); err == nil || r != nil {
		t.Fatal("config belum divalidasi diterima")
	}
}

func TestCatalogCopies(t *testing.T) {
	r, err := New(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	u, _, ok := r.Resolve("primary", "coding-default")
	if !ok {
		t.Fatal("alias hilang")
	}
	u.Relays[0] = "changed"
	u.Models[0].Alias = "changed"
	u, _, ok = r.Resolve("primary", "coding-default")
	if !ok || u.Relays[0] != "warp-1" || u.Models[0].Alias != "coding-default" {
		t.Fatal("hasil lookup memutasi catalog")
	}
	u, ok = r.Upstream("primary")
	if !ok {
		t.Fatal("upstream hilang")
	}
	u.Relays[0] = "changed"
	u.Models[0].UpstreamModel = "changed"
	next, ok := r.Upstream("primary")
	if !ok || next.Relays[0] != "warp-1" || next.Models[0].UpstreamModel != "provider-model-id" {
		t.Fatal("salinan metadata memutasi catalog")
	}
	if _, ok := r.Upstream("missing"); ok {
		t.Fatal("upstream unknown ditemukan")
	}
}

func TestConcurrentCatalogCopies(t *testing.T) {
	r, err := New(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				u, m, ok := r.Resolve("primary", "coding-default")
				if !ok || m.UpstreamModel != "provider-model-id" || u.Models[0].Alias != "coding-default" {
					t.Error("snapshot concurrent berubah")
					return
				}
				u.Relays[0] = "changed"
				u.Models[0].Alias = "changed"
				v, ok := r.Upstream("primary")
				if !ok || v.Relays[0] != "warp-1" {
					t.Error("metadata concurrent berubah")
					return
				}
				v.Models[0].Alias = "changed"
				_, _ = r.Relay("warp-1")
				_ = fmt.Sprintf("%#v", r)
			}
		}()
	}
	wg.Wait()
}
