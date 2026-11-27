package pool

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestEligibilityUnknownRejects(t *testing.T) {
	a, err := New(admissionConfig(t), time.Now, Policy{Failures: 3, Successes: 2})
	if err != nil {
		t.Fatal(err)
	}
	r, ok := a.Begin(100)
	if !ok {
		t.Fatal("persiapan ditolak")
	}
	defer r.Release()
	if err := admitOnly(r, "primary", "coding-default", 0); !errors.Is(err, ErrNoEligibleRelay) {
		t.Fatalf("startup tanpa observasi harus no_eligible_relay, mendapat %v", err)
	}
}

func TestEligibilityThresholds(t *testing.T) {
	now := time.Date(2026, 11, 27, 1, 0, 0, 0, time.UTC)
	a, err := New(admissionConfig(t), func() time.Time { return now }, Policy{Failures: 3, Successes: 2})
	if err != nil {
		t.Fatal(err)
	}
	observe := func(layer HealthLayer, state HealthState) {
		t.Helper()
		now = now.Add(time.Second)
		if err := a.Observe("warp-1", layer, ObservationView{State: state, ObservedAt: now, ValidUntil: now.Add(time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	observe(HealthSOCKS5, HealthHealthy)
	observe(HealthHTTPS, HealthHealthy)
	if a.View().Ready {
		t.Fatal("satu sukses tidak boleh menutup circuit startup dengan ambang dua")
	}
	observe(HealthSOCKS5, HealthHealthy)
	observe(HealthHTTPS, HealthHealthy)
	if !a.View().Ready {
		t.Fatal("dua sukses setiap lapisan harus ready")
	}
	for i := 0; i < 3; i++ {
		observe(HealthHTTPS, HealthUnhealthy)
		if a.View().Ready != (i < 2) {
			t.Fatalf("ambang gagal salah pada kegagalan %d", i+1)
		}
	}
	observe(HealthHTTPS, HealthHealthy)
	if a.View().Ready {
		t.Fatal("recovery sebelum ambang")
	}
	observe(HealthHTTPS, HealthHealthy)
	if !a.View().Ready {
		t.Fatal("recovery ambang tidak menutup circuit")
	}
	now = now.Add(time.Minute)
	if a.View().Ready {
		t.Fatal("transport expired tetap ready")
	}
	observe(HealthSOCKS5, HealthHealthy)
	observe(HealthHTTPS, HealthHealthy)
	if a.View().Ready {
		t.Fatal("expiry harus mereset streak recovery")
	}
}

func TestEligibilityModeAndActiveLease(t *testing.T) {
	a, err := newReadyPool(admissionConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	operator, ok := any(a).(interface{ SetMode(string, string) error })
	if !ok {
		t.Fatal("pool belum menyediakan mode operasi independen")
	}
	for _, mode := range []string{"draining", "disabled"} {
		active, _ := a.Begin(100)
		if err := admitOnly(active, "primary", "coding-default", 0); err != nil {
			t.Fatal(err)
		}
		if err := operator.SetMode("warp-1", mode); err != nil {
			t.Fatal(err)
		}
		at := time.Now()
		for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
			if err := a.Observe("warp-1", layer, ObservationView{State: HealthHealthy, ObservedAt: at, ValidUntil: at.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
		}
		if a.View().Ready {
			t.Fatal("mode non-serving tetap ready setelah probe sukses")
		}
		next, _ := a.Begin(100)
		if !errors.Is(admitOnly(next, "primary", "coding-default", 0), ErrNoEligibleRelay) {
			t.Fatal("probe mengaktifkan mode non-serving")
		}
		next.Release()
		if a.active != 1 || a.used["warp-1"] != 1 {
			t.Fatal("perubahan mode melepas lease aktif")
		}
		active.Release()
		if err := operator.SetMode("warp-1", "serving"); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][2]string{{"missing", "serving"}, {"warp-1", "invalid"}} {
		if operator.SetMode(args[0], args[1]) == nil {
			t.Fatal("mode/relay invalid diterima")
		}
	}
	if err := a.SetAccepting(false); err != nil {
		t.Fatal(err)
	}
	r, _ := a.Begin(100)
	defer r.Release()
	if !errors.Is(admitOnly(r, "primary", "coding-default", 0), ErrNoEligibleRelay) {
		t.Fatal("proses non-accepting menerima generasi")
	}
}

func TestEligibilityModelScope(t *testing.T) {
	now := time.Date(2026, 11, 27, 1, 0, 0, 0, time.UTC)
	c := admissionConfig(t, "    models:\n", "    models:\n      - alias: other\n        upstream_model: other-provider-model\n        primary_relay: warp-1\n        fallback_relays: []\n")
	a, err := New(c, func() time.Time { return now }, Policy{Failures: 1, Successes: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := a.Observe("warp-1", layer, ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	observer, ok := any(a).(interface {
		ObserveModel(string, string, ObservationView) error
	})
	if !ok {
		t.Fatal("pool belum memiliki observasi availability berscope")
	}
	check := func(alias string, want error) {
		t.Helper()
		r, _ := a.Begin(100)
		defer r.Release()
		if err := admitOnly(r, "primary", alias, 0); !errors.Is(err, want) {
			t.Fatalf("scope %s: %v, ingin %v", alias, err, want)
		}
	}
	check("coding-default", nil)
	failed := ObservationView{State: HealthUnhealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}
	if err := observer.ObserveModel("primary", "coding-default", failed); err != nil {
		t.Fatal(err)
	}
	check("coding-default", ErrNoEligibleRelay)
	check("other", nil)
	if !a.View().Ready {
		t.Fatal("availability model mengubah readiness transport")
	}
	before := a.View().Revision
	if err := observer.ObserveModel("primary", "coding-default", failed); err != nil || a.View().Revision != before {
		t.Fatal("duplikat availability tidak idempotent")
	}
	for _, pair := range [][2]string{{"missing", "coding-default"}, {"primary", "missing"}} {
		if observer.ObserveModel(pair[0], pair[1], failed) == nil {
			t.Fatal("scope unknown menambah state")
		}
	}
	now = failed.ValidUntil
	check("coding-default", nil)
	for _, m := range a.View().Models {
		if m.Alias == "coding-default" && m.Availability.State != HealthUnknown {
			t.Fatal("availability expired tidak kembali unknown")
		}
	}
	v := a.View()
	v.Models[0].Alias = "mutated"
	if a.View().Models[0].Alias == "mutated" {
		t.Fatal("view model mengubah owner")
	}
}

func TestEligibilityStateBudgetIncludesModels(t *testing.T) {
	c := admissionConfig(t, "state_bytes: 16777216", "state_bytes: 3000")
	if a, err := New(c, time.Now, Policy{Failures: 3, Successes: 2}); err == nil || a != nil {
		t.Fatal("budget state harus memperhitungkan circuit dan scope model")
	}
}

func TestEligibilityProbeSharesOutbound(t *testing.T) {
	a, err := newReadyPool(admissionConfig(t, "outbound_connections: 2", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1"))
	if err != nil {
		t.Fatal(err)
	}
	prober, ok := any(a).(interface {
		TryProbe(string, HealthLayer) (func(), bool)
	})
	if !ok {
		t.Fatal("probe belum memakai ledger outbound bersama")
	}
	release, ok := prober.TryProbe("warp-1", HealthSOCKS5)
	if !ok {
		t.Fatal("probe pertama ditolak")
	}
	if _, ok := prober.TryProbe("warp-1", HealthHTTPS); ok {
		t.Fatal("batas probe/outbound terlampaui")
	}
	r, _ := a.Begin(100)
	if !errors.Is(admitOnly(r, "primary", "coding-default", 0), ErrCapacityExceeded) {
		t.Fatal("generasi mengabaikan slot probe")
	}
	release()
	release()
	if admitOnly(r, "primary", "coding-default", 0) != nil {
		t.Fatal("slot probe tidak pulih")
	}
	if _, ok := prober.TryProbe("warp-1", HealthHTTPS); ok {
		t.Fatal("probe melampaui outbound generasi")
	}
	r.Release()
	for _, target := range []struct {
		relay string
		layer HealthLayer
	}{{"missing", HealthSOCKS5}, {"warp-1", HealthLayer(255)}} {
		if _, ok := prober.TryProbe(target.relay, target.layer); ok {
			t.Fatal("probe unknown menambah state")
		}
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 50 {
				if release, ok := prober.TryProbe("warp-1", HealthSOCKS5); ok {
					release()
					release()
				}
			}
		})
	}
	wg.Wait()
	assertFixedLedger(t, a, true)
}

func TestEligibilityLinearizationAndReplay(t *testing.T) {
	a, err := newReadyPool(admissionConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 100 {
				r, ok := a.Begin(100)
				if !ok {
					continue
				}
				_ = admitOnly(r, "primary", "coding-default", 0)
				r.Release()
				r.Release()
				assertFixedLedger(t, a, false)
			}
		})
	}
	for range 100 {
		if err := a.SetMode("warp-1", "draining"); err != nil {
			t.Fatal(err)
		}
		if r, ok := a.Begin(100); ok {
			if err := admitOnly(r, "primary", "coding-default", 0); !errors.Is(err, ErrNoEligibleRelay) {
				t.Error("promotion sesudah update drain berhasil")
			}
			r.Release()
		}
		if err := a.SetMode("warp-1", "serving"); err != nil {
			t.Fatal(err)
		}
	}
	workers.Wait()
	assertFixedLedger(t, a, true)
	now := time.Date(2026, 11, 27, 1, 0, 0, 0, time.UTC)
	a, err = New(admissionConfig(t), func() time.Time { return now }, Policy{Failures: 3, Successes: 2})
	if err != nil {
		t.Fatal(err)
	}
	one := ObservationView{State: HealthHealthy, ObservedAt: now, ValidUntil: now.Add(time.Minute)}
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := a.Observe("warp-1", layer, one); err != nil {
			t.Fatal(err)
		}
	}
	before := a.View().Revision
	for range 3 {
		if err := a.Observe("warp-1", HealthHTTPS, one); err != nil {
			t.Fatal(err)
		}
	}
	if a.View().Revision != before || a.View().Ready {
		t.Fatal("replay menambah streak atau revisi")
	}
	conflict := one
	conflict.State = HealthUnhealthy
	if a.Observe("warp-1", HealthHTTPS, conflict) == nil {
		t.Fatal("timestamp konflik diterima")
	}
	now = now.Add(time.Second)
	two := one
	two.ObservedAt = now
	two.ValidUntil = now.Add(time.Minute)
	for _, layer := range []HealthLayer{HealthSOCKS5, HealthHTTPS} {
		if err := a.Observe("warp-1", layer, two); err != nil {
			t.Fatal(err)
		}
	}
	if !a.View().Ready {
		t.Fatal("recovery kedua gagal")
	}
	now = two.ValidUntil
	r, _ := a.Begin(100)
	defer r.Release()
	if !errors.Is(admitOnly(r, "primary", "coding-default", 0), ErrNoEligibleRelay) {
		t.Fatal("expiry tepat valid_until masih eligible")
	}
}

func TestEligibilityStandaloneProbeMonitoringBudget(t *testing.T) {
	c := admissionConfig(t, "monitoring_bytes: 8388608", "monitoring_bytes: 1", "snapshot_bytes: 262144", "snapshot_bytes: 1", "parallel_status: 4", "parallel_status: 1")
	if a, err := New(c, time.Now, Policy{Failures: 3, Successes: 2}); err == nil || a != nil {
		t.Fatal("owner standalone mengabaikan budget monitoring probe")
	}
}
