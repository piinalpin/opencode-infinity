package pool

import (
	"opencode-infinity/internal/config"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestHealthInitial(t *testing.T) {
	now := time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)
	h, err := NewHealth(admissionConfig(t), func() time.Time { return now })
	if err != nil || h == nil {
		t.Fatal("health untuk konfigurasi valid tidak tersedia")
	}
	v := h.View()
	if v.Ready || !v.Accepting || v.Revision != 1 || !v.StateChangedAt.Equal(now) || !v.SnapshotAt.Equal(now) || v.DefaultUpstream != "primary" {
		t.Fatal("startup tanpa observasi harus unknown/not-ready dengan metadata awal")
	}
	if len(v.Relays) != 1 || v.Relays[0].ID != "warp-1" || !v.Relays[0].DefaultPrimary || v.Relays[0].SOCKS5.State != HealthUnknown || v.Relays[0].HTTPS.State != HealthUnknown {
		t.Fatal("primary default atau lapisan unknown salah")
	}
	v.Relays[0].ID = "mutated"
	if h.View().Relays[0].ID != "warp-1" {
		t.Fatal("salinan view mengubah state owner")
	}
}

func TestHealthLayersAndPrimary(t *testing.T) {
	now := time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)
	c := admissionConfig(t, "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:", "relays: [warp-1]", "relays: [warp-1, warp-2]", "fallback_relays: []", "fallback_relays: [warp-2]")
	h, err := NewHealth(c, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	healthy := ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := h.Observe("warp-2", layer, healthy); err != nil {
			t.Fatal("observasi relay terdaftar ditolak")
		}
	}
	if h.View().Ready {
		t.Fatal("fallback sehat tidak boleh membuat primary default siap")
	}
	if err := h.Observe("warp-1", HealthSOCKS5, healthy); err != nil {
		t.Fatal(err)
	}
	v := h.View()
	if v.Ready || v.Relays[0].HTTPS.State != HealthUnknown {
		t.Fatal("sukses SOCKS5 tidak membuktikan HTTPS")
	}
	if err := h.Observe("warp-1", HealthHTTPS, healthy); err != nil {
		t.Fatal(err)
	}
	v = h.View()
	if !v.Ready || v.Revision != 5 {
		t.Fatal("primary dengan dua lapisan sehat belum ready")
	}
	now = now.Add(time.Second)
	failed := ObservationView{State: HealthUnhealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}
	if err := h.Observe("warp-1", HealthSOCKS5, failed); err != nil {
		t.Fatal(err)
	}
	if v = h.View(); v.Ready || !v.Accepting || v.Relays[0].HTTPS.State != HealthHealthy || !v.StateChangedAt.Equal(now) {
		t.Fatal("relay gagal harus not-ready tanpa mengubah lapisan lain/proses")
	}
}

func TestHealthFreshnessAndOrdering(t *testing.T) {
	now := time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)
	h, _ := NewHealth(admissionConfig(t), func() time.Time { return now })
	observation := ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := h.Observe("warp-1", layer, observation); err != nil {
			t.Fatal(err)
		}
	}
	stored := h.View()
	now = observation.ValidUntil
	v := h.View()
	if v.Ready || v.Relays[0].SOCKS5.State != HealthStale || v.Revision != stored.Revision || !v.StateChangedAt.Equal(stored.StateChangedAt) || !v.Relays[0].SOCKS5.ObservedAt.Equal(observation.ObservedAt) || !v.SnapshotAt.Equal(now) {
		t.Fatal("expiry harus not-ready tanpa memperbarui observasi/revisi")
	}
	// Read berulang tidak memperpanjang expiry atau memutasi stored outcome.
	if !reflect.DeepEqual(v, h.View()) {
		t.Fatal("read health memutasi state")
	}
	now = now.Add(time.Second)
	failure := ObservationView{State: HealthUnhealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}
	if err := h.Observe("warp-1", HealthSOCKS5, failure); err != nil {
		t.Fatal(err)
	}
	before := h.View()
	for _, invalid := range []ObservationView{
		observation,
		{State: HealthHealthy, ObservedAt: now.Add(time.Second), ValidUntil: now.Add(time.Minute)},
		{State: HealthHealthy, ObservedAt: now, ValidUntil: now},
		{State: HealthHealthy, ObservedAt: now.Add(-time.Second), ValidUntil: now.Add(time.Minute)},
		{State: HealthHealthy, ObservedAt: now, ValidUntil: failure.ValidUntil},
		{State: HealthUnhealthy, ObservedAt: now, ValidUntil: failure.ValidUntil.Add(time.Minute)},
		{State: HealthHealthy, ValidUntil: failure.ValidUntil},
		{State: HealthUnknown, ObservedAt: now, ValidUntil: failure.ValidUntil},
	} {
		if err := h.Observe("warp-1", HealthSOCKS5, invalid); err == nil {
			t.Fatal("observasi invalid/lama/konflik diterima")
		}
		if !reflect.DeepEqual(before, h.View()) {
			t.Fatal("observasi ditolak mengubah state")
		}
	}
	if err := h.Observe("warp-1", HealthSOCKS5, failure); err != nil || !reflect.DeepEqual(before, h.View()) {
		t.Fatal("duplikat identik harus idempotent")
	}
	if h.Observe("unknown", HealthSOCKS5, failure) == nil || h.Observe("warp-1", HealthLayer(255), failure) == nil {
		t.Fatal("relay/lapisan unknown diterima")
	}
}

func TestHealthAccepting(t *testing.T) {
	now := time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)
	h, _ := NewHealth(admissionConfig(t), func() time.Time { return now })
	observation := ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := h.Observe("warp-1", layer, observation); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(time.Second)
	if err := h.SetAccepting(false); err != nil {
		t.Fatal("perubahan accepting tidak tersedia")
	}
	v := h.View()
	if v.Ready || v.Accepting || v.Revision != 4 || !v.StateChangedAt.Equal(now) || !v.Relays[0].SOCKS5.ObservedAt.Equal(observation.ObservedAt) {
		t.Fatal("accepting false harus mengubah kesiapan, bukan observasi")
	}
	if err := h.SetAccepting(false); err != nil || !reflect.DeepEqual(v, h.View()) {
		t.Fatal("state accepting sama tidak boleh menaikkan revision")
	}
	now = now.Add(time.Second)
	if err := h.SetAccepting(true); err != nil || !h.View().Ready || h.View().Revision != 5 {
		t.Fatal("accepting true dengan transport fresh tidak pulih")
	}
}

func TestHealthConstructorBounds(t *testing.T) {
	now := time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	for _, tc := range []struct {
		config *config.Config
		clock  func() time.Time
	}{
		{admissionConfig(t, "state_bytes: 16777216", "state_bytes: 1"), clock},
		{nil, clock}, {&config.Config{}, clock}, {admissionConfig(t), nil},
		{admissionConfig(t), func() time.Time { return time.Time{} }},
		{admissionConfig(t), func() time.Time { return time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) }},
	} {
		if h, err := NewHealth(tc.config, tc.clock); err == nil || h != nil {
			t.Fatal("health menerima budget/state/clock tidak valid")
		}
	}
	var zero Health
	if zero.View().Ready || zero.Observe("warp-1", HealthSOCKS5, ObservationView{}) == nil || zero.SetAccepting(true) == nil {
		t.Fatal("zero-value owner tidak fail-closed")
	}
}

func TestHealthDefaultScope(t *testing.T) {
	now := time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)
	relay := "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:"
	secondary := "affinity: none\n  - id: secondary\n    base_url: https://secondary.example/v1\n    api_key_env: INFINITY_UPSTREAM_KEY\n    relays: [warp-2]\n    models:\n      - alias: coding-secondary\n        upstream_model: secondary-model\n        primary_relay: warp-2"
	otherModel := "affinity: none\n      - alias: second-default\n        upstream_model: second-model\n        primary_relay: warp-2"
	for _, tc := range []struct {
		name  string
		edits []string
		ready bool
	}{
		{"nondefault", []string{"upstreams:", relay, "affinity: none", secondary}, false},
		{"one-default-mapping", []string{"upstreams:", relay, "relays: [warp-1]", "relays: [warp-1, warp-2]", "affinity: none", otherModel}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, err := NewHealth(admissionConfig(t, tc.edits...), func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
				if err := h.Observe("warp-2", layer, ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}); err != nil {
					t.Fatal(err)
				}
			}
			if h.View().Ready != tc.ready {
				t.Fatal("scope primary/default salah")
			}
		})
	}
}

func TestHealthConcurrentSnapshots(t *testing.T) {
	h, err := NewHealth(admissionConfig(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now()
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := h.Observe("warp-1", layer, ObservationView{State: HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	var workers sync.WaitGroup
	workers.Go(func() {
		for i := 0; i < 100; i++ {
			if err := h.SetAccepting(i%2 == 0); err != nil {
				t.Error(err)
			}
			at := time.Now()
			state := HealthHealthy
			if i%2 != 0 {
				state = HealthUnhealthy
			}
			if err := h.Observe("warp-1", HealthSOCKS5, ObservationView{State: state, ObservedAt: at, ValidUntil: at.Add(time.Minute)}); err != nil {
				t.Error(err)
			}
		}
	})
	for range 8 {
		workers.Go(func() {
			var last uint64
			for range 100 {
				v := h.View()
				if v.Revision < last || v.SnapshotAt.Before(v.StateChangedAt) || v.Ready && (!v.Accepting || v.Relays[0].SOCKS5.State != HealthHealthy || v.Relays[0].HTTPS.State != HealthHealthy) {
					t.Error("snapshot mencampur revisi/state/waktu")
				}
				last = v.Revision
				v.Relays[0].ID = "mutated"
			}
		})
	}
	workers.Wait()
	if h.View().Relays[0].ID != "warp-1" {
		t.Fatal("view konkuren memutasi owner")
	}
}
