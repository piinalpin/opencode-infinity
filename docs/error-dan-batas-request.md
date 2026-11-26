# Error dan batas request sebelum dispatch

Tanggal: 26 November 2026  
Task: [EP-02-S05](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s05)  
Status: komponen dan fixture lokal tersedia; penerimaan gateway terintegrasi tetap parsial

[D-024](DECISIONS.md#d-024-error-dan-batas-request-independen-ep-02-s05) menyetujui pekerjaan independen di atas [S02](akses-dan-credential.md), [JSON S03](model-alias-dan-json.md) dan [SSE S04](streaming-sse.md). Dependensi S03/T01/T06/M0 tidak dihapus. Tidak ada executable/listener/deployment produk, transport jaringan runtime atau request provider/WARP nyata dalam pekerjaan ini.

## Boundary HTTP

`NewAPI` tetap merakit `RequireGateway` paling luar. Token tidak valid/role salah menghasilkan 401 sebelum lookup route, validasi path atau pembacaan body. Route katalog/generasi default dan provider tetap memakai handler, snapshot dan pemetaan S03/S04. Fallback tidak membaca body atau membuka jaringan.

| Kondisi | Status dan code | Metadata tambahan |
|---|---|---|
| Token tidak valid | `401 invalid_gateway_token` | `WWW-Authenticate: Bearer realm="gateway"` |
| Path tidak dikenal | `404 endpoint_not_found` | Tidak memantulkan path/query |
| Method salah pada katalog | `405 method_not_allowed` | `Allow: GET, HEAD` |
| Method salah pada generasi | `405 method_not_allowed` | `Allow: POST` |
| Path noncanonical | `400 invalid_request` | Tanpa redirect/Location |
| `/v1/responses` dan `/providers/{provider}/v1/responses` | `501 unsupported_api` | Semua method; tanpa implementasi/fallback generasi |
| JSON/UTF-8/kontrol model atau stream invalid | `400 invalid_json` | Sebelum dispatch |
| Reader body gagal | `400 invalid_request` | Error reader tidak dipantulkan |
| Media type bukan JSON/ambigu | `415 unsupported_media_type` | Sebelum pembacaan body |
| Encoding request unsupported/ambigu | `415 unsupported_content_encoding` | Tanpa decoding |
| Header target/prefix client | `400 forbidden_target_override` | Sebelum pembacaan body/dispatch |
| Alias/provider tidak dikenal | `404 model_not_found` / `404 upstream_not_found` | Tanpa default fallback |
| Body input atau hasil remap oversize | `413 request_too_large` | Sebelum dispatch |
| Kapasitas global/buffer habis | `429 gateway_capacity_exceeded` | Tanpa antrean |
| Slot relay tidak tersedia | `503 no_eligible_relay` | Bukan bukti health runtime |
| Koneksi proxy/upstream gagal | `502 upstream_transport_error` | Tanpa direct fallback/replay |
| Timeout sebelum respons dimulai | `504 upstream_timeout` | Header/total mengikuti S03/S04 |
| Respons generasi 300–399 | `502 upstream_redirect_rejected` | Body ditutup; tanpa follow/replay |

Path diperiksa pada `URL.EscapedPath()` dengan `path.Clean`; segmen mentah `.`/`..` dan slash berulang ditolak sebelum ServeMux membuat redirect. Slash akhir dipertahankan. Escape seperti `%2e`/`%2f` tidak diubah menjadi dot/separator mentah. Matching dan `PathValue` provider tetap dikerjakan `ServeMux.ServeHTTP`, bukan router baru. Path escaped yang menunjuk provider unknown tetap menghasilkan `upstream_not_found`.

Setiap error buatan handler gateway sebelum commit memakai envelope dengan `error.message`, `error.type`, `error.code`, dan `request_id`. ID acak server berbentuk `req_` beserta 32 digit hex, sama dengan `X-Request-ID`; ID client/upstream tidak dipercaya. Header `X-Infinity-Error-Source: gateway`, `Content-Type: application/json`, `Cache-Control: no-store`, dan `X-Content-Type-Options: nosniff` dipasang. Message/type tetap dari server; error mentah, path/query, secret dan body client tidak disalin. HEAD mempertahankan status/header dan tidak mengirim body pada wire.

## Encoding dan ukuran request

Encoding request hanya absent atau `identity`, sesuai boundary S02 dan persetujuan S05. `gzip`, `br`, `deflate`, encoding bertumpuk atau duplikat ditolak 415 sebelum dispatch. Tidak ada decompressor request, sehingga tidak ada jalur kompresi yang melewati batas sesudah decoding. Dukungan decoding baru memerlukan kontrak dan pengujian terpisah.

`max_body_bytes` tetap input konfigurasi eksplisit T01, bukan default baru S05. Pemeriksaan meliputi Content-Length bila diketahui, jumlah byte aktual termasuk input chunked/unknown length, serta ukuran JSON sesudah alias diganti dengan model nyata. Pembacaan dibatasi max+1; reservasi request/buffer terjadi sebelum alokasi/read dan slot relay sebelum dispatch. Anggaran konservatif S03 tetap berlaku. Tepat batas hasil mapping diterima bila kapasitas cukup; input yang masih muat tetapi hasil mapping melewati cap ditolak 413. Jumlah dispatch pada penolakan tersebut nol.

## Target, redirect dan error upstream

Origin/prefix, key upstream dan relay tetap dari snapshot server. Header `X-Relay-Target`/`X-Relay-Path` ditolak; URL/query client tidak menjadi selector target dan tidak diteruskan sebagai query upstream. Field tambahan JSON tetap data untuk upstream, bukan instruksi memilih jaringan gateway.

Adapter yang diinjeksi tetap wajib memakai SOCKS5 terpilih, menyerahkan hostname upstream melalui proxy, memverifikasi TLS hostname dan menolak follow redirect/direct fallback/replay POST ambigu. Fixture `controlClient` membuktikan kontrak tersebut; pemeriksaan coordinator tidak dapat memperbaiki adapter yang sudah mengikuti redirect. Tidak ada HTTP client/transport direct bawaan.

Sesudah satu dispatch, coordinator menolak semua respons generasi 300–399 sebelum transfer ownership/commit, menutup body tanpa membacanya dan melepas lease melalui jalur defer yang sama. Rentang ini sengaja termasuk 304 meskipun 304 bukan perpindahan origin: request generasi tidak meneruskan conditional headers. Ini menggantikan passthrough 307/body historis S03, dengan tetap membuktikan nol koneksi/credential ke origin redirect.

Error upstream 4xx/5xx mempertahankan status dan byte body, termasuk non-JSON/binary dan gzip opaque. Tidak menambahkan envelope atau scrubber body upstream, sesuai D-021/D-022. Content-Type dan Content-Encoding yang masih berlaku mengikuti allowlist S03; `X-Request-ID` server dan `X-Infinity-Error-Source: upstream` membedakan asal error. Location, cookie, authorization dan ID/source buatan upstream tidak disalin. Hanya satu nilai `Retry-After` yang valid sebagai delta detik atau HTTP-date diteruskan; nilai invalid, overflow, list atau duplikat dibuang. Ini tidak menambahkan cooldown, rotasi relay/key atau retry.

Setelah respons mulai diteruskan, aturan S04 tetap berlaku: kegagalan memutus framing HTTP, tanpa envelope pengganti, DONE palsu atau generasi ulang. Lease aktif sampai caller selesai meneruskan respons dan menutup body.

## Bukti dan batas penerimaan

| AC | Bukti otomatis | Batas |
|---|---|---|
| AC1 | `TestModelAliasJSON`, `TestModelNetworkBoundaries`, `TestGatewayRequestErrors`: ledger relay/hostname, TLS, nol direct | Adapter fixture; bukan WARP/provider runtime |
| AC2 | `TestGatewayRequestErrors`, `TestGatewayErrorRedirect`, `TestModelCoordinatorRejectRedirect`: target ditolak, 3xx terstruktur, origin lain nol, satu dispatch dan cleanup | Tidak membuktikan adapter arbitrer aman |
| AC3 | `TestGatewayRequestErrors`, `TestGatewayErrorBodyRead`, regresi parser/batas S03: malformed input/coding/size/remap, nol CONNECT sebelum dispatch, batas valid diterima | Request absent/identity saja |
| AC4 | Tes `TestGatewayError*` pada boundary dan wire, regresi timeout/partial JSON/SSE | Error parser HTTP/TLS/server sebelum handler di luar scope |

Sumber tes berada di `internal/httpapi/errors_test.go`, `internal/gateway/coordinator_test.go`, `tests/integration/errors_test.go`, serta suite S03/S04 existing. Seluruh token/key adalah nilai sintetis. Verifikasi pada Go `go1.27.4-X:nodwarf5 linux/amd64`:

- `go test -count=1 ./...`: lulus seluruh 10 package, termasuk regresi S03/S04 dan fixture spike lokal. Native WARP tetap opt-in dan tidak diaktifkan.
- `go vet ./...`: lulus. `gofmt -l internal tests/integration` kosong; `git diff --check` lulus.
- `go test -race -count=1 ./...`: lulus seluruh 10 package.
- `go test -race -count=20 ./internal/httpapi ./internal/gateway ./internal/pool ./tests/integration -run 'TestGatewayError|TestGatewayRequestErrors|TestModelCoordinatorRejectRedirect|TestModelCoordinatorLifecycle|TestModelAdmissionConcurrent|TestModelNetworkBoundaries|TestModelAmbiguousPOST|TestGatewayStreamAmbiguousPOST'`: 20 pengulangan per tes terpilih, tanpa kegagalan/race. Ini pengujian error/lifecycle, bukan benchmark performa.

Enam siklus TDD membuktikan RED/GREEN untuk 404, 405/Allow, path noncanonical, Responses unsupported, penolakan 3xx dan validasi `Retry-After` duplikat. Coverage tambahan membuktikan perilaku input/body/upstream existing tanpa memperluas scope. Review diff strict inline selesai dengan verdict **APPROVE**, tanpa temuan terbuka. Tiga subagent menganalisis kebutuhan, state dan tes; implementasi/review dikerjakan inline, tanpa eskalasi tier atau deviasi rencana.

Pengujian fixture tidak membuktikan default listener, deadline inbound body yang macet, transport runtime, integrasi OpenCode/provider/WARP, health/cooldown/binding, performa/RSS atau kelulusan M0/MVP. Parser HTTP/TLS dan respons server sebelum handler, termasuk OPTIONS `*`, memerlukan verifikasi saat aplikasi listener dirakit.
