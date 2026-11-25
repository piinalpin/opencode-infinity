package pool

import (
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
	c := admissionConfig(t, "outbound_connections: 2", "outbound_connections: 1", "parallel_probes: 2", "parallel_probes: 1", "upstreams:", "  - id: warp-2\n    proxy_url: socks5h://warp-2:1080\n    max_inflight: 1\nupstreams:")
	a, err := NewAdmission(c)
	if err != nil {
		t.Fatal(err)
	}
	one, _ := a.Begin(100)
	two, _ := a.Begin(100)
	if !one.Assign("warp-1") {
		t.Fatal("relay valid ditolak")
	}
	if one.Assign("warp-1") || two.Assign("missing") || two.Assign("warp-1") || two.Assign("warp-2") {
		t.Fatal("slot relay/outbound terlampaui")
	}
	one.Release()
	if one.Assign("warp-2") || !two.Assign("warp-2") {
		t.Fatal("release/reservasi relay salah")
	}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(two.Release)
	}
	wg.Wait()
	three, ok := a.Begin(4096)
	if !ok || !three.Assign("warp-1") {
		t.Fatal("resource tidak kembali penuh")
	}
	three.Release()
	if a.active != 0 || a.bytes != 0 || a.outbound != 0 {
		t.Fatal("resource bocor setelah release berulang")
	}
}

func TestModelAdmissionConcurrent(t *testing.T) {
	a, err := NewAdmission(admissionConfig(t))
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
				r.Assign("warp-1")
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
		if a, err := NewAdmission(c); err == nil || a != nil {
			t.Fatal("config invalid diterima")
		}
	}
	var zero Admission
	if _, ok := zero.Begin(1); ok {
		t.Fatal("zero admission menerima request")
	}
	var nilAdmission *Admission
	if _, ok := nilAdmission.Begin(1); ok {
		t.Fatal("nil admission menerima request")
	}
	var nilReservation *Reservation
	nilReservation.Release()
	if nilReservation.Assign("warp-1") {
		t.Fatal("nil reservation menerima relay")
	}
}

func TestModelAdmissionGlobalBudget(t *testing.T) {
	a, err := NewAdmission(admissionConfig(t))
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

func TestGatewayStreamBudget(t *testing.T) {
	a, err := NewAdmission(admissionConfig(t, "max_active_requests: 2", "max_active_requests: 3", "stream_buffer_bytes: 8388608", "stream_buffer_bytes: 65536"))
	if err != nil {
		t.Fatal(err)
	}
	one, _ := a.Begin(100)
	two, _ := a.Begin(100)
	three, _ := a.Begin(100)
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
				r.ReserveStream(32768)
				r.Release()
				r.Release()
			}
		})
	}
	wg.Wait()
	for _, n := range []int64{0, -1, 65537} {
		r, _ := a.Begin(100)
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
	if !ok || !final.ReserveStream(65536) {
		t.Fatal("budget stream tidak kembali penuh")
	}
	final.Release()
}
