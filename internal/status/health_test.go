package status

import (
	"encoding/json"
	"opencode-infinity/internal/pool"
	"strings"
	"testing"
	"time"
)

func TestHealthProjection(t *testing.T) {
	build := Build{Version: "v0.1.0-test", Revision: strings.Repeat("a", 40)}
	p, err := NewHealth(build)
	if err != nil || p == nil {
		t.Fatal("proyeksi health tidak tersedia")
	}
	at := time.Date(2026, 11, 26, 10, 0, 0, 0, time.FixedZone("private-zone", 7*3600))
	view := pool.HealthView{Accepting: true, Revision: 2, StateChangedAt: at, SnapshotAt: at.Add(time.Second), DefaultUpstream: "primary", Relays: []pool.RelayHealthView{{ID: "warp-1", DefaultPrimary: true, SOCKS5: pool.ObservationView{State: pool.HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}}}}
	v := p.Snapshot(view)
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{`"scope":"default_primary_transport"`, `"live":true`, `"ready":false`, `"upstream_model":"unknown"`, `"version":"v0.1.0-test"`, `"revision":2`, `"state":"healthy"`, `"state":"unknown"`, `"observed_at":null`, `2026-11-26T03:00:00Z`, `2026-11-26T03:00:01Z`} {
		if !strings.Contains(string(data), wanted) {
			t.Fatalf("metadata health hilang: %s", wanted)
		}
	}
	if strings.Contains(string(data), "private-zone") || strings.Contains(string(data), "+07:00") {
		t.Fatal("metadata waktu tidak dinormalisasi UTC")
	}
	view.Relays[0].ID = "mutated"
	if again, _ := json.Marshal(v); string(again) != string(data) {
		t.Fatal("proyeksi masih menunjuk slice caller")
	}
	unknown, err := NewHealth(Build{})
	if err != nil {
		t.Fatal(err)
	}
	data, _ = json.Marshal(unknown.Snapshot(pool.HealthView{}))
	if !strings.Contains(string(data), `"version":"unknown"`) || !strings.Contains(string(data), `"revision":"unknown"`) {
		t.Fatal("build absent ditebak sebagai versi produk")
	}
	for _, invalid := range []Build{{Version: "Bearer private-secret"}, {Version: "https://private.invalid"}, {Version: strings.Repeat("v", 100)}, {Revision: "private-token"}} {
		if _, err := NewHealth(invalid); err == nil || strings.Contains(err.Error(), "private") {
			t.Fatal("build metadata invalid diterima atau dibocorkan")
		}
	}
}
