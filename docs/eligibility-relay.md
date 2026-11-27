# Eligibility relay, health dan mode operasi

Tanggal: 27 November 2026  
Task: [EP-03-S02](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02)  
Status: komponen dan fixture lokal independen; penerimaan produk terintegrasi tetap parsial

[D-027](DECISIONS.md#d-027-eligibility-health-dan-mode-independen-ep-03-s02) menyetujui rencana lokal ini. Bukti tidak meluluskan dependensi produk S01/T06/M0. Tidak ada listener, transport runtime, provider/WARP nyata, generasi berbayar, dashboard atau perubahan deployment.

## API dan ownership

`pool.New(config, clock, policy)` menghasilkan satu `Manager`. Owner ini menyimpan observasi transport, availability mapping, circuit, mode operator, accepting, dan counter reservasi dalam satu mutex. `gateway.New(config, dispatcher, clock, policy)` membentuk coordinator serta owner itu dari snapshot yang sama. `httpapi.NewComponents(config, dispatcher, build, clock, policy)` menghasilkan `Models`, `Health` dan `Pool`; handler model serta handler admin memakai owner yang sama. Constructor lama `NewAdmission`, `pool.NewHealth`, `httpapi.NewAPI` dan `NewHealthAPI` digantikan, sehingga tidak ada jalur capacity-only atau pemeriksaan health opsional.

Konfigurasi loader/catalog, credential boundary, guard gateway/admin, dan kontrak response body S01 tetap digunakan. Pool menyimpan identifier konfigurasi, bukan credential atau URL. Operasi mutasi hanya melalui `Observe`, `ObserveModel`, `SetMode`, `SetAccepting`, `ExtendRelayCooldown` (S03), dan lease pool; handler/prober tidak menulis map/counter langsung. Clock harus thread-safe dan tidak mundur. Scheduler dan clock owner menggunakan basis waktu yang sama; fixture scheduler memakai `time.Now`, termasuk waktu virtual `testing/synctest`.

Sejak [S03/D-028](least-inflight.md), `Reservation.Admit(upstream, alias, streamBytes) (relayID, error)` menerima scope hasil resolve registry server, memilih primary fixed atau kandidat least tervalidasi, memeriksa eligibility serta kapasitas, lalu melakukan commit pending/admitted/gateway/relay/outbound/stream di bawah mutex yang sama. Tidak membaca readiness/snapshot sebagai precheck. S03 menambahkan least-inflight opt-in serta cooldown relay eksplisit; interpretasi 429/cooldown provider, fallback/retry dan binding tetap task berikutnya.

## Lapisan, circuit dan mode

Setiap relay memiliki observasi SOCKS5 dan HTTPS keluar yang independen. Setiap mapping fixed memiliki availability upstream ID + alias; tidak dialokasikan matrix relay × model/fallback. Observasi latest outcome (`unknown`, `healthy`, `unhealthy`, `stale`) terpisah dari circuit (`checking`, `closed`, `open`). Snapshot tidak menyamakan satu kegagalan terbaru dengan circuit terbuka jika ambang belum tercapai.

Policy threshold wajib diberikan eksplisit. Profil fixture memakai dua sukses berurutan untuk menutup circuit startup/open dan tiga gagal berurutan untuk membukanya. Outcome berlawanan mereset streak; streak dibatasi pada ambang agar tidak overflow. Duplikat identik tidak menambah streak/revisi. Timestamp masa depan, lama, konflik, expiry invalid serta ID/layer/state tidak sah ditolak tanpa menambah state.

Transport unknown/expired atau circuit checking/open tidak eligible. Tepat `valid_until`, transport menjadi efektif stale tanpa mengubah observasi/revisi tersimpan. Sesudah expiry, streak sukses dimulai ulang. Streak gagal tetap dihitung jika observasi sebelumnya gagal: backoff yang lebih panjang daripada freshness tidak mencegah tiga kegagalan membuka circuit. Hasil sukses baru tetap memerlukan ambang pemulihan.

Mode operator hanya `serving`, `draining` atau `disabled`. Probe tidak mengubah mode. Startup serving tidak berarti transport sehat. Draining/disabled dan accepting=false menolak generasi baru; stream/lease yang sudah admitted tetap berjalan sampai selesai atau cancellation/deadline yang sudah berlaku. Ini bukan implementasi shutdown atau maintenance identitas EP-04.

Availability tanpa pemeriksaan aman tetap **unknown** dan tidak memblokir transport sehat. Availability circuit open yang masih fresh memblokir mapping itu saja; expiry kembali efektif unknown, bukan healthy. Tidak ada probe generasi otomatis untuk mengisi availability. `ObserveModel` membuktikan scope dan threshold pada data fixture eksplisit; ketersediaan provider/model nyata belum dibuktikan.

Readiness tetap `default_primary_transport`: accepting, mode serving dan dua circuit transport closed/fresh pada sedikitnya satu primary mapping default. Model availability dan saturasi kapasitas tidak mengubah readiness menjadi janji generasi pasti berhasil. Field agregat historis `upstream_model` tetap unknown; daftar `models[].availability` adalah observasi scoped yang aktual.

Pembaruan S03 menampilkan deklarasi routing/stateless dan cooldown_until/cooldown_active yang terpisah dari health/circuit. Deadline relay berlaku untuk fixed/least, hanya diperpanjang, tidak dihapus probe, dan expiry tidak mengubah revisi. Readiness tetap primary transport; kapasitas/cooldown tidak menjadi klaim kesiapan generasi. [Bukti dan batas S03](least-inflight.md) tetap terpisah dari penerimaan produk.

## Scheduler dan batas resource

`health.New(owner, schedule, prober, jitter)` tidak menjalankan jaringan. `Run(context)` dipanggil eksplisit oleh perakitan server. Setiap relay mempunyai dua job transport tetap. Worker dibatasi `min(parallel_probes, 2 × jumlah_relay)`; satu target tidak overlap. Tidak ada goroutine per tick, riwayat observasi, atau antrean tick tak terbatas. Ketika cap outbound penuh, jadwal dicoba pada interval berikutnya tanpa membuka socket.

Profil fixture eksplisit:

| Parameter | Nilai |
|---|---|
| Interval | 30 detik |
| Jitter | ±6 detik, setara ±20% interval |
| Timeout per probe | 5 detik |
| Freshness | 90 detik sejak mulai probe |
| Threshold | 3 gagal / 2 sukses |
| Backoff | Eksponensial setelah kegagalan; delay termasuk jitter maksimal 120 detik |

Keberhasilan mereset backoff ke interval. Timeout menghasilkan outcome gagal; cancellation shutdown tidak membuat generasi pengganti. Hasil diberi timestamp mulai probe sehingga hasil lama yang selesai belakangan tidak menimpa observasi lebih baru. Owner menolak update kalah/expired; scheduler tetap melanjutkan jadwal. Tes jaringan memakai interval 20 ms agar fixture cepat, sedangkan profil 30 detik dibuktikan dengan waktu virtual. Angka tersebut bukan default produk; parameter final tetap [TBD pada task pemilik](M0-gerbang-dan-pengukuran.md#6-input-dan-default-yang-belum-final).

`TryProbe(relay, layer)` mengambil slot probe dan outbound dari ledger yang juga dipakai generasi. Release idempotent dilakukan setelah adapter selesai dan jaringan ditutup. Adapter tepercaya wajib menghormati Context dan tidak meninggalkan socket idle. Scheduler membatalkan pekerjaan dan menunggu worker saat shutdown; komponen Go tidak dapat memutus paksa callback yang mengabaikan Context. Fixture menonaktifkan keepalive dan membuktikan cleanup; accounting seluruh socket/idle runtime tetap membutuhkan adapter produk.

Adapter fixture melakukan greeting SOCKS5 serta GET HTTPS non-generasi melalui proxy dengan hostname tujuan pada CONNECT, verifikasi TLS hostname, tanpa redirect/direct fallback/retry. Target berasal dari konfigurasi fixture server. Pembacaan health tidak menjadwalkan probe atau melakukan I/O upstream, termasuk saat dibaca berulang.

State tetap finite pada 128 relay dan 1.024 mapping model. Accounting konservatif:

- State owner/job/circuit: `4096 + 1024 × relay + 1024 × model + 64 × referensi_upstream_relay` byte sejak S03.
- Bound snapshot: `4096 + 2048 × relay + 1024 × model` byte.
- Monitoring: `parallel_status × 3 × bound_snapshot + worker × 4096` byte untuk copy/proyeksi/encode dan scratch worker.

Constructor pool mandiri juga menolak monitoring yang tidak cukup untuk scratch worker; assembly HTTP memeriksa total scratch dan snapshot. Constructor menolak budget yang tidak cukup. Perkalian memakai ukuran registry yang sudah dibatasi loader; pemeriksaan monitoring memakai pembagian terhadap budget. Snapshot disalin di bawah lock, lalu proyeksi/encoding/write dilakukan di luar lock. Snapshot maksimum gabungan 128 relay dan 1.024 model memakai budget fixture eksplisit yang lebih besar, tanpa mengubah contoh konfigurasi/default. Ledger bukan hasil pengukuran RSS.

## Error dan bukti penerimaan

Persiapan body tetap bounded sebelum parsing. Setelah scope diketahui, eligibility diperiksa sebelum kapasitas promotion: ineligible menghasilkan `503 no_eligible_relay`, sedangkan relay sehat yang penuh menghasilkan `429 gateway_capacity_exceeded`. Batas pending/body yang diperiksa lebih awal tetap dapat menghasilkan 429 sebelum mapping diketahui. Penolakan tidak melakukan dispatch; tidak ada internet langsung.

| AC child | Bukti lokal | Batas penerimaan produk |
|---|---|---|
| AC1 | `TestProbeScheduleRecoveryAndReadPurity`, `TestEligibilityModelScope`, `TestEligibilityScheduledWireAndDrain`: SOCKS5/HTTPS terjadwal dan status model terpisah | Availability provider tetap unknown tanpa sumber aman; bukan scheduler runtime aktif |
| AC2 | `TestEligibilityThresholds`, `TestEligibilityLinearizationAndReplay`, `TestProbeBackoffJitter`, `TestProbeLateResultCannotReplaceNewObservation`, `TestProbeInvalidSchedule` | Policy fixture eksplisit; default produksi TBD |
| AC3 | `TestEligibilityModeAndActiveLease`, `TestEligibilityProjectionAndSharedOwner`, `TestEligibilityScheduledWireAndDrain`: probe pulih tidak mengaktifkan mode; SSE admitted sampai DONE | Bukan lifecycle shutdown/maintenance penuh |
| AC4 | `TestEligibilityUnknownRejects`, `TestEligibilityScheduledWireAndDrain`, `TestEligibilityProbeOutboundAndModelScope`: 503 pada primary default/nondefault tanpa tambahan dial/CONNECT/POST/direct | Hanya fixture lokal; bukan WARP/provider |

Tes tambahan membuktikan ledger probe bersama, 429/recovery, replay dan expiry, copy/redaksi snapshot, budget kurang, registry maksimum, timeout/cancel dan linearization. Source: `internal/pool/*_test.go`, `internal/health/probe_test.go`, `internal/httpapi/health_test.go`, dan `tests/integration/eligibility_test.go`. Tes lama S01/T06 mempertahankan assertion; helper positif memberi observasi sehat secara eksplisit dengan policy fixture, bukan menganggap registry sehat.

Verifikasi pada Go `go1.27.4-X:nodwarf5 linux/amd64`:

- `go test -count=1 ./...`: 12 package root module lulus setelah scheduler ditambahkan.
- `go test -race -count=1 ./...`: seluruh 12 package lulus.
- `go test -race -count=50 ./internal/pool ./internal/health`: 50 pengulangan lulus tanpa kegagalan/race.
- `go test -race -count=20 ./tests/integration -run 'TestEligibility|TestHealth|TestFixedAdmission'`: 20 pengulangan setiap tes terpilih lulus tanpa kegagalan/race.
- `go vet ./...`: lulus; `gofmt -l internal tests/integration`: kosong.

Satu run antara saat TDD mengalami kegagalan cleanup fixture lama `TestMalformedSOCKS5` dengan `shutdown: transport endpoint is not connected`; fixture tersebut tidak diubah. Run final normal/race lulus. Ini tidak dipakai untuk menurunkan assertion atau menyatakan RSS/performa, cancellation target S06, OpenCode/provider/WARP maupun M0/MVP lulus.

Dokumentasi timer/cancellation dan `testing/synctest` diperiksa melalui Context7 untuk [source Go 1.27.4](https://github.com/golang/go/tree/go1.27.4/src/testing/synctest) sesuai pin `go.mod`; tidak menambah dependency.
