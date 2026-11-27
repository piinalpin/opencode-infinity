package pool

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func readyAt(t *testing.T, cEdits []string, clock func() time.Time) *Manager {
	t.Helper()
	a, err := New(admissionConfig(t, cEdits...), clock, Policy{Failures: 1, Successes: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range a.View().Relays {
		for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
			at := clock()
			if err := a.Observe(r.ID, layer, ObservationView{State: HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
		}
	}
	return a
}

func TestCooldownDeadlineAndLease(t *testing.T) {
	now := time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)
	a := readyAt(t, nil, func() time.Time { return now })
	setter, ok := any(a).(interface{ ExtendRelayCooldown(string, time.Time) error })
	if !ok {
		t.Fatal("pool belum memiliki cooldown relay berdeadline")
	}
	held, _ := a.Begin(100)
	if err := admitOnly(held, "primary", "coding-default", 0); err != nil {
		t.Fatal(err)
	}
	until := now.Add(time.Minute)
	if err := setter.ExtendRelayCooldown("warp-1", until); err != nil {
		t.Fatal(err)
	}
	revision := a.View().Revision
	for _, deadline := range []time.Time{until, now.Add(30 * time.Second)} {
		if err := setter.ExtendRelayCooldown("warp-1", deadline); err != nil || a.View().Revision != revision {
			t.Fatal("duplicate/shorter deadline mengubah cooldown")
		}
	}
	reject := func() {
		t.Helper()
		r, ok := a.Begin(100)
		if !ok {
			t.Fatal("pending fixture ditolak")
		}
		defer r.Release()
		if !errors.Is(admitOnly(r, "primary", "coding-default", 0), ErrNoEligibleRelay) {
			t.Fatal("cooldown tidak memblokir assignment")
		}
	}
	reject()
	if a.active != 1 {
		t.Fatal("cooldown melepas lease admitted")
	}
	now = now.Add(time.Second)
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := a.Observe("warp-1", layer, ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	reject()
	if !a.View().Ready {
		t.Fatal("cooldown merusak readiness transport")
	}
	extended := until.Add(time.Minute)
	if err := setter.ExtendRelayCooldown("warp-1", extended); err != nil {
		t.Fatal(err)
	}
	revision = a.View().Revision
	held.Release()
	now = until
	reject()
	now = extended
	r, _ := a.Begin(100)
	if err := admitOnly(r, "primary", "coding-default", 0); err != nil {
		t.Fatal("cooldown tidak pulih tepat deadline")
	}
	r.Release()
	if a.View().Revision != revision {
		t.Fatal("read/expiry mengubah revisi persisted")
	}
	for _, target := range []struct {
		id    string
		until time.Time
	}{{"unknown", now.Add(time.Minute)}, {"warp-1", time.Time{}}, {"warp-1", now}, {"warp-1", now.Add(-time.Second)}, {"warp-1", time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}} {
		if setter.ExtendRelayCooldown(target.id, target.until) == nil {
			t.Fatal("cooldown invalid diterima")
		}
	}
	now = time.Time{}
	if setter.ExtendRelayCooldown("warp-1", extended.Add(time.Minute)) == nil {
		t.Fatal("clock invalid diterima")
	}
	assertFixedLedger(t, a, true)
}

func leastEdits() []string {
	return []string{
		"upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 3\n  - id: outside\n    proxy_url: socks5h://outside:1080\n    max_inflight: 3\nupstreams:",
		"relays: [warp-1]", "relays: [warp-2, warp-1]",
		"max_active_requests: 2", "max_active_requests: 10", "outbound_connections: 2", "outbound_connections: 10", "max_inflight: 1", "max_inflight: 3",
		"routing: fixed", "routing: least_inflight\n        stateless: true",
		"    models:\n", "    models:\n      - alias: fixed-a\n        upstream_model: fixture-model\n        primary_relay: warp-1\n      - alias: fixed-b\n        upstream_model: fixture-model\n        primary_relay: warp-2\n",
	}
}
func TestLeastInflightMinimumAndTie(t *testing.T) {
	a, err := newReadyPool(admissionConfig(t, leastEdits()...))
	if err != nil {
		t.Fatal(err)
	}
	var held []*Reservation
	defer func() {
		for _, r := range held {
			r.Release()
		}
	}()
	for _, wanted := range []string{"warp-2", "warp-1", "warp-2", "warp-1", "warp-2", "warp-1"} {
		r, ok := a.Begin(100)
		if !ok {
			t.Fatal("persiapan ditolak")
		}
		held = append(held, r)
		if selected, err := r.Admit("primary", "coding-default", 0); err != nil || selected != wanted {
			t.Fatalf("hasil admission %s/%v, ingin %s", selected, err, wanted)
		}
		if r.relay != wanted {
			t.Fatalf("minimum/tie memilih %s, ingin %s", r.relay, wanted)
		}
		assertFixedLedger(t, a, false)
	}
	extra, _ := a.Begin(100)
	if !errors.Is(admitOnly(extra, "primary", "coding-default", 0), ErrCapacityExceeded) {
		t.Fatal("semua kandidat penuh tidak memberi capacity")
	}
	extra.Release()
	for _, r := range held {
		r.Release()
	}
	for range 4 {
		r, _ := a.Begin(100)
		if err := admitOnly(r, "primary", "coding-default", 0); err != nil || r.relay != "warp-2" {
			t.Fatal("tie serial tidak stabil sesuai allowlist")
		}
		r.Release()
	}
	if a.used["outside"] != 0 {
		t.Fatal("relay di luar allowlist dipakai")
	}
	assertFixedLedger(t, a, true)
}

func TestLeastInflightCandidateStateBudget(t *testing.T) {
	c := admissionConfig(t, "state_bytes: 16777216", "state_bytes: 6160")
	if a, err := New(c, time.Now, Policy{Failures: 1, Successes: 1}); err == nil || a != nil {
		t.Fatal("metadata kandidat upstream tidak masuk budget state")
	}
}

func TestLeastInflightEligibilityAndCapacity(t *testing.T) {
	for _, name := range []string{"cooldown", "disabled", "unhealthy", "stale", "full-low", "all-full", "all-cooldown", "cooldown-full", "model-unavailable", "non-accepting", "stream-cap", "fixed-busy", "fixed-full", "fixed-disabled"} {
		t.Run(name, func(t *testing.T) {
			now := time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)
			edits := leastEdits()
			if name == "full-low" {
				edits[1] = strings.Replace(edits[1], "max_inflight: 3", "max_inflight: 1", 1)
			}
			a := readyAt(t, edits, func() time.Time { return now })
			var held []*Reservation
			defer func() {
				for _, r := range held {
					r.Release()
				}
			}()
			hold := func(alias string, n int) {
				t.Helper()
				for range n {
					r, ok := a.Begin(100)
					if !ok {
						t.Fatal("persiapan hold ditolak")
					}
					held = append(held, r)
					if _, err := r.Admit("primary", alias, 0); err != nil {
						t.Fatal(err)
					}
				}
			}
			cool := func(id string) {
				t.Helper()
				if err := a.ExtendRelayCooldown(id, now.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
			}
			wanted := "warp-1"
			var want error
			alias := "coding-default"
			var stream int64
			switch name {
			case "cooldown":
				cool("warp-2")
			case "disabled":
				if err := a.SetMode("warp-2", "disabled"); err != nil {
					t.Fatal(err)
				}
			case "unhealthy":
				now = now.Add(time.Second)
				if err := a.Observe("warp-2", HealthHTTPS, ObservationView{State: HealthUnhealthy, ObservedAt: now, ValidUntil: now.Add(time.Hour)}); err != nil {
					t.Fatal(err)
				}
			case "stale":
				now = now.Add(time.Hour)
				for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
					if err := a.Observe("warp-1", layer, ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Hour)}); err != nil {
						t.Fatal(err)
					}
				}
			case "full-low":
				hold("fixed-b", 1)
				hold("fixed-a", 2)
			case "all-full":
				hold("fixed-b", 3)
				hold("fixed-a", 3)
				want = ErrCapacityExceeded
			case "all-cooldown":
				cool("warp-1")
				cool("warp-2")
				want = ErrNoEligibleRelay
			case "cooldown-full":
				cool("warp-2")
				hold("fixed-a", 3)
				want = ErrCapacityExceeded
			case "model-unavailable":
				if err := a.ObserveModel("primary", alias, ObservationView{State: HealthUnhealthy, ObservedAt: now, ValidUntil: now.Add(time.Hour)}); err != nil {
					t.Fatal(err)
				}
				want = ErrNoEligibleRelay
			case "non-accepting":
				if err := a.SetAccepting(false); err != nil {
					t.Fatal(err)
				}
				want = ErrNoEligibleRelay
			case "stream-cap":
				stream = 1 << 60
				want = ErrCapacityExceeded
			case "fixed-busy":
				hold("fixed-a", 1)
				alias = "fixed-a"
			case "fixed-full":
				hold("fixed-a", 3)
				alias = "fixed-a"
				want = ErrCapacityExceeded
			case "fixed-disabled":
				if err := a.SetMode("warp-1", "disabled"); err != nil {
					t.Fatal(err)
				}
				alias = "fixed-a"
				want = ErrNoEligibleRelay
			}
			r, ok := a.Begin(100)
			if !ok {
				t.Fatal("persiapan tambahan ditolak")
			}
			defer r.Release()
			selected, err := r.Admit("primary", alias, stream)
			if !errors.Is(err, want) {
				t.Fatalf("kategori %v, ingin %v", err, want)
			}
			if want == nil {
				if selected != wanted || r.relay != selected {
					t.Fatalf("pilihan %s, ingin %s", selected, wanted)
				}
			} else if selected != "" || r.admitted || r.relay != "" || r.streamBytes != 0 {
				t.Fatal("admission gagal meninggalkan pilihan/reservasi parsial")
			}
			assertFixedLedger(t, a, false)
		})
	}
}

func TestLeastInflightUnknownAndCooldownExpiry(t *testing.T) {
	now := time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)
	a, err := New(admissionConfig(t, leastEdits()...), func() time.Time { return now }, Policy{Failures: 1, Successes: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := a.Observe("warp-1", layer, ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	r, _ := a.Begin(100)
	id, err := r.Admit("primary", "coding-default", 0)
	if err != nil || id != "warp-1" {
		t.Fatal("unknown transport dipilih")
	}
	r.Release()
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := a.Observe("warp-2", layer, ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	until := now.Add(time.Minute)
	if err := a.ExtendRelayCooldown("warp-2", until); err != nil {
		t.Fatal(err)
	}
	for _, instant := range []time.Time{until.Add(-time.Nanosecond), until, until.Add(time.Nanosecond)} {
		now = instant
		r, _ := a.Begin(100)
		id, err := r.Admit("primary", "coding-default", 0)
		r.Release()
		want := "warp-2"
		if instant.Before(until) {
			want = "warp-1"
		}
		if err != nil || id != want {
			t.Fatalf("pilihan sebelum/tepat/sesudah expiry %s/%v, ingin%s", id, err, want)
		}
	}
}

func TestLeastInflightConcurrentReservations(t *testing.T) {
	a, err := newReadyPool(admissionConfig(t, leastEdits()...))
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		start := make(chan struct{})
		leases := make(chan *Reservation, 6)
		var workers sync.WaitGroup
		for range 6 {
			workers.Go(func() {
				<-start
				r, ok := a.Begin(100)
				if !ok {
					t.Error("pending burst ditolak")
					leases <- nil
					return
				}
				_, err := r.Admit("primary", "coding-default", 0)
				if err != nil {
					r.Release()
					t.Error(err)
					leases <- nil
					return
				}
				leases <- r
			})
		}
		close(start)
		workers.Wait()
		a.mu.Lock()
		if a.used["warp-1"] != 3 || a.used["warp-2"] != 3 || a.active != 6 {
			t.Error("burst tidak mereservasi kedua relay secara atomik")
		}
		a.mu.Unlock()
		r, ok := a.Begin(100)
		if !ok {
			t.Fatal("pending tambahan ditolak")
		}
		if id, err := r.Admit("primary", "coding-default", 0); id != "" || !errors.Is(err, ErrCapacityExceeded) {
			t.Error("burst tambahan melebihi kapasitas")
		}
		r.Release()
		close(leases)
		for r := range leases {
			if r != nil {
				workers.Go(r.Release)
				workers.Go(r.Release)
			}
		}
		workers.Wait()
		assertFixedLedger(t, a, true)
	}
	// Sesudah update cooldown selesai, promotion tidak boleh memakai relay itu walau inflight nol.
	if err := a.ExtendRelayCooldown("warp-2", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		workers := sync.WaitGroup{}
		for range 4 {
			workers.Go(func() {
				r, ok := a.Begin(100)
				if !ok {
					return
				}
				defer r.Release()
				id, err := r.Admit("primary", "coding-default", 0)
				if err == nil && id != "warp-1" {
					t.Error("promotion mengabaikan update cooldown yang selesai")
				}
				assertFixedLedger(t, a, false)
			})
		}
		workers.Wait()
	}
	assertFixedLedger(t, a, true)
}

func TestLeastInflightSharedAcrossUpstreams(t *testing.T) {
	edits := append(leastEdits(), "        affinity: none", "        affinity: none\n  - id: secondary\n    base_url: https://secondary.example/v1\n    api_key_env: INFINITY_UPSTREAM_KEY\n    relays: [warp-2, warp-1]\n    models:\n      - alias: coding-secondary\n        upstream_model: fixture-secondary\n        primary_relay: warp-1\n        routing: least_inflight\n        stateless: true")
	a, err := newReadyPool(admissionConfig(t, edits...))
	if err != nil {
		t.Fatal(err)
	}
	first, _ := a.Begin(100)
	defer first.Release()
	if id, err := first.Admit("primary", "coding-default", 0); err != nil || id != "warp-2" {
		t.Fatal("pilihan awal salah")
	}
	second, _ := a.Begin(100)
	defer second.Release()
	if id, err := second.Admit("secondary", "coding-secondary", 0); err != nil || id != "warp-1" {
		t.Fatal("counter inflight tidak dibagi lintas upstream")
	}
}

func TestLeastInflightMixedLifecycle(t *testing.T) {
	a, err := newReadyPool(admissionConfig(t, leastEdits()...))
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	start := make(chan struct{})
	workers.Go(func() {
		<-start
		for range 100 {
			_ = a.SetMode("warp-1", "draining")
			_ = a.SetMode("warp-1", "serving")
			_ = a.ExtendRelayCooldown("warp-2", time.Now().Add(time.Minute))
		}
	})
	for index := range 8 {
		workers.Go(func() {
			<-start
			for range 100 {
				r, ok := a.Begin(100)
				if !ok {
					continue
				}
				switch index % 3 {
				case 0:
					_ = r.AdmitLocal()
				case 1:
					_, _ = r.Admit("primary", "fixed-a", 0)
				default:
					_, _ = r.Admit("primary", "coding-default", 32768)
				}
				r.ReserveStream(32768)
				r.Release()
				r.Release()
				if release, ok := a.TryProbe("warp-1", HealthSOCKS5); ok {
					release()
					release()
				}
				assertFixedLedger(t, a, false)
			}
		})
	}
	close(start)
	workers.Wait()
	assertFixedLedger(t, a, true)
}

func TestLeastInflightAbsoluteCountNotUtilization(t *testing.T) {
	edits := leastEdits()
	edits[1] = strings.Replace(edits[1], "max_inflight: 3", "max_inflight: 100", 1)
	a, err := newReadyPool(admissionConfig(t, edits...))
	if err != nil {
		t.Fatal(err)
	}
	var held []*Reservation
	defer func() {
		for _, r := range held {
			r.Release()
		}
	}()
	for _, alias := range []string{"fixed-b", "fixed-b", "fixed-a"} {
		r, _ := a.Begin(100)
		held = append(held, r)
		if _, err := r.Admit("primary", alias, 0); err != nil {
			t.Fatal(err)
		}
	}
	// warp-2 memiliki utilisasi 2/100, tetapi inflight 2 melebihi warp-1 dengan 1/3.
	r, _ := a.Begin(100)
	defer r.Release()
	selected, err := r.Admit("primary", "coding-default", 0)
	if err != nil || selected != "warp-1" {
		t.Fatal("least harus membandingkan jumlah absolut, bukan rasio kapasitas")
	}
}
