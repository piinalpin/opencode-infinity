# Streaming content dan tool calls melalui SSE

Tanggal: 25 November 2026  
Task: [EP-02-S04](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s04)  
Status: komponen dan wiring fixture lokal; penerimaan gateway terintegrasi tetap parsial

[D-023](DECISIONS.md#d-023-streaming-sse-independen-ep-02-s04) menyetujui perluasan independen atas [S03](model-alias-dan-json.md). Pengujian melewati client HTTP, handler gateway, adapter tepercaya dan upstream HTTPS melalui SOCKS5 fixture. Hasil ini tidak meluluskan transport runtime, executable/listener/deployment produk, OpenCode/provider/WARP, performa/RSS atau gerbang M0.

[S05/D-024](error-dan-batas-request.md) melengkapi error sebelum stream: path/method/protokol memakai envelope gateway, respons generasi 3xx ditolak sebelum commit, dan `Retry-After` duplikat tidak diteruskan. Kontrak fragmentasi, Flush, timeout dan kegagalan setelah commit di bawah tetap berlaku.

## Kontrak request dan respons

Endpoint generasi tetap `POST /v1/chat/completions` dan `POST /providers/{provider}/v1/chat/completions`. Token gateway, media type JSON, batas body, alias, target/key server dan fixed primary relay mengikuti S03. `stream:true` kini diterima dan dipertahankan pada JSON outbound; absent/false mempertahankan perilaku non-stream. Tipe stream selain boolean dan kontrol model/stream duplikat, termasuk nama escaped, tetap ditolak sebelum dispatch. Pemetaan hanya mengganti nilai top-level model.

Mode forwarding SSE ditentukan bersama oleh coordinator dan HTTP boundary dari media type respons upstream `text/event-stream`, termasuk parameter charset. Body tidak diperiksa untuk menebak mode. Error/JSON non-SSE dari upstream tetap diteruskan lewat jalur normal, meskipun request meminta stream.

Forwarding SSE menyalin byte opaque menggunakan satu buffer 32 KiB dan memeriksa short write serta error Read/Write/Flush. Header dan setiap potongan data di-flush; tidak menunggu event berikutnya atau EOF. Batas Read, Write, paket TCP dan event SSE tidak diasumsikan sama. Tidak ada parser event/vendor, antrean pembaca latar, buffer seluruh event/stream, atau cap ukuran event produk baru.

Content/tool-call deltas, IDs/index/nama/argumen tool, usage, extension/angka besar, komentar, line ending dan urutan byte dipertahankan. Usage absent tidak dibuat nol. `[DONE]` hanya diteruskan jika dikirim upstream. EOF HTTP normal tanpa terminal juga diteruskan tanpa sintesis terminal; gateway tidak menambahkan validasi terminal vendor yang belum menjadi kontrak.

Content-Type, Content-Encoding yang masih berlaku dan Retry-After valid mengikuti allowlist S03. Bila adapter sudah mendekompresi body, adapter wajib menghapus metadata encoding yang tidak lagi berlaku. Fixture gzip opaque menonaktifkan auto-decode adapter/client, mem-flush compressor dan HTTP upstream, lalu membuktikan event pertama dapat dibaca sebelum gate event berikutnya dibuka. Gateway tidak memaksa byte yang masih ditahan compressor upstream.

## Admission, backpressure dan kepemilikan

Menurut pembaruan [EP-03-S01/D-026](admission-fixed.md), `internal/pool.Reservation.AdmitFixed(primary, streamBytes)` mengambil gateway/relay/outbound serta stream deklaratif secara atomik setelah persiapan pending yang bounded. `ReserveStream(bytes)` hanya mengambil budget SSE yang baru diketahui sesudah header pada generasi yang sudah di-admit. Keduanya memakai mutex dan lease yang sama. Budget berasal dari `limits.stream_buffer_bytes`. Reservasi copy 32 KiB harus tersedia sebelum buffer forwarding dialokasikan; ukuran copy bukan ukuran event. Budget body S03 tetap konservatif dan tidak dioptimalkan/dilepas sebagian selama stream. Menolak ketika penuh, tanpa antrean atau read-ahead tanpa batas.

- Request true mengambil reservasi stream sebelum dispatch. Kapasitas habis menghasilkan `429 gateway_capacity_exceeded` tanpa dispatch generasi.
- SSE tak terduga pada request absent/false juga wajib mengambil reservasi sesudah header upstream, sebelum forwarding. Bila penuh, body ditutup dan 429 dikembalikan tanpa retry; satu dispatch sudah dapat terjadi.
- Counter stream bytes ikut dilepas tepat sekali bersama lease. Penutupan jaringan saja tidak membebaskan buffer/slot yang masih digunakan HTTP writer.
- Saat Write downstream terblokir, loop tidak membaca potongan upstream berikutnya. Buffer hanya tetap satu potongan; kernel/transport fixture mempunyai batas terpisah dan tidak dianggap bagian dari bukti RSS aplikasi.

Request/global/body/stream/relay/dispatch lease hidup sampai forwarding selesai dan caller menutup body, juga setelah EOF/error/cancellation. HTTP handler memasang `response.Body.Close()` sebagai defer. Close berulang/konkuren tidak menggandakan penutupan jaringan atau pelepasan lease. Pemakai coordinator langsung tetap wajib Close setelah EOF sesuai kontrak S03.

Admission ini tetap kapasitas statis, bukan health eligibility, cooldown, binding, drain, maintenance atau routing dinamis. Socket idle milik adapter dan koneksi inbound listener runtime bukan counter yang dibuktikan oleh lease ini. Tidak ada klaim 20 stream/30 menit, overhead p95 atau RSS dari tes fungsional fixture.

## Timeout, cancellation dan client lambat

Semua nilai timeout berasal dari snapshot konfigurasi; tidak ada field atau default numerik produk baru.

| Batas | Pemilik | Waktu yang dibatasi |
|---|---|---|
| `response_headers_seconds` | Adapter tepercaya; fixture memakai `http.Transport.ResponseHeaderTimeout` | Menunggu header upstream, bukan lifetime body setelah header |
| `stream_idle_seconds` pada upstream | Pembungkus response body coordinator | Read sedang menunggu byte; tidak mengukur waktu Write akibat backpressure client |
| `stream_idle_seconds` pada downstream | HTTP forwarding | Kemajuan Write/Flush; dibedakan dari upstream idle |
| `total_request_seconds` | Context coordinator dan deadline forwarding | Batas total, tidak diperpanjang oleh kemajuan byte |
| Deadline/cancellation client | Context request | Batas client yang lebih awal dan disconnect yang terdeteksi |

Setiap Read upstream mempunyai timer idle sendiri. Read dengan kemajuan byte menyelesaikan timer tersebut; Read berikutnya memulai penantian baru. Callback yang terlanjur berjalan disinkronkan sampai selesai, sehingga tidak dapat mematikan Read berikutnya. Timer tidak hidup ketika caller belum memanggil Read atau sedang menulis downstream. `Read` dengan buffer kosong tidak membatalkan stream. Adapter reader yang berulang kali mengembalikan nol byte tanpa error dibatasi dengan `io.ErrNoProgress`, bukan ditafsirkan sebagai EOF sukses.

Deadline penulisan dipasang melalui `http.ResponseController`. Saat menunggu Read upstream, deadline Write hanya memakai batas total/client agar idle Write tidak kedaluwarsa tanpa aktivitas menulis. Saat Write/Flush dimulai, deadline juga dibatasi interval idle dari saat itu. Client cancellation memajukan write deadline agar Write/Flush yang terblokir dapat keluar. Callback cancellation dihentikan atau ditunggu sampai selesai sebelum deadline dibersihkan dan handler kembali; callback lama tidak boleh mengubah koneksi request berikutnya. Acuan API: [ResponseController](https://pkg.go.dev/net/http#ResponseController), [SetWriteDeadline](https://pkg.go.dev/net/http#ResponseController.SetWriteDeadline), serta [Timer.Stop](https://pkg.go.dev/time#Timer.Stop).

Writer server harus menyediakan Flush dan deadline, langsung atau melalui `Unwrap` yang kompatibel dengan ResponseController. Writer yang terdeteksi tidak mendukung capability ini menghasilkan `501 unsupported_api` sebelum stream dikomit, tanpa fallback buffering. Header upstream belum disalin pada penolakan ini agar envelope JSON gateway tidak mendapat Content-Encoding gzip yang tidak berlaku. Dispatch generasi dapat sudah terjadi; tidak ada retry untuk menutupi kegagalan capability server.

Timeout header sebelum stream menghasilkan 504 dengan envelope/request ID server. Setelah respons dikomit, error Read/Write/Flush, idle/total timeout atau disconnect menghentikan framing HTTP dengan `http.ErrAbortHandler`. Tidak menambahkan envelope, DONE atau generasi lain. Client dapat menerima potongan yang sudah dikirim dan kemudian error/truncation, tanpa EOF sukses palsu. Error/stream upstream tidak digunakan untuk mengganti model, key, upstream atau relay.

Cancellation upstream diuji dalam target ≤2 detik setelah client Context dibatalkan atau response body ditutup, ketika handler upstream masih aktif dan sebelum teardown fixture. Callback/timer hanya membatalkan jaringan; lease tetap hidup sampai writer/caller selesai. Header/connection enforcement yang berada pada adapter fixture tidak menjadi bukti adapter arbitrer atau transport runtime aman.

## Privasi dan regresi

Guard serta request boundary S02 tidak diubah. Token admin/upstream bukan token gateway. Auth invalid, alias/provider unknown, input kontrol/body/target tidak valid tetap ditolak sebelum dispatch yang relevan; tes membuktikan nol CONNECT/dispatch pada penolakan tersebut.

Modul tidak mencatat prompt, output/tool bodies, Authorization, key atau raw session ID. Error milik gateway hanya kategori tetap, tanpa error jaringan/parser mentah atau potongan body. Batas refleksi upstream mengikuti D-021/S03: body diteruskan utuh, tidak ada scrubber yang mengubah payload upstream. Raw outbound request/response.Request tetap objek istimewa yang tidak boleh di-dump.

Katalog GET/HEAD, pemetaan alias default/provider, JSON absent/false, usage, gzip, snapshot/key, partial JSON dan batas S03 tetap diperiksa. Tiga tes lama yang menolak true dipindahkan menjadi dukungan positif S04; negatif tipe/duplikat kontrol tetap dipertahankan.

## Bukti AC dan verifikasi

| AC S04 | Bukti komponen/fixture | Penerimaan produk |
|---|---|---|
| AC1 | Gateway HTTP default/provider melewatkan first event sebelum barrier event kedua/EOF; gzip opaque juga diuji | Parsial; integrasi client/provider/WARP belum diuji |
| AC2 | Fragmentasi byte/UTF-8/tool deltas dan event >64 KiB diteruskan tanpa rekonstruksi/cap event | Parsial; kompatibilitas upstream nyata belum dibuktikan |
| AC3 | Transcript content/tools/usage/extensions serta raw byte/line endings/encoding utuh; absent tetap absent | Parsial; penerimaan gateway terintegrasi tetap terbuka |
| AC4 | Drop sesudah event/fragmen memberi truncation tanpa terminal/envelope/replay; POST ambigu fresh/reused tetap satu attempt | Parsial; fault provider/WARP/runtime belum diuji |

Tes baru berada di `internal/gateway/stream_test.go`, `internal/httpapi/stream_test.go` dan `tests/integration/gateway_stream_test.go`. Tes parser, coordinator, admission serta fixture model yang sudah ada diperluas di tempatnya. Fixture T03 reusable tidak dilonggarkan atau dijadikan dependency runtime. Semua data sintetis; network hanya loopback.

Hasil verifikasi:

- `go test -json -count=1 ./...`: full suite lulus; hitungan tes aktual tercatat pada laporan pengerjaan.
- `go test -race -count=1 ./...`: seluruh package lulus.
- `go test -race -count=30 -timeout=10m -run '^TestGatewayStream' ./internal/gateway ./internal/httpapi ./internal/pool ./tests/integration`: seluruh 30 pengulangan lulus tanpa laporan race, termasuk timeout dan cancellation integrasi. Tes no-progress/Read kosong yang ditambahkan sesudah run ini mulai juga lulus 30 kali dengan race secara terpisah.
- Fuzz `FuzzModelJSON` 30 detik dengan dua worker: 178.493 eksekusi tanpa kegagalan.
- `go vet ./...`, format/diff, 803 tautan lokal dan 338 anchor: lulus.

Review strict memeriksa delta S04 terhadap snapshot awal, termasuk berkas S03 yang semula belum di-commit. Selama pengerjaan, commit S03 `bb3ad08` muncul dengan seluruh 17 hash baseline identik; tidak ada commit/push yang dijalankan agent. Native smoke, provider/WARP/deployment, HTTP/2, performa/RSS dan gerbang M0 tidak dijalankan atau diluluskan.

Catatan pengujian: satu run full suite menemukan flake lama `TestMalformedSOCKS5/bad-atyp` pada `CloseWrite` dengan `transport endpoint is not connected`. Helper tersebut tidak berubah; kasus ini sudah tercatat pada S03. Run full suite/race sesudahnya lulus. Tidak ada perbaikan di luar scope atau pelemahan assertion helper untuk menyembunyikan flake.
