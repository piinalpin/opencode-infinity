package pool

import (
	"errors"
	"opencode-infinity/internal/config"
	"os"
	"strings"
	"sync"
	"testing"
)

func admissionConfig(t *testing.T, edits ...string) *config.Config {
	t.Helper()
	data, err := os.ReadFile("../../examples/infinity.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.NewReplacer("max_active_requests: 20", "max_active_requests: 2", "max_body_bytes: 10485760", "max_body_bytes: 1024", "body_buffer_bytes: 67108864", "body_buffer_bytes: 4096", "max_inflight: 5", "max_inflight: 1", "outbound_connections: 32", "outbound_connections: 2").Replace(string(data))
	text = strings.NewReplacer(edits...).Replace(text)
	c, err := config.Load(strings.NewReader(text), config.Sources{LookupEnv: func(name string) (string, bool) {
		v, ok := map[string]string{"INFINITY_GATEWAY_TOKEN": "synthetic-gateway-credential-12345", "INFINITY_ADMIN_TOKEN": "synthetic-admin-credential-67890", "INFINITY_UPSTREAM_KEY": "synthetic-provider-key"}[name]
		return v, ok
	}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestModelAdmissionRelayAndRelease(t *testing.T) {
	c := capacityConfig(t, "outbound_connections: 2", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1", "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:")
	a, err := newReadyPool(c)
	if err != nil {
		t.Fatal(err)
	}
	one, _ := a.Begin(100)
	two, _ := a.Begin(100)
	if admitFixtureRelay(one, "warp-1", 0) != nil {
		t.Fatal("relay valid ditolak")
	}
	if admitFixtureRelay(one, "warp-1", 0) == nil || admitFixtureRelay(two, "missing", 0) == nil || admitFixtureRelay(two, "warp-1", 0) == nil || admitFixtureRelay(two, "warp-2", 0) == nil {
		t.Fatal("slot relay/outbound terlampaui")
	}
	one.Release()
	if admitFixtureRelay(one, "warp-2", 0) == nil || admitFixtureRelay(two, "warp-2", 0) != nil {
		t.Fatal("release/reservasi relay salah")
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(two.Release)
	}
	wg.Wait()
	three, ok := a.Begin(4096)
	if !ok || admitFixtureRelay(three, "warp-1", 0) != nil {
		t.Fatal("resource tidak kembali penuh")
	}
	three.Release()
	if a.active != 0 || a.bytes != 0 || a.outbound != 0 {
		t.Fatal("resource bocor setelah release berulang")
	}
}

func TestModelAdmissionConcurrent(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 100 {
				r, ok := a.Begin(2048)
				if !ok {
					continue
				}
				admitFixtureRelay(r, "warp-1", 0)
				a.mu.Lock()
				if a.active > 2 || a.bytes > 4096 || a.outbound > 1 {
					t.Error("kapasitas konkuren terlampaui")
				}
				a.mu.Unlock()
				r.Release()
				r.Release()
			}
		})
	}
	wg.Wait()
	if a.active != 0 || a.bytes != 0 || a.outbound != 0 {
		t.Fatal("resource konkuren bocor")
	}
	for _, c := range []*config.Config{nil, {}} {
		if a, err := newReadyPool(c); err == nil || a != nil {
			t.Fatal("config invalid diterima")
		}
	}
	var zero Manager
	if _, ok := zero.Begin(1); ok {
		t.Fatal("zero admission menerima request")
	}
	var nilAdmission *Manager
	if _, ok := nilAdmission.Begin(1); ok {
		t.Fatal("nil admission menerima request")
	}
	var nilReservation *Reservation
	nilReservation.Release()
	if admitFixtureRelay(nilReservation, "warp-1", 0) == nil {
		t.Fatal("nil reservation menerima relay")
	}
}

func TestModelAdmissionGlobalBudget(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	one, ok := a.Begin(3000)
	if !ok {
		t.Fatal("reservasi pertama ditolak")
	}
	if _, ok := a.Begin(1097); ok {
		t.Fatal("budget agregat terlampaui")
	}
	two, ok := a.Begin(1096)
	if !ok {
		t.Fatal("batas budget tepat ditolak")
	}
	if _, ok := a.Begin(1); ok {
		t.Fatal("batas request terlampaui")
	}
	one.Release()
	one.Release()
	two.Release()
	three, ok := a.Begin(4096)
	if !ok {
		t.Fatal("budget/slot tidak dilepas")
	}
	three.Release()
	for _, n := range []int64{-1, 0, 4097} {
		if _, ok := a.Begin(n); ok {
			t.Fatal("reservasi invalid diterima")
		}
	}
	if a.active != 0 || a.bytes != 0 {
		t.Fatal("resource bocor")
	}
}

func TestFixedAdmissionAtomicPromotion(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	r, ok := a.Begin(100)
	if !ok {
		t.Fatal("persiapan valid ditolak")
	}
	defer r.Release()
	if a.pending != 1 || a.active != 0 || a.outbound != 0 || a.used["warp-1"] != 0 || a.bytes != 100 {
		t.Fatal("persiapan body sudah menjadi admission generasi parsial")
	}
	if admitFixtureRelay(r, "warp-1", 2048) != nil || a.pending != 0 || a.active != 1 || a.outbound != 1 || a.used["warp-1"] != 1 || a.streamBytes != 2048 {
		t.Fatal("admission tidak mengambil gateway/relay/outbound bersama")
	}
	r.Release()
	if a.pending != 0 || a.active != 0 || a.outbound != 0 || a.used["warp-1"] != 0 || a.bytes != 0 || a.streamBytes != 0 {
		t.Fatal("reservasi admitted tidak pulih")
	}
}

func TestFallbackRelaySwitchKeepsLeaseAtomic(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t,
		"upstreams:",
		"  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:",
		"primary_relay: warp-1\n        fallback_relays: []",
		"primary_relay: warp-1\n        fallback_relays: [warp-2]",
	))
	if err != nil {
		t.Fatal(err)
	}
	primary, _ := a.Begin(100)
	held, _ := a.Begin(100)
	if err := admitFixtureRelay(primary, "warp-1", 2048); err != nil {
		t.Fatal(err)
	}
	if err := admitFixtureRelay(held, "warp-2", 0); err != nil {
		t.Fatal(err)
	}
	if err := primary.SwitchFallback("primary", "coding-default", "warp-2"); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("fallback penuh harus ditolak, mendapat %v", err)
	}
	if primary.relay != "warp-1" || a.used["warp-1"] != 1 || a.used["warp-2"] != 1 {
		t.Fatal("fallback penuh mengubah reservasi primary")
	}
	held.Release()
	if err := primary.SwitchFallback("primary", "coding-default", "warp-1"); !errors.Is(err, ErrNoEligibleRelay) {
		t.Fatalf("relay yang bukan fallback diterima: %v", err)
	}
	if err := primary.SwitchFallback("primary", "coding-default", "warp-2"); err != nil {
		t.Fatalf("fallback eligible ditolak: %v", err)
	}
	if primary.relay != "warp-2" || a.used["warp-1"] != 0 || a.used["warp-2"] != 1 || a.active != 1 || a.outbound != 1 || a.streamBytes != 2048 {
		t.Fatal("perpindahan relay mengubah lease/stream accounting")
	}
	primary.Release()
	assertFixedLedger(t, a, true)
	var nilLease *Reservation
	if nilLease.SwitchFallback("primary", "coding-default", "warp-2") != ErrInvalidReservation {
		t.Fatal("lease nil mengubah relay")
	}
}

func TestFallbackRelaySwitchConcurrentCapacity(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t,
		"upstreams:",
		"  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:",
		"  - id: warp-1\n    proxy_url: socks5h://warp-1:1080\n    max_inflight: 1",
		"  - id: warp-1\n    proxy_url: socks5h://warp-1:1080\n    max_inflight: 2",
		"primary_relay: warp-1\n        fallback_relays: []",
		"primary_relay: warp-1\n        fallback_relays: [warp-2]",
	))
	if err != nil {
		t.Fatal(err)
	}
	reservations := make([]*Reservation, 2)
	for i := range reservations {
		reservations[i], _ = a.Begin(100)
		if err := admitFixtureRelay(reservations[i], "warp-1", 0); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	results := make(chan error, len(reservations))
	var workers sync.WaitGroup
	for _, reservation := range reservations {
		workers.Go(func() {
			<-start
			results <- reservation.SwitchFallback("primary", "coding-default", "warp-2")
		})
	}
	close(start)
	workers.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if !errors.Is(err, ErrCapacityExceeded) {
			t.Fatalf("hasil switch konkuren tak terduga: %v", err)
		}
	}
	if succeeded != 1 || a.used["warp-1"] != 1 || a.used["warp-2"] != 1 {
		t.Fatal("switch konkuren melampaui kapasitas atau kehilangan lease")
	}
	for _, reservation := range reservations {
		reservation.Release()
	}
	assertFixedLedger(t, a, true)
}

func TestFixedAdmissionRejectedPromotion(t *testing.T) {
	for _, tc := range []struct {
		name, relay  string
		stream, held int64
		edits        []string
		want         error
	}{
		{name: "unknown", relay: "missing", want: ErrNoEligibleRelay},
		{name: "relay", relay: "warp-1", stream: 1024, want: ErrCapacityExceeded},
		{name: "outbound", relay: "warp-2", stream: 1024, edits: []string{"outbound_connections: 2", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1"}, want: ErrCapacityExceeded},
		{name: "stream", relay: "warp-2", held: 32768, stream: 1, want: ErrCapacityExceeded},
		{name: "negative-stream", relay: "warp-2", stream: -1, want: ErrInvalidReservation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			edits := append([]string{"stream_buffer_bytes: 8388608", "stream_buffer_bytes: 32768", "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:"}, tc.edits...)
			a, err := newReadyPool(capacityConfig(t, edits...))
			if err != nil {
				t.Fatal(err)
			}
			held, _ := a.Begin(100)
			defer held.Release()
			if err := admitFixtureRelay(held, "warp-1", tc.held); err != nil {
				t.Fatal(err)
			}
			r, _ := a.Begin(100)
			defer r.Release()
			if err := admitFixtureRelay(r, tc.relay, tc.stream); !errors.Is(err, tc.want) {
				t.Fatalf("kategori penolakan %v, ingin %v", err, tc.want)
			}
			if a.pending != 1 || a.active != 1 || a.bytes != 200 || a.outbound != 1 || a.used["warp-1"] != 1 || a.used["warp-2"] != 0 || a.streamBytes != tc.held || r.admitted || r.relay != "" || r.streamBytes != 0 {
				t.Fatal("promosi gagal menyisakan reservasi parsial atau mengganti primary")
			}
			r.Release()
			held.Release()
			next, ok := a.Begin(4096)
			if !ok || admitFixtureRelay(next, "warp-2", 32768) != nil {
				t.Fatal("seluruh kapasitas gagal pulih setelah penolakan")
			}
			next.Release()
			assertFixedLedger(t, a, true)
		})
	}
}

func TestFixedAdmissionLocalActivation(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	local, _ := a.Begin(100)
	defer local.Release()
	if local.AdmitLocal() != nil || a.pending != 0 || a.active != 1 || a.outbound != 0 || a.streamBytes != 0 {
		t.Fatal("katalog tidak diaktifkan lokal atau mengambil slot jaringan")
	}
	if local.AdmitLocal() != ErrInvalidReservation || admitFixtureRelay(local, "warp-1", 0) != ErrInvalidReservation || local.ReserveStream(1) {
		t.Fatal("lease katalog dipromosikan dua kali atau mengambil stream")
	}
	pending, _ := a.Begin(3996)
	defer pending.Release()
	if _, ok := a.Begin(1); ok {
		t.Fatal("pending bersama katalog melewati batas global/buffer")
	}
	if admitFixtureRelay(pending, "warp-1", 0) != nil || a.pending != 0 || a.active != 2 {
		t.Fatal("generasi tidak berbagi kapasitas dengan katalog")
	}
	local.Release()
	pending.Release()
	if local.AdmitLocal() != ErrInvalidReservation {
		t.Fatal("lease released diaktifkan ulang")
	}
	var zero Reservation
	var nilLease *Reservation
	if zero.AdmitLocal() != ErrInvalidReservation || nilLease.AdmitLocal() != ErrInvalidReservation {
		t.Fatal("lease invalid diterima")
	}
	assertFixedLedger(t, a, true)
}

func assertFixedLedger(t *testing.T, a *Manager, empty bool) {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	var total int64
	for relay, used := range a.used {
		if used < 0 || used > a.relays[relay] {
			t.Errorf("kapasitas relay terlampaui: %s", relay)
		}
		total += used
	}
	if a.pending < 0 || a.active < 0 || a.pending+a.active > a.maxActive || total+a.probeCount != a.outbound || total > a.active || a.outbound < 0 || a.outbound > a.maxOutbound || a.bytes < 0 || a.bytes > a.maxBytes || a.streamBytes < 0 || a.streamBytes > a.maxStreamBytes {
		t.Error("ledger global/relay/outbound/buffer tidak konsisten")
	}
	if empty && (a.pending != 0 || a.active != 0 || a.bytes != 0 || a.outbound != 0 || a.streamBytes != 0) {
		t.Error("ledger tidak kosong setelah semua lease selesai")
	}
}

func TestFixedAdmissionConcurrent(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t, "max_active_requests: 2", "max_active_requests: 4", "stream_buffer_bytes: 8388608", "stream_buffer_bytes: 32768", "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:"))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := range 20 {
		workers.Go(func() {
			<-start
			for range 100 {
				r, ok := a.Begin(100)
				if !ok {
					continue
				}
				assertFixedLedger(t, a, false)
				if i%3 == 0 {
					r.AdmitLocal()
				} else {
					admitFixtureRelay(r, []string{"warp-1", "warp-2"}[i%2], 16384)
				}
				assertFixedLedger(t, a, false)
				r.Release()
				r.Release()
			}
		})
	}
	close(start)
	workers.Wait()
	assertFixedLedger(t, a, true)
	for range 100 {
		r, ok := a.Begin(4096)
		if !ok {
			t.Fatal("buffer penuh tidak pulih")
		}
		start := make(chan struct{})
		var race sync.WaitGroup
		for _, action := range []func(){func() { admitFixtureRelay(r, "warp-1", 32768) }, func() { r.AdmitLocal() }, func() { r.ReserveStream(32768) }, r.Release, r.Release} {
			race.Go(func() { <-start; action() })
		}
		close(start)
		race.Wait()
		r.Release()
		assertFixedLedger(t, a, true)
	}
}

func TestFixedAdmissionPendingCannotReserveStream(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	r, ok := a.Begin(100)
	if !ok {
		t.Fatal("persiapan valid ditolak")
	}
	defer r.Release()
	if r.ReserveStream(32768) {
		t.Fatal("pending mengambil buffer stream sebelum admission gateway/relay")
	}
}

func TestGatewayStreamBudget(t *testing.T) {
	a, err := newReadyPool(capacityConfig(t, "max_active_requests: 2", "max_active_requests: 3", "stream_buffer_bytes: 8388608", "stream_buffer_bytes: 65536", "outbound_connections: 2", "outbound_connections: 3", "max_inflight: 1", "max_inflight: 3"))
	if err != nil {
		t.Fatal(err)
	}
	one, _ := a.Begin(100)
	two, _ := a.Begin(100)
	three, _ := a.Begin(100)
	for _, r := range []*Reservation{one, two, three} {
		if err := admitFixtureRelay(r, "warp-1", 0); err != nil {
			t.Fatal(err)
		}
	}
	if !one.ReserveStream(32768) || !two.ReserveStream(32768) {
		t.Fatal("budget stream tepat ditolak")
	}
	if three.ReserveStream(1) || one.ReserveStream(1) {
		t.Fatal("budget stream terlampaui/reservasi ganda")
	}
	one.Release()
	if one.ReserveStream(1) || !three.ReserveStream(32768) {
		t.Fatal("release/reservasi stream salah")
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(two.Release)
		wg.Go(three.Release)
	}
	wg.Wait()
	for range 20 {
		wg.Go(func() {
			for range 100 {
				r, ok := a.Begin(100)
				if !ok {
					continue
				}
				admitFixtureRelay(r, "warp-1", 0)
				r.ReserveStream(32768)
				r.Release()
				r.Release()
			}
		})
	}
	wg.Wait()
	for _, n := range []int64{0, -1, 65537} {
		r, _ := a.Begin(100)
		if err := admitFixtureRelay(r, "warp-1", 0); err != nil {
			t.Fatal(err)
		}
		if r.ReserveStream(n) {
			t.Fatal("ukuran stream invalid diterima")
		}
		r.Release()
	}
	var zero Reservation
	var nilLease *Reservation
	if zero.ReserveStream(1) || nilLease.ReserveStream(1) {
		t.Fatal("lease zero menerima stream")
	}
	if a.active != 0 || a.bytes != 0 {
		t.Fatal("resource stream bocor")
	}
	final, ok := a.Begin(4096)
	if !ok || admitFixtureRelay(final, "warp-1", 0) != nil || !final.ReserveStream(65536) {
		t.Fatal("budget stream tidak kembali penuh")
	}
	final.Release()
}
