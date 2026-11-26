# Katalog alias model dan generasi JSON

Tanggal: 25 November 2026  
Task: [EP-02-S03](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s03)  
Status: komponen dan wiring fixture lokal tersedia; penerimaan gateway terintegrasi tetap parsial

[D-022](DECISIONS.md#d-022-katalog-alias-dan-json-independen-ep-02-s03) menyetujui pekerjaan independen ini. Input memakai [konfigurasi T01](konfigurasi.md), [guard dan request boundary S02](akses-dan-credential.md). Dependensi S02/T01/T06/M0 tidak dihapus. Tidak ada executable/listener gateway baru, transport jaringan runtime, perubahan deployment, provider request atau klaim integrasi OpenCode/WARP/performa.

Dokumen ini menyimpan kontrak dan bukti tahap S03. Dukungan `stream:true` kemudian ditambahkan pada [EP-02-S04/D-023](streaming-sse.md); batas unsupported true di bawah adalah keadaan historis S03. Perilaku absent/false tetap mengikuti kontrak JSON ini.

[EP-02-S05/D-024](error-dan-batas-request.md) melengkapi error path/method/protokol serta penolakan redirect. Passthrough 3xx historis S03 digantikan `502 upstream_redirect_rejected` untuk semua respons generasi 300–399; status/body 4xx/5xx tetap utuh. Hanya satu nilai `Retry-After` valid diteruskan. Encoding request tetap absent/`identity`.

## API komponen

| Komponen | API | Tanggung jawab |
|---|---|---|
| `internal/httpapi` | `NewAPI(snapshot, dispatcher)` | Merakit empat route, guard gateway, validasi media type, envelope error dan penerusan respons |
| `internal/gateway` | `New(snapshot, dispatcher)`, `Models(provider)`, `Chat(request, provider)` | Memiliki snapshot, katalog/remap, koordinasi admission dan satu dispatch |
| `internal/gateway` | `Dispatcher.Dispatch(request, relay)` | Kontrak efek jaringan server tepercaya; implementasi S03 berada pada fixture pengujian |
| `internal/pool` | `NewAdmission(snapshot)`, `Begin(bytes)`, `Reservation.Assign(relay)`, `Reservation.Release()` | Kapasitas statis request/buffer dan slot relay/dispatch outbound dengan satu mutex |

Constructor menolak snapshot belum valid dan dispatcher nil, termasuk typed nil. Tidak ada default client atau transport direct implisit. Snapshot disalin sebagai satu versi sebelum komponen dirakit; mengganti object konfigurasi caller setelah constructor tidak mencampur key/target/model. Caller tidak boleh menulis object tersebut secara serentak ketika constructor membacanya. Tidak ada hot reload.

## Endpoint dan scope

| Method | Path | Scope |
|---|---|---|
| GET | `/v1/models` | Alias milik default upstream |
| GET | `/providers/{provider}/v1/models` | Alias milik upstream spesifik |
| POST | `/v1/chat/completions` | Generasi non-stream pada default upstream |
| POST | `/providers/{provider}/v1/chat/completions` | Generasi non-stream pada upstream spesifik |

Semua route melewati `RequireGateway` S02 sebelum lookup/body read. Token admin/upstream, token salah/tidak ada, dan Authorization ambigu tetap menghasilkan 401 tanpa dispatch. Listener ini tidak menyediakan endpoint admin. Registrasi route memakai komponen `http.ServeMux`; HEAD pada kedua route katalog memberikan header tanpa body atau dispatch. Penanganan seluruh path/method yang tidak dikenal belum merupakan kontrak error lengkap S05.

Katalog lokal tidak menghubungi provider atau melakukan probe. Contoh bentuk respons:

```json
{"object":"list","data":[{"id":"coding-default","object":"model","owned_by":"primary"}]}
```

Urutan mengikuti konfigurasi. Katalog hanya menampilkan alias, tipe object dan ID upstream pemilik. Model nyata, origin/prefix, relay dan key tidak dimuat. Alias yang sama boleh mempunyai model nyata berbeda pada dua upstream; endpoint menentukan scope dan tidak mencari alias lintas provider. Provider unknown ditolak dengan `404 upstream_not_found`, tanpa fallback default.

## Kontrak JSON

Request generasi memakai `Content-Type: application/json`; parameter media type diperbolehkan. Encoding hanya absent atau `identity`, mengikuti boundary S02. Body harus satu objek JSON lengkap dengan UTF-8 valid. `model` wajib string alias nonkosong, maksimal 128 byte. `stream` absent/false diterima; `stream:true` menghasilkan `501 unsupported_api` sampai S04. Tipe stream lain ditolak.

Nama kontrol dibandingkan setelah decoding escape, sehingga `model` dan `m\u006fdel` bukan dua keputusan berbeda. Duplikat top-level `model` atau `stream` ditolak, termasuk nama escaped. Nested `model`/`stream` dan field tambahan bukan kontrol routing. Parser memvalidasi sintaks menggunakan stdlib lalu menentukan offset satu nilai model; tidak membangun tree/map seluruh request.

Hanya rentang nilai model diganti dengan string JSON model nyata dari registry. Byte lainnya dipertahankan: whitespace, urutan, nested fields, tools/tool choice, Unicode, null, duplicate extension keys, angka besar dan eksponen seperti `1e1000`. Tidak ada konversi angka melalui float64. Raw field seperti URL/target di dalam JSON tetap data untuk upstream dan tidak menjadi tujuan jaringan gateway. Alias unknown menghasilkan `404 model_not_found` sebelum dispatch.

Perilaku duplicate names dan presisi angka perlu ditangani eksplisit karena decoder JSON mempunyai semantik berbeda; acuan [RFC 8259 §§4/6/8.3](https://www.rfc-editor.org/rfc/rfc8259.html) dan [dokumentasi encoding/json](https://pkg.go.dev/encoding/json). S03 memakai API stdlib pada toolchain Go yang sudah dipin repository; tidak menambah dependency parser.

## Admission dan buffer

Reservasi request global dan byte dilakukan sebelum membaca/mengalokasikan body. Jumlah request aktif, anggaran body buffer, slot primary relay dan slot dispatch outbound berasal dari konfigurasi tervalidasi. Semua check/update kapasitas berada di satu pemilik admission. Penolakan langsung, tanpa antrean. Slot relay baru diambil setelah alias tervalidasi dan sebelum dispatch.

Anggaran per generasi bersifat konservatif:

```text
2 × (max_body_bytes + 1) + 2 × metadata_terbesar + 256 KiB
```

Dua buffer mencakup body wire dan hasil remap. Metadata terbesar menghitung paling banyak satu upstream: 128 byte per struct model, 16 byte per referensi relay, dan 16 byte per elemen fallback. Faktor dua mencakup copy-out lookup coordinator dan request boundary. Asumsi ukuran sesuai host amd64 D-006. Cadangan 256 KiB mencakup scanner JSON, alokasi kecil/header/key dan copy respons 32 KiB. Katalog memakai `2 × metadata_terbesar + 256 KiB`; tidak membuat buffer input body.

Anggaran adalah batas konservatif komponen, bukan pengukuran RSS Go/GC/TLS/socket atau bukti Q-5. Budget agregat yang kecil dapat menolak request dengan 429 meskipun body di bawah batas individual; tidak ada pelonggaran diam-diam. Batas individual ditegakkan pada byte aktual yang dibaca dan ukuran setelah remap, bukan hanya Content-Length. Representasi melewati batas menghasilkan 413 sebelum dispatch.

Admission ini hanya kapasitas statis. `Assign` membatasi request yang sedang memakai slot relay/dispatch outbound; tidak menghitung socket idle milik adapter. Batas TCP/socket, listener inbound, header/read/write timeout, serta health eligibility/cooldown/binding menunggu transport dan perakitan runtime. Tidak ada klaim bahwa relay terkonfigurasi sudah sehat. Jumlah koneksi fixture dibatasi oleh adapter/fixture dan diperiksa lewat observasi peer.

## Dispatch dan respons

Coordinator memakai `RequestBoundary.ChatCompletions` S02 untuk target HTTPS/prefix dan key server. Pemilihan relay hanya primary dari mapping fixed/none. Header client tidak memilih origin/key; metadata idempotency/replay tidak diteruskan dan `GetBody` tetap nil setelah remap. Header Content-Type JSON yang diperlukan server dipasang kembali bila nominasi hop-by-hop menghapusnya. Tidak ada fallback atau loop retry coordinator.

Dispatcher wajib menjalankan SOCKS5 terpilih, menyerahkan hostname tujuan melalui proxy, memverifikasi TLS hostname, menolak redirect/direct fallback/retry POST ambigu, dan menghormati cancellation/ownership body. [Kontrak net/http.Transport](https://pkg.go.dev/net/http#Transport) menjelaskan dukungan SOCKS5 dan kondisi retry internal; fixture menguji koneksi fresh dan reused dengan POST yang diterima upstream lalu diputus sebelum header. Interface tidak membuktikan adapter arbitrer aman. **S03 hanya membuktikan wiring adapter fixture tepercaya; transport jaringan runtime belum diimplementasikan.**

Status, Content-Type dan body respons upstream diteruskan, termasuk usage/field tambahan bila tersedia. Content-Encoding yang masih tersedia juga diteruskan agar body terkompresi tetap dapat dibaca client; adapter yang sudah mendekompresi body wajib menghapus header encoding yang tidak lagi berlaku. Model pada respons tidak dipetakan kembali. Usage absent tidak dibuat nol. Retry-After valid diteruskan; cookie, credential, Location dan header operasional upstream tidak disalin. Error gateway memakai request ID acak server dan `X-Infinity-Error-Source: gateway`; respons upstream memakai source `upstream`. Tidak ada error JSON/URL/network mentah atau potongan request dalam envelope gateway. Timeout sebelum respons dibedakan sebagai 504; transport failure sebagai 502.

Penerusan memakai copy buffer tetap 32 KiB, bukan buffer seluruh respons JSON. Partial-body error menghentikan framing HTTP dengan `http.ErrAbortHandler`; tidak menambah envelope, generasi pengganti atau EOF sukses palsu. Bukti streaming SSE mempunyai [kontrak S04 tersendiri](streaming-sse.md).

Request global, byte dan slot relay tetap dipegang sampai caller selesai meneruskan respons dan menutup body. EOF/error/cancellation menutup body jaringan tepat sekali, tetapi tidak membebaskan lease sebelum write client selesai. Caller `Models`/`Chat` wajib memanggil `response.Body.Close()` juga sesudah EOF; handler HTTP memasangnya sebagai defer. Close berulang tidak menggandakan release. Cancellation/timeout menutup jaringan aktif; batas waktu baca/tulis listener tetap tanggung jawab perakitan aplikasi.

## Privasi

Boundary, coordinator dan handler tidak menulis log prompt/output/tool/session/key. Error komponen hanya kategori tetap. Raw outbound request dan `response.Request` dapat mengandung API key; hanya kode server tepercaya boleh menggunakannya, tanpa dump. Guard token tidak diganti atau dilonggarkan.

Batas refleksi upstream mengikuti D-021: tidak ada scrubber yang mengubah body respons. Payload upstream diteruskan utuh, termasuk jika upstream sengaja merefleksikan data sensitif. Jaminan di sini adalah gateway tidak menyisipkan secret/input/error mentah miliknya ke respons/log sendiri, bukan penghapusan data yang dikembalikan upstream.

## Bukti dan batas penerimaan

| AC S03 | Bukti komponen/fixture | Penerimaan produk |
|---|---|---|
| AC1 | Katalog default/provider lokal; alias scope benar; guard empat route; nol CONNECT/dispatch untuk katalog/auth invalid | Parsial; listener/client/deployment terintegrasi belum diuji |
| AC2 | Dua HTTPS upstream menerima model/key/prefix/relay benar; status/content type/body/usage identik; usage absent tetap absent | Parsial; provider/WARP/OpenCode nyata belum diuji |
| AC3 | Byte selain nilai model tetap; nested/tools/Unicode/angka besar/duplicate extension keys tersedia; kontrol ambigu ditolak | Parsial; kompatibilitas upstream nyata belum diuji |
| AC4 | Alias unknown 404 model_not_found dan nol CONNECT/dispatch; provider unknown tanpa fallback default | Parsial; penerimaan gateway terintegrasi tetap terbuka |

Tes unit berada di `internal/gateway/*_test.go`, `internal/pool/admission_test.go` dan `internal/httpapi/api_test.go`. Fixture lintas modul berada di `tests/integration/models_test.go` dan `models_fixture_test.go`, memakai helper HTTPS/SOCKS5 yang sudah tersedia. Data sintetis; tidak ada provider/key nyata atau lifecycle WARP/deployment aktif.

Hasil verifikasi setelah perbaikan review:

- `go test -json -count=1 ./...`: 10 package lulus, 96 tes utama dan 295 subtes lulus; satu native opt-in smoke dilewati.
- `go test -race -count=1 ./...` dan `go vet ./...`: lulus.
- `go test -race -count=30 -run '^TestModel(Admission|CoordinatorLifecycle|Cancellation|AmbiguousPOST|EncodedJSON|APIHeadCatalogue)' ./internal/pool ./internal/gateway ./internal/httpapi ./tests/integration`: lulus tanpa kegagalan.
- Fuzz `FuzzModelJSON` selama 30 detik dengan dua worker: 30.305 eksekusi tanpa kegagalan. Parser tidak berubah dalam perbaikan HEAD/encoding respons.
- Format/diff serta 794 tautan lokal dan 335 anchor: lulus.

Review strict selesai dengan verdict **APPROVE** setelah dua putaran. Dua kasus tepi diperbaiki dengan regresi: HEAD katalog dan metadata Content-Encoding untuk body gzip. Keduanya menjalani RED/GREEN, verifikasi ulang di atas, dan review perbaikan; tidak ada temuan terbuka. Pengerjaan memakai tier high tanpa eskalasi, 12 siklus TDD dan lima subagent analisis/review. Tidak ada deviasi rencana, commit atau push.

Toolchain: `go1.27.4-X:nodwarf5` Linux amd64, bukan verifikasi provenance archive resmi T02. Sandbox awal menolak socket loopback dan cache fuzz; pemeriksaan tersebut berhasil setelah dijalankan di luar sandbox. Native opt-in smoke tidak dijalankan; kelulusan fixture tidak menggantikan M0 atau bukti performa.
