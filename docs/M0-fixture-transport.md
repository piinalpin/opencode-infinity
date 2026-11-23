# M0 — Fixture jalur dan attempt request

Versi: 0.1.0 · Tanggal: 23 November 2026 · Task: [EP-01-T03](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03)

AC1–AC5 T03 terbukti pada fixture sintetis loopback. Pengguna menyetujui pengerjaan fixture ini secara independen saat keterbatasan T01/T02 masih dicatat. Hasil ini tidak menutup dependensi tersebut, gerbang M0, atau penerimaan AC sumber pada gateway produksi.

✅ pada tabel AC di bawah menyatakan lingkup fixture lokal ini. [Spike T04](M0-warp-request.md) memakai helper tanpa mengubah guard loopback dan menambahkan kontrol test-only terpisah; hasilnya tidak mengubah bukti historis T03 menjadi kelulusan provider/WARP.

## Struktur dan pemakaian

[go.mod](../go.mod) menetapkan module lokal `opencode-infinity`, Go `1.27.4`, tanpa dependency eksternal. Distribusi resmi dan checksum toolchain mengikuti [bukti T02](M0-stack-dan-versi.md). Module ini menyediakan helper dan tes; belum ada binary gateway.

| Lokasi | Tanggung jawab |
|---|---|
| [socks5.go](../internal/testutil/socks5.go) | Listener SOCKS5 terbatas pada satu authority → target IP loopback; rekaman relay, hostname domain, port, refusal, serta peer TCP keluar |
| [upstream.go](../internal/testutil/upstream.go) | Upstream TLS bersertifikat sintetis, ledger dispatch/body-read/lifecycle, oracle pencocokan peer dan pemutusan HTTP/1.1 |
| [stream.go](../internal/testutil/stream.go) | Pemotongan byte deterministik dengan flush dan validasi batas |
| [fixture_test.go](../internal/testutil/fixture_test.go) | Guard input, handshake rusak, kapasitas koneksi, dan cleanup |
| [client_test.go](../tests/integration/client_test.go) | Client kontrol tanpa retry loop, dialer allowlist terinstrumentasi, dan consumer SSE terbatas |
| [routing_test.go](../tests/integration/routing_test.go) | Jalur, bypass, TLS, allowlist, serta batas body/ledger |
| [fault_test.go](../tests/integration/fault_test.go) | Koneksi reusable, refusal, dan POST ambigu |
| [stream_test.go](../tests/integration/stream_test.go) | Semantik/flushing SSE, stream putus, cancellation, sukses, dan cleanup konkuren |

`internal/testutil` hanya diimpor berkas `*_test.go`, sesuai [batas arsitektur pengujian](ARCHITECTURE.md#134-organisasi-frontend-dan-pengujian). Task berikutnya dapat menjalankan gateway/transport yang sedang diuji terhadap fixture ini. Handler fixture upstream harus menghormati `Request.Context()` agar cleanup dapat selesai.

`NewUpstream(handler)` menyediakan `Authority`, `Addr()`, dan root CA sintetis. `NewSOCKS5(relay, authority, target, refuse)` hanya meneruskan CONNECT domain yang cocok ke target IP loopback tersebut. Untuk client yang diuji, pasang proxy `socks5h`, root CA fixture, dan hostname URL asli. Pemetaan TCP ke loopback tidak mematikan verifikasi TLS. Tidak perlu DNS/hosts sistem, provider, credential, atau container.

## Arti observasi

Koneksi TCP client ke proxy, socket proxy yang diterima untuk diproses, SOCKS CONNECT valid, dan dispatch HTTP upstream adalah penghitung berbeda. Dua POST selesai pada koneksi reusable menghasilkan **dua dispatch dan satu CONNECT**. Dialer kontrol mencatat attempt proxy termasuk kegagalan TCP, serta attempt direct secara terpisah.

`Upstream.Snapshot()` mengembalikan salinan ledger: `Total` menghitung semua dispatch, `Rejected` menunjukkan ledger penuh, dan `Dispatches` berisi rekaman terbatas. `BodyRead` menunjukkan body sintetis terbaca lengkap; body/header tidak disimpan. `Returned` berarti handler wrapper kembali, bukan bukti respons sukses. `Canceled` diamati sebelum wrapper kembali; pembatalan context otomatis setelah return normal tidak dihitung sebagai pembatalan saat aktif. Respons sukses dibuktikan terpisah melalui terminal SSE dan EOF valid.

`VerifyProxyPath` mencocokkan peer TCP setiap dispatch dengan socket keluar relay. Tes positif juga memeriksa relay yang dipilih dan jumlah dispatch. Pada proxy mati, tes menuntut nol direct dial/dispatch. Kontrol direct sengaja dapat mencapai upstream yang sama, lalu oracle harus menolaknya. Dengan demikian kegagalan DNS atau upstream yang tidak hidup tidak dapat menyamarkan bypass.

## Matriks bukti

| AC T03 | Skenario otomatis | Bukti |
|---|---|---|
| ✅ AC1 | `TestRoutingThroughSOCKS5`, `TestReusableConnectionCounters` | Relay B menerima domain `upstream.invalid` dan port; relay A tidak digunakan; peer cocok; dua dispatch dapat memakai satu CONNECT |
| ✅ AC2 | `TestFragmentedSSESemanticsAndFlush` | Tiga pola pemotongan byte, termasuk satu byte per write dan UTF-8; urutan JSON/content/tool-call arguments, usage, field tambahan, serta terminal tetap utuh |
| ✅ AC2 | Barrier event pertama | Event berikutnya/akhir ditahan sampai client menerima event pertama; tes tidak menyamakan batas Write, paket TCP, Read, dan event SSE |
| ✅ AC3 | `TestRefusalBeforeDispatch` | Fault SOCKS ditandai sebelum target dial/tunnel; bukan kesimpulan dari dispatch nol saja |
| ✅ AC3 | `TestAmbiguousPOSTNotReplayed` | Upstream sudah membaca POST, kemudian koneksi diputus sebelum header; koneksi baru dan reusable masing-masing menghasilkan satu dispatch untuk POST ambigu |
| ✅ AC3 | `TestStreamStartedNotReplayed` | Client mengakui event pertama sebelum koneksi diputus; truncation menghasilkan error, tanpa terminal sukses atau dispatch tambahan |
| ✅ AC4 | `TestClientDisconnectCancelsUpstream` | Context cancel dan body-close diamati upstream saat handler masih aktif, dalam batas observasi ≤2 detik; ledger kemudian mencatat cancellation |
| ✅ AC4 | `TestSuccessfulStreamIsNotCancellation` | Terminal SSE dan EOF valid; return normal tidak tercatat sebagai cancellation |
| ✅ AC5 | `TestUnavailableProxyAndDirectOracle` | Proxy mati menggagalkan request; kontrol direct benar-benar mencapai upstream dan ditolak oracle |
| Turunan | Tes TLS/allowlist, guard frame/handshake/body/ledger, serta `TestConcurrentFixtureCleanup` | Hostname salah ditolak, tujuan lain tidak didial, input/buffer/koneksi dibatasi, dan resource fixture selesai setelah mixed success/error/cancel |

Label fase berasal dari fault boundary dan observasi positif yang disinkronkan. Nol dispatch, error string, ataupun `httptrace.WroteRequest` saja tidak membuktikan POST aman diulang. Client kontrol memakai body tanpa `GetBody`, tanpa idempotency header, serta tanpa retry loop. Jumlah dispatch menguji interaksi dengan retry internal standard library pada HTTP/1.1; ini belum menguji retry SDK/client nyata.

## Batas resource dan reproduksibilitas

Listener memakai `127.0.0.1` dan port ephemeral. Tiap fixture membatasi 16 koneksi aktif dan 256 penerimaan/rekaman; ledger penuh menolak request tambahan dengan 503 dan tetap menghitungnya. Body serta frame dibatasi 64 KiB; buffer copy proxy 16 KiB per arah. Socket proxy mempunyai deadline absolut 5 detik; upstream membatasi header, pembacaan, penulisan, dan idle. Ini batas fixture, bukan nilai konfigurasi produk atau benchmark.

Snapshot disalin di bawah mutex. Barrier mengatur urutan observasi; timer membatasi kegagalan. Cleanup menutup listener dan socket, membatalkan stream, menutup idle transport, dan menunggu worker milik fixture. `Close` dapat dipanggil ulang, dengan batas tunggu 5 detik. Tes menuntut socket aktif nol; penghitung goroutine proses secara global tidak digunakan sebagai bukti utama.

## Menjalankan ulang

Gunakan distribusi resmi Go `1.27.4` pada `PATH`; race detector memerlukan cgo dan compiler C. Semua koneksi pengujian terbatas pada loopback. Environment berikut mencegah pengunduhan toolchain/module secara otomatis:

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off

go test -run '^TestRoutingThroughSOCKS5$' -count=1 ./tests/integration
go test -count=1 -timeout=120s ./...
go vet ./...
gofmt -l internal/testutil tests/integration
CGO_ENABLED=1 go test -race -count=50 -timeout=10m ./...
```

Hasil aktual pada 23 November 2026, Linux amd64, distribusi resmi Go 1.27.4, cgo aktif dan GCC 16.2.1:

- Seluruh suite: **18 tes utama lulus** (4 helper, 14 integrasi), termasuk subtest negatif dan lifecycle.
- `go vet ./...`: lulus; `gofmt -l`: kosong.
- `go test -race -count=50 -timeout=10m ./...`: **900 eksekusi tes utama lulus**, nol kegagalan dan nol laporan data race. Penghitungan event PASS yang juga mencakup parent/subtest adalah 1.750.
- Baseline sebelum implementasi belum tersedia: `go test ./...` gagal setup karena belum ada root module; tidak diklaim sebagai tes lulus atau regresi.

Sandbox sesi memblokir pembukaan socket. Tes jaringan aktual dijalankan di luar sandbox dengan endpoint sintetis loopback; tidak ada pengujian upstream nyata.

## Batas pembuktian dan tindak lanjut

Fixture ini memakai listener/target TCP IPv4 loopback, HTTP/1.1 dan SOCKS5 CONNECT domain tanpa autentikasi. Ia tidak menjadi proxy umum atau implementasi gateway/coordinator. HTTP/2, auth SOCKS, timeout/retry SDK aktif, DNS jaringan container, WARP, provider/model, serta performa produksi belum dibuktikan. Target cancellation Q-6 hanya diamati pada jalur kontrol lokal; harus diuji ulang pada gateway terintegrasi.

Keterbatasan [T01](M0-lingkungan-dan-integrasi.md) dan [T02](M0-stack-dan-versi.md) tetap berlaku. [EP-01-T04](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t04) membuktikan jalur WARP/upstream nyata, sedangkan penerimaan AC sumber tetap mengikuti gerbang integrasi/rilis terkait. Status `approved` pada tiket menyatakan persetujuan cakupan dan tidak diubah otomatis oleh hasil fixture.
