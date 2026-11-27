# Liveness dan readiness dasar

Tanggal: 26 November 2026  
Task: [EP-02-T06](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-t06)  
Status: komponen dan fixture lokal tersedia; penerimaan produk terintegrasi tetap parsial

[D-025](DECISIONS.md#d-025-liveness-dan-readiness-dasar-independen-ep-02-t06) menyetujui pekerjaan independen di atas [konfigurasi](konfigurasi.md), [akses](akses-dan-credential.md) dan [boundary error S05](error-dan-batas-request.md). T06 tidak meluluskan dependensi S05/T01/EP-01-T06 atau gerbang M0. Tidak ada listener produk, scheduler/probe produksi, provider/WARP nyata, atau perubahan deployment.

## API dan ownership

Pembaruan 27 November 2026 melalui [S02/D-027](eligibility-relay.md) mengganti owner `Health` dan `Admission` terpisah dengan `pool.Manager`. `pool.New(config, clock, policy)` membentuk owner, `gateway.New` membentuk coordinator serta owner dari satu snapshot, dan `httpapi.NewComponents(config, dispatcher, build, clock, policy)` mengembalikan handler `Models`, `Health` dan `Pool` yang sama. `status.NewHealth(build).Snapshot(view)` tetap membentuk proyeksi teredaksi.

Constructor/read tidak menjalankan probe. Scheduler fixture dijalankan eksplisit di luar handler; availability provider tanpa sumber aman tetap unknown. State hanya menyimpan ID dan metadata typed, bukan URL/key/raw error adapter. Tidak ada hot reload melalui perubahan object caller.

`Observe`, `ObserveModel`, `SetMode`, `SetAccepting` dan lease berkoordinasi dengan satu mutex. `SetAccepting(false)` kini menolak admission generasi baru; mode draining/disabled juga menolak assignment tanpa memutus stream admitted. Cooldown/binding/lifecycle penuh tetap task berikutnya. Bukti historis T06 di bawah tidak otomatis meluluskan penerimaan produk terintegrasi.

## Endpoint privat

| Method | Path | Hasil |
|---|---|---|
| GET/HEAD | `/health/live` | 200 selama komponen handler berfungsi, terlepas dari relay/accepting |
| GET/HEAD | `/health/ready` | 200 bila scope readiness terpenuhi; 503 bila belum siap |

Kedua endpoint memakai `RequireAdmin` paling luar. Missing/invalid/wrong-role/Authorization ambigu menghasilkan `401 invalid_admin_token` dengan challenge realm admin sebelum body atau snapshot dibaca. Token gateway/upstream bukan token admin; query dan cookie tidak menjadi tempat token. Handler health tetap terpisah dari handler model, dan handler health tidak melayani endpoint model atau `/admin/status` dashboard lengkap.

Respons GET berisi `health` dan `request_id`; saat not-ready, respons 503 juga mempunyai `error.message`, `error.type: gateway_unavailable`, `error.code: gateway_not_ready`. Jadi error tetap memakai shape boundary S05 dengan snapshot diagnostik tambahan. `X-Request-ID` server cocok dengan body dan tidak menerima ID client. `X-Infinity-Error-Source: gateway`, JSON, `no-store` dan `nosniff` dipasang. HEAD hanya mengirim status/header pada wire.

Method salah pada path health menghasilkan `405 method_not_allowed` dengan `Allow: GET, HEAD`; path unknown menghasilkan `404 endpoint_not_found`. Path noncanonical ditolak `400 invalid_request` tanpa redirect/refleksi query, memakai guard S05. Gagal membentuk snapshot menghasilkan `503 health_snapshot_unavailable` sebelum commit; response tidak berupa JSON terpotong. Pemisahan handler dan token terbukti, sedangkan binding loopback/Tailscale tetap harus dibuktikan saat listener aplikasi dirakit.

## Arti readiness

Scope tetap `default_primary_transport`:

1. Konfigurasi sudah lolos loader dan kondisi proses `accepting` bernilai true.
2. Sedikitnya satu **primary relay** milik mapping model pada default upstream mempunyai observasi SOCKS5 dan HTTPS keluar yang belum kedaluwarsa. Sejak S02, mode harus serving dan kedua circuit closed sesuai policy eksplisit; outcome gagal terbaru di bawah ambang tidak langsung membuka circuit.
3. Relay fallback, relay yang hanya terdaftar pada upstream, atau primary upstream nondefault tidak menggantikan primary mapping tersebut. Tidak disyaratkan seluruh model default sehat.

Proses hidup, SOCKS5 terjangkau, HTTPS keluar, serta availability upstream/model adalah lapisan berbeda. Update SOCKS5 tidak mengubah HTTPS. Field agregat historis `upstream_model` tetap `unknown`; sejak S02 observasi scoped ada pada `models[].availability`. Readiness tidak membuktikan provider/model tersedia, quota/auth upstream valid, IP tertentu, ataupun bahwa request selanjutnya pasti diterima. Saturasi kapasitas tidak termasuk scope diagnostik ini; request generasi yang masih memegang satu-satunya slot tidak membuat read health mengambil/melepas slotnya.

Startup tanpa observasi menghasilkan live 200 dan ready 503. Kondisi relay unavailable, unknown, expired, hanya satu lapisan sehat, atau accepting=false tetap not-ready. Liveness tidak memicu probe/generasi atau ikut gagal hanya karena relay unavailable. Endpoint membaca state yang sudah dikumpulkan, tanpa socket atau pekerjaan probe baru meskipun dipanggil berulang.

## Observasi, revisi dan waktu

Sumber server tepercaya mengirim `ObservationView` untuk satu relay terdaftar dan satu lapisan (`HealthSOCKS5` atau `HealthHTTPS`). Outcome update hanya healthy/unhealthy; waktu `observed_at` dan `valid_until` eksplisit. Clock harus thread-safe dan tidak mundur; deadline/freshness produksi tidak ditebak oleh T06. Waktu internal dapat mempertahankan clock monotonic Go; proyeksi wire memakai UTC.

- Waktu zero/tidak serializable, observasi masa depan, expiry yang tidak melewati observasi/waktu sekarang, relay/lapisan/outcome unknown, serta update lebih tua ditolak dengan kode tetap tanpa input mentah.
- Observasi dengan timestamp sama tetapi isi/expiry berbeda ditolak. Duplikat identik yang masih valid bersifat no-op. Success lama tidak dapat menimpa failure baru atau memperpanjang expiry melalui replay timestamp.
- Tepat pada `valid_until`, effective state menjadi `stale` dan tidak dianggap sehat. Sumber tidak dianggap baru hanya karena endpoint dibaca lagi.
- `revision` dan `state_changed_at` menunjukkan update state tersimpan yang diterima, termasuk observasi baru dengan outcome yang sama; accepting dengan nilai sama dan duplikat identik tidak menaikkannya.
- `snapshot_at` menunjukkan waktu evaluasi hasil baca. `observed_at`/`valid_until` tetap menunjukkan waktu observasi dan batasnya. Expiry dapat mengubah effective readiness tanpa menaikkan revisi atau mengubah `state_changed_at`; kedua waktu itu bukan `ready_since`.

Snapshot memisahkan versi aplikasi dalam `application.version`/`application.revision` dari `revision` state health. Versi aplikasi yang belum tersedia adalah `unknown`, bukan versi schema YAML `1`. Metadata build hanya menerima versi semver bounded atau `devel`/`unknown`, serta revisi hex 40/64 karakter atau `unknown`. Tidak ada dump `debug.BuildInfo`, environment, Config atau error adapter. Field observasi yang belum tersedia mempunyai state unknown dan waktu null.

## Batas state dan pembacaan

Registry tetap dibatasi loader (maksimal 128 relay). Update untuk ID unknown tidak menambah collection. State disalin di bawah lock owner; evaluasi, proyeksi, encode serta write berlangsung setelah lock dilepas. Caller menerima slice/value terpisah, sehingga perubahan pada view tidak menulis state.

Budget memakai konfigurasi existing, tanpa schema/default numerik baru:

- Sejak S02, state owner/job/circuit memakai bound `4096 + 1024 × relay + 1024 × model` byte.
- Bound snapshot memakai `4096 + 2048 × relay + 1024 × model`; monitoring mencakup `parallel_status × 3 × bound_snapshot + worker × 4096` byte.
- Constructor menolak budget kurang; worker dibatasi parallel_probes dan jumlah job finite. Rincian current policy serta bukti maksimum gabungan 128 relay/1.024 model ada pada [kontrak S02](eligibility-relay.md#scheduler-dan-batas-resource).
- Pembacaan dibatasi `parallel_status` sampai write selesai; overload tetap 429 tanpa queue. Copy/proyeksi/encode/write tidak menahan lock dispatch.

Accounting tersebut konservatif untuk komponen ini, bukan bukti RSS proses atau resource accounting gabungan semua fitur mendatang. Deadline koneksi/client yang tidak selesai tetap tanggung jawab runtime listener. T06 historis tidak membuat timer/scheduler; S02 menambah scheduler fixture terpisah yang hanya berjalan melalui Run eksplisit, tanpa histori observasi.

## Bukti penerimaan

| AC | Bukti otomatis | Batas |
|---|---|---|
| AC1 | `TestHealthAdminAndReadiness`, `TestHealthWireAndNoDispatch`: read GET/HEAD berulang, ledger tetap; observasi fixture dikumpulkan eksplisit | Bukan startup/probe runtime |
| AC2 | `TestHealthLayersAndPrimary`, `TestHealthDefaultScope`: dua lapisan, actual primary/default, at-least-one; wire ready 200 setelah observasi fixture | Bukan eligibility/dispatch terpadu atau model availability |
| AC3 | `TestHealthFreshnessAndOrdering`, `TestHealthAccepting`, wire relay ditutup: ready 503, live 200, expiry/read purity | Bukan drain/container lifecycle |
| AC4 | `TestHealthProjection`, `TestHealthConcurrentSnapshots`, `TestHealthMaximumRegistrySnapshot`, tes admin/error/budget/slow writer dan wire | Build metadata dari server; private binding dan RSS belum dibuktikan |

Fixture SOCKS5 melakukan greeting eksplisit; fixture HTTPS menerima GET non-generasi untuk observasi jalur keluar. Pengumpulan itu dilakukan oleh tes di luar handler health, bukan otomatis oleh endpoint. Tes kemudian memverifikasi read tidak menambah koneksi/dispatch/generasi. Saat satu request generasi memegang satu-satunya lease, health tetap terbaca dan request generasi kedua tetap ditolak 429 sampai lease pertama selesai.

Sumber tes: `internal/pool/health_test.go`, `internal/status/health_test.go`, `internal/httpapi/health_test.go`, dan `tests/integration/health_test.go`, dengan token/key sintetis. Pemeriksaan pada Go `go1.27.4-X:nodwarf5 linux/amd64`:

- `go test -count=1 ./...`: lulus seluruh 11 package, termasuk regresi S02–S05. Native WARP opt-in tidak diaktifkan.
- `go vet ./...`: lulus; `gofmt -l internal tests/integration` kosong; `git diff --check` lulus.
- `go test -race -count=1 ./...`: lulus seluruh 11 package.
- `go test -race -count=20 ./internal/pool ./internal/status ./internal/httpapi ./tests/integration -run 'TestHealth|TestModelAdmissionConcurrent|TestGatewayStreamBackpressureAndLease'`: 20 pengulangan per tes terpilih pada empat package, tanpa kegagalan/race. Ini verifikasi konkurensi/lifecycle, bukan benchmark.

Sembilan siklus TDD mencakup state awal, lapisan/primary, freshness/order, accepting, budget state, proyeksi metadata, handler admin, error routing, dan budget/parallel monitoring. Coverage tambahan menguji registry maksimum, isolasi handler, snapshot konkuren, serialize failure serta wire fixture. Review strict inline selesai dengan verdict **APPROVE**, tanpa temuan terbuka. Tier high tanpa eskalasi; tiga subagent analisis, satu putaran review, tanpa deviasi rencana.

Scope yang belum terbukti: sumber/scheduler/probe produksi, default freshness/backoff, integrasi health/admission/cooldown EP-03, lifecycle drain EP-04, listener privat/deployment, OpenCode/provider/WARP, RSS/performa serta kelulusan M0/MVP. Health status ini bukan dashboard operasional lengkap EP-05.
