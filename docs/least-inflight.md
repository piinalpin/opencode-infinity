# Least-inflight untuk model stateless

Tanggal: 27 November 2026  
Task: [EP-03-S03](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s03)  
Status: komponen dan fixture lokal independen; penerimaan produk terintegrasi tetap parsial

[D-028](DECISIONS.md#d-028-least-inflight-stateless-dan-cooldown-minimum-ep-03-s03) menyetujui konfigurasi stateless, pemilihan relay atomik, dan cooldown minimum untuk pembuktian lokal. S03 tidak meluluskan dependensi produk S02/S01/T06/M0 atau menyatakan provider nyata stateless. Tidak ada listener/deployment/transport runtime, generasi provider atau perubahan WARP.

## Konfigurasi dan kandidat

Schema tetap `version: 1`, dengan field additive `models[].stateless` bertipe bool. Absent menjadi false. Hanya spelling YAML `true`/`false` huruf kecil yang diterima; string berpetik, yes/no/on/off, angka, uppercase boolean, null, struktur lain dan tag custom ditolak. Unknown/duplicate key, batas byte/node/depth, serta tipe field lain tetap diperiksa. Guard AST mendahului decode typed karena library YAML masih memiliki kompatibilitas yes/no untuk bool.

`routing: least_inflight` memerlukan `stateless: true` dan `affinity: none`. Strict tetap belum didukung, termasuk kombinasi strict+least. Fixed tetap default dan tetap memilih primary, meskipun diberi deklarasi stateless true. Primary wajib valid pada schema existing; pada least, primary bukan prioritas pemilihan atau fallback tersembunyi.

Contoh berikut hanya untuk model fixture stateless yang telah diuji. Bukan deklarasi provider/model nyata atau perubahan contoh deployment default:

```yaml
# Di dalam satu upstream terkonfigurasi; kedua relay wajib ada di registry.
relays: [warp-2, warp-1]
models:
  - alias: fixture-stateless
    upstream_model: fixture-model
    primary_relay: warp-1
    routing: least_inflight
    affinity: none
    stateless: true
    fallback_relays: []
```

Kandidat adalah ordered allowlist `upstreams[].relays`, bukan seluruh registry global atau daftar bebas dari client. `fallback_relays` tidak menjalankan retry pada S03. Counter yang dibandingkan adalah total generasi aktif `used[relay]` lintas alias/upstream; bukan persentase utilisasi, sisa kapasitas, jumlah koneksi atau probe.

Di antara kandidat eligible dan belum penuh, pilih inflight absolut paling rendah. Tie mengambil anggota pertama dalam allowlist. Pada contoh, tie memilih warp-2. Tidak ada round-robin atau janji fairness serial: jika request selesai sebelum request berikutnya, keduanya dapat memilih warp-2. Pilihan dihitung dari ledger terbaru, tanpa menyimpan histori request/sesi.

## Pemilihan, reservasi dan dispatch

`Reservation.Admit(upstream, alias, streamBytes) (relayID, error)` menggantikan AdmitFixed. Owner memegang mutex yang sama untuk resolve scope, satu pembacaan clock, model availability, accepting, mode, freshness/circuit transport, cooldown, kapasitas kandidat, perbandingan inflight dan commit gateway/relay/outbound/stream. Fixed dan least memakai satu jalur commit/release; tidak ada View/readiness sebagai precheck.

Gateway menggunakan ID hasil reservasi untuk lookup relay pada registry lalu menjalankan tepat satu Dispatch. Model mapping, origin/prefix, credential server dan semantik body JSON/SSE tetap mengikuti kontrak gateway. Jika invariant lookup gagal, request ditolak tanpa dispatch dan lease dilepas. Tidak beralih ke primary/default client. Tidak ada retry/failover setelah POST atau perpindahan stream yang sudah dimulai.

Lease pilihan tetap aktif sampai forwarding/caller Close selesai, termasuk sesudah EOF/error yang masih menyisakan pekerjaan tail. Release berulang/konkuren idempotent. Fixed tidak memilih relay lain ketika primary penuh atau ineligible. Least tidak menutup stream admitted ketika mode/cooldown berubah.

Urutan error promotion:

- Scope model diblokir atau seluruh kandidat gagal health/mode/cooldown: `503 no_eligible_relay`.
- Ada kandidat yang lolos faktor tersebut tetapi seluruhnya penuh: `429 gateway_capacity_exceeded`.
- Global active/outbound/buffer stream penuh: 429. Campuran cooldown+full adalah 429; cooldown+unhealthy adalah 503.
- Persiapan `Begin` tetap dapat menolak pending/body lebih awal dengan 429 sebelum scope diketahui. Tidak ada antrean tak terbatas.
- Penolakan sebelum admission tidak membuka koneksi/CONNECT/POST/direct. SSE tak terduga yang kehabisan stream buffer sesudah header tetap dapat menghasilkan 429 sesudah satu dispatch menurut D-023; bukan izin retry.

## Cooldown minimum dan snapshot

`Manager.ExtendRelayCooldown(relay, until)` hanya untuk sumber server tepercaya. Satu deadline finite per relay berlaku relay-wide pada seluruh mapping fixed/least yang memakai relay itu. ID unknown, clock/deadline invalid atau zero, dan deadline yang tidak di masa depan ditolak dengan error tetap tanpa data mentah.

Deadline future yang sama/lebih pendek adalah no-op; deadline lebih besar memperpanjang. Tidak ada clear/shorten atau timer/histori baru. `now < until` memblokir; tepat pada deadline cooldown tidak aktif lagi bila faktor lain layak. Mode, hasil observasi, probe dan request admitted tidak diubah. Probe sukses tidak menghapus cooldown.

Kontrak ini **tidak menafsirkan 429/Retry-After**. Respons upstream tersebut tidak otomatis dijadikan relay cooldown. Scope upstream/model/credential dan klasifikasi provider tetap S05; cooldown relay-only tidak menjadi alasan merotasi WARP, key, model atau upstream.

Snapshot menambah `models[].routing`, `models[].stateless`, `relays[].cooldown_until` nullable dan `cooldown_active` efektif. Deadline tersimpan tetap terlihat setelah expired dengan active=false. Revisi/state_changed_at berubah hanya ketika deadline persisted diperpanjang; read/expiry tidak memperbaruinya. Timestamp wire UTC, tanpa URL/key/payload/raw error. Daftar kandidat privat tidak dikirim ke UI atau dimiliki caller.

Readiness tetap `default_primary_transport` D-025/D-027. Cooldown/kapasitas/model availability bukan prediksi admission; primary tidak ready dapat membuat readiness false saat kandidat least lain masih dapat menerima request. UI tidak menghitung aturan eligibility sendiri.

Kandidat disimpan sekali per upstream, bukan matrix per model. Bound state owner menjadi `4096 + 1024 × relay + 1024 × model + 64 × referensi_upstream_relay` byte, maksimum 8.192 referensi. Tambahan metadata dan satu deadline per relay tetap bounded. Bound snapshot/monitoring S02 tetap mencukupi field kecil tambahan dan diverifikasi pada registry maksimum; konfigurasi dengan budget kurang ditolak, bukan menaikkan default diam-diam. Accounting bukan bukti RSS/performa.

## Bukti AC dan batas

| AC | Bukti lokal | Batas produk |
|---|---|---|
| AC1 | `TestLeastInflightMinimumAndTie`, `AbsoluteCountNotUtilization`, `SharedAcrossUpstreams`, `ConcurrentReservations`, `CoordinatorSelectedRelay`, `WireStreamCooldownAndCapacity`: counter total, pemilihan/reservasi dan jalur relay cocok | Fixture stateless, bukan provider/WARP |
| AC2 | `MinimumAndTie`, `WireStableSerialTie`: urutan allowlist warp-2 sebelum warp-1, termasuk setelah release | Tidak menjanjikan round-robin/fairness serial |
| AC3 | `TestLeastInflightConfiguration`, `MetadataCopies`: least true/none diterima; absent/false/strict/boolean ambigu ditolak; metadata immutable | Deklarasi operator bukan pembuktian provider otomatis |
| AC4 | `EligibilityAndCapacity`, `UnknownAndCooldownExpiry`, `TestCooldownDeadlineAndLease`, `WireStreamCooldownAndCapacity`: full-low dilewati, cooldown/expiry tepat, 503/429 campuran, lease tetap | Sumber deadline eksplisit, bukan classifier S05 |

Nama pada tabel memakai prefix `TestLeastInflight` kecuali nama yang ditulis penuh. Source: `internal/config/load_test.go`, `internal/catalog/registry_test.go`, `internal/pool/least_inflight_test.go`, `internal/gateway/admission_test.go`, `internal/httpapi/health_test.go`, dan `tests/integration/least_inflight_test.go`.

Tes tambahan membuktikan fixed default, mode/health/stale/model availability, cap agregat/probe, lifecycle mixed/race, state budget, metadata redaksi/expiry, serta satu dispatch pada transport error/redirect/upstream429 tanpa otomatisasi cooldown. Tes S01/S02 mempertahankan assertion; helper test memigrasikan API return ID tanpa mengubah kebijakan produk.

Fixture wire memakai dua SOCKS5 menuju satu authority HTTPS yang sama. Bukti memakai dial proxy/direct, CONNECT hostname, korelasi peer HTTP dengan socket relay, jumlah POST, model/key/body, event pertama dan terminal SSE. Client fixture tidak meninggalkan idle keepalive. Ini fixture HTTP/1.1 lokal; bukan transport runtime/HTTP2/client native.

Verifikasi Go `go1.27.4-X:nodwarf5 linux/amd64`:

- `go test -count=1 ./...`: 12 package root lulus, termasuk regresi config/auth/JSON/SSE/health/admission.
- `go test -race -count=1 ./...`: 12 package lulus.
- `go test -race -count=20 ./tests/integration -run 'TestLeastInflight|TestEligibility|TestFixedAdmission'`: 20 pengulangan setiap tes terpilih lulus, tanpa kegagalan/race.
- `go test -race -count=50 ./internal/pool`: 50 pengulangan lulus tanpa kegagalan/race.
- `go vet ./...`: lulus; `gofmt -l internal tests/integration`: kosong.

Tidak menjalankan provider/WARP/native smoke, benchmark/RSS atau membuktikan gerbang M0/MVP. Default numerik produk, scope429 S05, retry S04, binding/maintenance, dashboard dan deployment tetap task pemilik. Dependency dan contoh fixed default tidak berubah. Kontrak boolean diverifikasi melalui Context7 pada [go-yaml v4.0.0](https://github.com/yaml/go-yaml/blob/v4.0.0/README.md) dan [opsi loader](https://github.com/yaml/go-yaml/blob/v4.0.0/docs/options.md).
