# Admission kapasitas untuk primary fixed

Tanggal: 26 November 2026  
Task: [EP-03-S01](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s01)  
Status: komponen dan fixture lokal tersedia; penerimaan produk terintegrasi tetap terbuka

[D-026](DECISIONS.md#d-026-admission-fixed-independen-ep-03-s01) menyetujui pekerjaan lokal independen sementara EP-02-S05, EP-01-T06 dan gerbang M0 masih parsial. Pembuktian ini memakai primary fixture yang tersedia. Keanggotaan relay dalam konfigurasi bukan bukti health runtime. Eligibility health/mode/circuit terpadu tetap [EP-03-S02](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02), sedangkan cooldown mengikuti S05; tidak ada snapshot readiness yang dipakai sebagai pemeriksaan sebelum reservasi.

## Pemilik dan transisi lease

`internal/pool.Admission` memiliki seluruh counter persiapan, request admitted, byte body, byte stream, relay dan slot dispatch outbound. Semua operasi check/update menggunakan mutex owner yang sama. Tidak ada antrean menunggu kapasitas.

| Operasi | Pemeriksaan dan efek |
|---|---|
| `NewAdmission(snapshot)` | Memerlukan snapshot tervalidasi; mengambil batas eksplisit dari konfigurasi |
| `Begin(bytes)` | Memeriksa `pending + active < max_active_requests` dan sisa buffer body sebelum alokasi/read; mengambil satu pending lease serta byte konservatif |
| `Reservation.AdmitFixed(primary, streamBytes)` | Menerima primary dari mapping fixed server tervalidasi; memeriksa lease, keanggotaan relay serta kapasitas gateway/relay/outbound/stream; lalu memindahkan pending ke active dan mengambil seluruh slot terkait dalam satu commit |
| `Reservation.AdmitLocal()` | Memindahkan pending ke active untuk katalog lokal; tidak mengambil relay, outbound atau stream |
| `Reservation.ReserveStream(bytes)` | Hanya untuk generasi yang sudah di-admit dan mendapat SSE tanpa deklarasi stream; mengambil byte sebelum alokasi buffer forwarding |
| `Reservation.Release()` | Mengurangi counter sesuai tahap pending/admitted tepat sekali; panggilan berulang atau konkuren tidak menggandakan pelepasan |

Lease berada dalam satu tahap: pending, admitted, lalu released. Lease tidak boleh disalin. Pending sudah membebani batas pekerjaan dan buffer, tetapi belum merupakan keputusan admission generasi. Batas `pending + active <= max_active_requests` mencegah perubahan ini membuka pekerjaan parsing yang tidak terbatas. Katalog dan generasi berbagi batas yang sama.

Seluruh pemeriksaan promosi mendahului mutasi counter di bawah mutex. Promosi gagal tidak mengambil sebagian slot gateway/relay/outbound/stream. Pending dan byte yang sudah dimiliki dilepas melalui finalisasi caller. Fixed tidak mencari relay pengganti ketika primary penuh; request untuk primary lain masih dapat berjalan jika batas global dan primary tersebut mencukupi.

`AdmitFixed` memvalidasi keanggotaan primary yang diberikan server. Operasi ini belum menentukan eligibility health/mode/cooldown produksi. Tidak ada constructor baru, kebijakan health default sehat, atau jalur jaringan langsung. Sebelum penerimaan runtime, eligibility terpadu harus dikoordinasikan dengan reservasi di owner pool sesuai [arsitektur](ARCHITECTURE.md#eligibility-dan-kapasitas).

## Byte sebelum alokasi

Budget body tetap konservatif menurut [kontrak S03](model-alias-dan-json.md#admission-dan-buffer):

```text
2 × (max_body_bytes + 1) + 2 × metadata_terbesar + 256 KiB
```

Budget ini mencakup body wire, hasil remap, salinan metadata serta cadangan scanner/header/key dan copy respons JSON. Byte diambil sebelum persiapan request, alokasi buffer input, parsing, atau remap. Body known-length maupun unknown/chunked tetap dibatasi pada byte aktual dan ukuran hasil remap. Request kecil tetap memakai budget konservatif; tidak ada estimasi body kecil yang mengizinkan alokasi melebihi ledger.

Untuk `stream:true`, copy buffer 32 KiB diambil bersama commit admission sebelum dispatch. Ukuran copy bukan batas ukuran event. SSE tak terduga pada absent/false baru diketahui dari header upstream: `ReserveStream` mengambil budget setelah satu dispatch dan sebelum forwarding. Jika tidak muat, body ditutup dan 429 dikembalikan tanpa retry, sesuai [D-023](DECISIONS.md#d-023-streaming-sse-independen-ep-02-s04).

Profil D-019 dan angka konfigurasi contoh tetap bukan default final atau bukti RSS. Ledger body/stream tidak mengukur GC, stack, TLS, socket, maupun seluruh heap proses. Slot outbound membatasi dispatch aktif, bukan seluruh koneksi idle/probe. Accounting koneksi runtime dan penerimaan Q-5 tetap membutuhkan pengujian terintegrasi.

## Error dan finalisasi

- Batas global, body, relay, outbound atau stream deklaratif habis menghasilkan `429 gateway_capacity_exceeded` sebelum dispatch. Ini memperjelas kontrak lama S03/S05 yang menyatukan slot relay penuh dengan `503 no_eligible_relay`.
- Primary tidak dikenal pada operasi pool menghasilkan `no_eligible_relay`; lease invalid/released/admitted ganda menghasilkan `invalid_reservation`, tanpa mutasi. Endpoint normal mengambil primary dari registry tervalidasi, bukan input relay client. Eligibility produksi 503 seluruh kandidat tidak layak tetap harus dibuktikan pada S02.
- Envelope, request ID server dan error source memakai boundary HTTP yang sudah ada. Status/body 429 dari upstream tetap opaque; perubahan kategori kapasitas gateway tidak menambahkan cooldown atau retry.
- Kegagalan sebelum transfer ownership melepas pending/admitted lease melalui defer. Setelah respons tersedia, caller wajib `response.Body.Close()` juga sesudah EOF/error/cancellation. Handler HTTP menutup body setelah forwarding selesai.
- EOF/error/cancellation menutup jaringan tepat sekali, tetapi tidak melepas lease sebelum caller selesai menulis tail respons. Close berulang/konkuren tetap melepas seluruh kapasitas tepat sekali.

Target/key/model server, SOCKS5, TLS hostname, tanpa redirect/direct fallback/replay ambigu, serta byte JSON/SSE tetap mengikuti kontrak S03–S05. Tidak ada perubahan transport atau lifecycle jaringan pada S01.

## Bukti dan batas penerimaan

| AC | Bukti lokal | Batas |
|---|---|---|
| AC1 | `TestFixedAdmissionAtomicPromotion`, `RejectedPromotion`, `Concurrent`: transisi global/primary/outbound/stream, tanpa reservasi admitted parsial atau fallback | Primary fixture tersedia; bukan integrasi health runtime |
| AC2 | `TestFixedAdmissionCapacityCategory`, `CapacityBeforeDispatch`: saturasi global/relay/outbound dan 429, nol tambahan koneksi/CONNECT/POST | Slot dispatch aktif, bukan accounting semua socket |
| AC3 | `TestFixedAdmissionBodyBudget`, `PendingWorkAndCatalogue`, `RejectedPromotion`, regresi `TestModelAdmissionGlobalBudget`/`TestGatewayStreamBudget`: budget agregat, known/unknown body, exact cap dan recovery | Ledger konservatif; bukan benchmark alokasi/RSS |
| AC4 | `TestFixedAdmissionReleaseLifecycle`, `Concurrent`, `LocalActivation`, `CapacityBeforeDispatch` serta regresi input/dispatch/timeout/disconnect: Close/cancel bersaing, lease tail dan kapasitas kembali | Bukan penerimaan target cancellation/performa S06 |

Nama pada tabel memakai prefix `TestFixedAdmission` kecuali nama regresi yang ditulis penuh. Source utama berada di `internal/pool/admission_test.go`, `internal/gateway/admission_test.go`, dan `tests/integration/admission_test.go`. Fixture jaringan memakai dua HTTPS upstream dan dua SOCKS5 loopback yang sudah tersedia. Pemeriksaan sebelum dispatch membandingkan delta counter upstream dan proxy; ketiadaan CONNECT saja tidak dipakai sebagai bukti ketiadaan POST pada koneksi reuse.

Verifikasi pada Go `go1.27.4-X:nodwarf5 linux/amd64` menggunakan dependency existing secara offline:

- `go test -count=1 ./...`: seluruh 11 package lulus, termasuk regresi JSON/SSE, auth, health, input/error dan fixture spike lokal.
- `go test -race -count=1 ./...`: seluruh 11 package lulus tanpa laporan race.
- `go test -race -count=20 -run '^TestFixedAdmission' ./internal/pool ./internal/gateway ./tests/integration`: 20 pengulangan per tes terpilih, tanpa kegagalan atau race. Ini bukan benchmark.
- `go vet ./...`, `gofmt -l` pada berkas Go yang berubah dan `git diff --check`: lulus; keluaran format kosong.
- Pemeriksaan tautan/anchor pada tujuh dokumen yang berubah: 241 tautan lokal valid.

Tiga siklus RED/GREEN membuktikan pemisahan pending, kategori saturasi 429 dan larangan buffer stream pada pending. Aktivasi katalog merupakan refactor pada kondisi green yang mempertahankan perilaku kapasitas bersama; tes tambahan memperkuat bukti konkurensi, lifecycle dan jaringan. Full suite dijalankan setelah setiap siklus dan refactor. Tidak ada perubahan produk di luar rencana.

Review `diff-review` depth strict selesai dengan verdict **APPROVE**, tanpa temuan terbuka. Review memeriksa semua caller API admission yang berubah, jalur HTTP/dispatch/finalisasi, tes, serta konsistensi dokumentasi. Pengerjaan memakai tier high, tiga subagent analisis, 16 command pengujian setelah baseline dan satu putaran review; tidak ada eskalasi atau deviasi rencana.

Tes jaringan dijalankan di luar sandbox karena socket loopback ditolak oleh sandbox. Native smoke tetap opt-in dan tidak diaktifkan. Tidak ada provider/WARP nyata, deployment, benchmark, kelulusan M0, commit atau push dalam pembuktian ini.
