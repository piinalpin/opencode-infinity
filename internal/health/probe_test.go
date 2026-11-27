package health

import (
	"context"
	"opencode-infinity/internal/config"
	"opencode-infinity/internal/pool"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type probeFunc func(context.Context, string, pool.HealthLayer) (pool.HealthState, error)

func (f probeFunc) Probe(ctx context.Context, r string, l pool.HealthLayer) (pool.HealthState, error) {
	return f(ctx, r, l)
}
func probeOwner(t *testing.T, edits ...string) *pool.Manager {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.NewReplacer(edits...).Replace(string(data))
	c, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
		switch name {
		case "INFINITY_GATEWAY_TOKEN":
			return "synthetic-gateway-token-123456", true
		case "INFINITY_ADMIN_TOKEN":
			return "synthetic-admin-token-12345678", true
		case "INFINITY_UPSTREAM_KEY":
			return "synthetic-upstream-key-12345", true
		}
		return "", false
	}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := pool.New(c, time.Now, pool.Policy{Failures: 3, Successes: 2})
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func fixtureSchedule() Schedule {
	return Schedule{Interval: 30 * time.Second, Jitter: 6 * time.Second, Timeout: 5 * time.Second, Freshness: 90 * time.Second, MaxBackoff: 120 * time.Second}
}
func TestProbeScheduleRecoveryAndReadPurity(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		owner := probeOwner(t)
		var mu sync.Mutex
		calls := map[pool.HealthLayer]int{}
		prober := probeFunc(func(ctx context.Context, relay string, layer pool.HealthLayer) (pool.HealthState, error) {
			mu.Lock()
			calls[layer]++
			mu.Unlock()
			return pool.HealthHealthy, nil
		})
		scheduler, err := New(owner, fixtureSchedule(), prober, func(time.Duration) time.Duration { return 0 })
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- scheduler.Run(ctx) }()
		synctest.Wait()
		mu.Lock()
		first := calls[pool.HealthSOCKS5] + calls[pool.HealthHTTPS]
		mu.Unlock()
		if first != 2 || owner.View().Ready {
			t.Fatal("scheduler harus menjalankan dua job awal tanpa mengabaikan ambang dua sukses")
		}
		for range 10 {
			owner.View()
		}
		mu.Lock()
		after := calls[pool.HealthSOCKS5] + calls[pool.HealthHTTPS]
		mu.Unlock()
		if after != first {
			t.Fatal("read memicu probe")
		}
		time.Sleep(30 * time.Second)
		synctest.Wait()
		if !owner.View().Ready {
			t.Fatal("jadwal kedua tidak memulihkan transport")
		}
		for _, m := range owner.View().Models {
			if m.Availability.State != pool.HealthUnknown {
				t.Fatal("probe transport mengarang availability model")
			}
		}
		cancel()
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		stopped := calls[pool.HealthSOCKS5] + calls[pool.HealthHTTPS]
		mu.Unlock()
		time.Sleep(time.Minute)
		synctest.Wait()
		mu.Lock()
		after = calls[pool.HealthSOCKS5] + calls[pool.HealthHTTPS]
		mu.Unlock()
		if after != stopped {
			t.Fatal("probe berlanjut sesudah shutdown")
		}
	})
}

func TestProbeBackoffJitter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		owner := probeOwner(t)
		var calls atomic.Int32
		scheduler, err := New(owner, fixtureSchedule(), probeFunc(func(context.Context, string, pool.HealthLayer) (pool.HealthState, error) {
			calls.Add(1)
			return pool.HealthUnhealthy, nil
		}), func(bound time.Duration) time.Duration { return bound })
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- scheduler.Run(ctx) }()
		synctest.Wait()
		if calls.Load() != 2 {
			t.Fatal("probe awal tidak lengkap")
		}
		time.Sleep(65 * time.Second)
		synctest.Wait()
		if calls.Load() != 2 {
			t.Fatal("failure tidak menerapkan backoff 60 detik dengan jitter 6 detik")
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if calls.Load() != 4 {
			t.Fatal("probe kedua tidak terjadwal pada 66 detik")
		}
		time.Sleep(119 * time.Second)
		synctest.Wait()
		if calls.Load() != 4 {
			t.Fatal("backoff kedua terlalu pendek")
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if calls.Load() != 6 {
			t.Fatal("backoff+jitter harus dibatasi 120 detik")
		}
		for _, r := range owner.View().Relays {
			if r.SOCKS5.Circuit != pool.CircuitOpen || r.HTTPS.Circuit != pool.CircuitOpen {
				t.Fatal("tiga gagal tetap membuka circuit meski interval backoff melebihi freshness")
			}
		}
		cancel()
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestProbeInvalidSchedule(t *testing.T) {
	owner := probeOwner(t)
	prober := probeFunc(func(context.Context, string, pool.HealthLayer) (pool.HealthState, error) {
		return pool.HealthHealthy, nil
	})
	for _, change := range []func(*Schedule){func(s *Schedule) { s.Interval = 0 }, func(s *Schedule) { s.Jitter = s.Interval }, func(s *Schedule) { s.Timeout = 0 }, func(s *Schedule) { s.Freshness = s.Timeout }, func(s *Schedule) { s.MaxBackoff = s.Interval - 1 }, func(s *Schedule) { s.MaxBackoff = time.Duration(1<<63 - 1) }} {
		schedule := fixtureSchedule()
		change(&schedule)
		if s, err := New(owner, schedule, prober, func(time.Duration) time.Duration { return 0 }); err == nil || s != nil {
			t.Fatal("schedule invalid diterima")
		}
	}
	for _, p := range []Prober{nil, probeFunc(nil)} {
		if s, err := New(owner, fixtureSchedule(), p, nil); err == nil || s != nil {
			t.Fatal("prober nil diterima")
		}
	}
	if s, err := New(nil, fixtureSchedule(), prober, nil); err == nil || s != nil {
		t.Fatal("owner nil diterima")
	}
	if s, err := New(owner, fixtureSchedule(), prober, nil); err == nil || s != nil {
		t.Fatal("jitter source wajib eksplisit")
	}
}

func TestProbeLateResultCannotReplaceNewObservation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		owner := probeOwner(t)
		gate := make(chan struct{})
		scheduler, err := New(owner, fixtureSchedule(), probeFunc(func(ctx context.Context, r string, l pool.HealthLayer) (pool.HealthState, error) {
			select {
			case <-gate:
				return pool.HealthHealthy, nil
			case <-ctx.Done():
				return pool.HealthUnhealthy, ctx.Err()
			}
		}), func(time.Duration) time.Duration { return 0 })
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- scheduler.Run(ctx) }()
		synctest.Wait()
		time.Sleep(time.Second)
		at := time.Now()
		failure := pool.ObservationView{State: pool.HealthUnhealthy, ObservedAt: at, ValidUntil: at.Add(time.Minute)}
		if err := owner.Observe("warp-1", pool.HealthSOCKS5, failure); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		close(gate)
		synctest.Wait()
		if o := owner.View().Relays[0].SOCKS5; o.State != pool.HealthUnhealthy || !o.ObservedAt.Equal(at) {
			t.Fatal("hasil probe lama menimpa observasi lebih baru")
		}
		cancel()
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestProbeTimeoutBoundAndCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		owner := probeOwner(t, "outbound_connections: 32", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1")
		var active, peak atomic.Int32
		scheduler, err := New(owner, fixtureSchedule(), probeFunc(func(ctx context.Context, r string, l pool.HealthLayer) (pool.HealthState, error) {
			n := active.Add(1)
			if n > peak.Load() {
				peak.Store(n)
			}
			defer active.Add(-1)
			<-ctx.Done()
			return pool.HealthHealthy, nil
		}), func(time.Duration) time.Duration { return 0 })
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- scheduler.Run(ctx) }()
		synctest.Wait()
		if _, ok := owner.TryProbe("warp-1", pool.HealthHTTPS); ok {
			t.Fatal("probe melebihi cap outbound")
		}
		if err := scheduler.Run(ctx); err == nil {
			t.Fatal("Run overlap diterima")
		}
		time.Sleep(5 * time.Second)
		synctest.Wait()
		if peak.Load() != 1 {
			t.Fatal("worker/probe melebihi cap satu")
		}
		v := owner.View()
		if v.Relays[0].SOCKS5.State != pool.HealthUnhealthy {
			t.Fatal("hasil sukses sesudah deadline dianggap sehat")
		}
		cancel()
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		if active.Load() != 0 {
			t.Fatal("worker tidak selesai saat shutdown")
		}
		for _, layer := range []pool.HealthLayer{pool.HealthSOCKS5, pool.HealthHTTPS} {
			release, ok := owner.TryProbe("warp-1", layer)
			if !ok {
				t.Fatal("slot outbound/probe bocor setelah cancel")
			}
			release()
		}
	})
}

func TestProbeZeroRunRejected(t *testing.T) {
	for _, scheduler := range []*Scheduler{nil, {}} {
		for _, canceled := range []bool{false, true} {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			if canceled {
				cancel()
			}
			err := scheduler.Run(ctx)
			cancel()
			if err == nil || err.Error() != "probe_invalid_run" {
				t.Fatal("scheduler zero/nil harus ditolak segera tanpa loop timer")
			}
		}
	}
}
