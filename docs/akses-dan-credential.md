# Akses model dan credential server

Tanggal: 24 November 2026  
Task: [EP-02-S02](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s02)  
Status: kontrol token/role dan persiapan request dibuktikan sebagai komponen independen; gateway terintegrasi belum tersedia

Pengguna menyetujui pengerjaan independen dan kontrak terbatas melalui [D-021](DECISIONS.md#d-021-kontrol-token-dan-credential-independen-ep-02-s02). Input memakai [snapshot konfigurasi T01](konfigurasi.md). Dependensi T01/T06 dan gerbang M0 tetap terbuka. Hasil ini tidak menyatakan katalog/generasi produk, restart proses, integrasi client atau WARP/provider lulus.

## Tanggung jawab modul

| Modul | API | Batas |
|---|---|---|
| `internal/access` | `New(snapshot)`, `Gateway(candidate)`, `Admin(candidate)`, `Ready()` | Verifier dua role; tidak membaca environment/file, membuat sesi, atau mengolah HTTP |
| `internal/httpapi` | `RequireGateway(policy, next)`, `RequireAdmin(policy, next)` | Membentuk guard HTTP yang menolak sebelum memanggil handler berikutnya; constructor menolak policy/handler nil termasuk handler bertipe nil |
| `internal/transport` | `NewRequestBoundary(snapshot)`, `ChatCompletions(in, upstreamID)` | Menyiapkan pesan outbound dari target/key server; tidak menjalankan jaringan, memilih relay/model, mengolah JSON/SSE atau menangani respons |

Constructor hanya menerima snapshot tervalidasi. Verifier menyimpan digest di closure privat dan membandingkan SHA-256 candidate/expected dengan `crypto/subtle.ConstantTimeCompare` pada dua digest 32 byte. Candidate kosong atau lebih dari 4 KiB ditolak sebelum hash. Ini menghindari perbandingan berdasarkan prefix; tidak mengklaim seluruh parser atau request mempunyai waktu eksekusi konstan. Raw token maupun digest tidak menjadi field yang dapat ditelusuri fmt, termasuk melalui field privat struct pembungkus.

Request boundary memiliki salinan value snapshot sendiri dan membangun catalog privat dari snapshot tersebut. Lookup target dan key selalu memakai versi yang sama. Mengganti pointer/isi object konfigurasi milik caller setelah constructor tidak mengubah snapshot boundary. Setter/hot reload tidak disediakan. Caller tidak boleh menulis object sumber secara serentak ketika constructor membacanya.

Pemakai server tepercaya harus memasang guard sebelum lookup/dispatch dan memasang role yang sesuai pada setiap endpoint. Guard tidak menjalankan registrasi route/listener atau memverifikasi jaringan deployment. Browser login, cookie dan TTL sesi admin tetap milik EP-05-T01/S02.

## Profil header autentikasi

- Semua endpoint model memakai role gateway, termasuk katalog default/provider dan generasi default/provider. Token admin bukan token model; token upstream bukan token gateway/admin.
- Tepat satu field `Authorization` dengan tepat satu value. Nama header dan scheme `Bearer` tidak membedakan huruf besar/kecil; token tetap case-sensitive.
- Setelah scheme terdapat tepat satu SP. Sisa value merupakan token opaque yang cocok penuh, tanpa trim, split-list atau pemilihan credential pertama/terakhir.
- Token mengikuti T01: printable ASCII 33–126 tanpa whitespace; gateway/admin minimal 24 byte dan maksimal 4.096 byte. Value header maksimal 4.103 byte termasuk `Bearer `.
- Koma, quote, titik dua dan backslash di token opaque diterima hanya jika seluruh token tepat sama dengan konfigurasi. Bentuk `Bearer a, Bearer b` mengandung whitespace dan ditolak. Profil lokal ini tidak mengklaim grammar b64token/OAuth lengkap menurut [RFC 6750 §2.1](https://www.rfc-editor.org/rfc/rfc6750.html#section-2.1).
- Duplikat ditolak meskipun nilainya sama, termasuk nama map Go dengan casing berbeda dan value slice berisi lebih dari satu item. Normalisasi OWS pada wire oleh parser HTTP Go tidak mengganti kecocokan token; pemeriksaan whitespace berlaku pada value hasil parse.
- Query, cookie, body, Proxy-Authorization, X-API-Key dan forwarded role tidak menjadi sumber autentikasi alternatif. Header tersebut tidak menyelamatkan Authorization yang tidak valid.

Cap 4.103 byte hanya membatasi value Authorization yang diperiksa guard. Batas total header, koneksi dan timeout pembacaan listener tetap perlu ditegakkan aplikasi server kelak. Guard tidak membaca, menampung, menutup atau mengambil ownership body request.

## Penolakan HTTP

Semua kegagalan autentikasi model menghasilkan 401 dan challenge `Bearer realm="gateway"`. Admin menghasilkan 401 dengan realm `admin` bagi credential selain token admin; tidak membedakan gateway token dari token salah. Tidak ada pemanggilan handler berikutnya pada penolakan.

```json
{
  "error": {
    "message": "Token tidak valid.",
    "type": "authentication_error",
    "code": "invalid_gateway_token"
  },
  "request_id": "req_<32hex>"
}
```

Guard admin memakai `invalid_admin_token`. Request ID berasal dari 16 byte acak server, bukan header/query client. Header respons menyertakan `X-Request-ID` yang sama dengan envelope, `X-Infinity-Error-Source: gateway`, `Content-Type: application/json`, `Cache-Control: no-store` dan `X-Content-Type-Options: nosniff`.

Envelope mengikuti [kontrak error](PRD-technical-notes.md#14-error-contract) untuk error autentikasi saja. Penanganan seluruh error generasi/transport dan request ID sepanjang lifecycle tetap pekerjaan task pemilik. Modul tidak menulis log request/Authorization/body dan tidak membungkus error mentah parser/OS/network. Kegagalan penulisan respons tidak dicatat bersama data sensitif.

## Persiapan request upstream

`ChatCompletions` menerima request POST dan ID upstream. ID kosong memilih default yang telah divalidasi; ID tak dikenal menghasilkan error tetap `upstream_not_found`, tanpa fallback. Boundary nil/zero, request nil, method lain, panjang body kurang dari -1, atau body nil/NoBody dengan panjang bukan nol ditolak. Error tidak menyalin ID, URL, header, body atau credential input.

Target berasal dari `base_url`/prefix server dengan suffix tetap `/chat/completions`. URL, Host, query dan RequestURI inbound tidak memilih tujuan. Prefix termasuk encoding yang diizinkan T01 dipertahankan. Boundary tidak melakukan model mapping atau membaca JSON/body untuk memilih upstream.

Request/header outbound dibuat baru. Authorization berisi tepat satu `Bearer <API key server>` untuk upstream terdaftar. Header input tidak dapat memilih key atau origin. Aturan penerusan:

- Hanya Content-Type dan Accept. Content-Type mempunyai satu value; Accept boleh beberapa value dalam satu field canonical. Map key ganda dengan casing berbeda ditolak.
- Maksimal 8.192 byte agregat pada field terpilih, dihitung sebagai panjang nama + panjang value + 4 byte untuk `: ` dan CRLF per value. CR/LF/NUL, control selain HTAB, dan non-ASCII ditolak.
- Content-Encoding hanya absent atau satu value `identity`; encoding lain ditolak agar penyaringan header tidak membuat body salah diinterpretasi.
- Connection dibaca hanya untuk mengenali field hop-by-hop yang dinominasikan client. Daftar ini dibatasi 8.192 byte dengan accounting field yang sama; field terpilih yang dinominasikan tidak diteruskan. Authorization server tetap dipasang sesudah penyaringan.
- X-Relay-Target dan X-Relay-Path ditolak eksplisit, tanpa membedakan casing. Cookie, Proxy-Authorization, key vendor/legacy, Forwarded/X-Forwarded-*, Host input, header lain, header hop-by-hop dan idempotency tidak disalin.
- Header beta/provider tambahan belum dijanjikan; dukungan memerlukan allowlist eksplisit pada task integrasi.

RequestURI outbound kosong, Host mengikuti target, Trailer/TransferEncoding client tidak disalin, dan GetBody nil. Body, ContentLength serta cancellation context dipertahankan; persiapan tidak membaca/menutup body. HTTP client pemakai menentukan framing dan memiliki cleanup body sesudah dispatch. Persiapan ini belum membuktikan enforcement body/buffer/slot, transport retry, cancellation upstream atau release runtime.

Raw `*http.Request` hasil persiapan sengaja mengandung API key. Hanya transport server tepercaya boleh menggunakannya. Jangan menampilkan raw request/response, `Response.Request`, Authorization atau error jaringan mentah. Redaksi Config/Secret/boundary tidak membuat objek HTTP tersebut aman untuk dump. Transport pemakai kelak tetap wajib SOCKS5, DNS tujuan melalui proxy, verifikasi TLS hostname, larangan redirect/direct fallback dan retry ambigu; boundary tidak membuat HTTP client/default transport sendiri.

## Privasi dan perubahan credential

AC5 pada irisan ini membuktikan credential outbound berasal dari server, token client tidak menjadi key upstream, dan modul tidak menyisipkan secret ke keluaran/error/log milik gateway. Fixture upstream memeriksa credential dalam memory dan mengembalikan payload tetap tanpa key; snapshot fixture hanya berisi counter/boolean/metadata.

Tidak ada scrubber body upstream. JSON/SSE harus mempertahankan semantik; jaminan menghapus secret yang sengaja direfleksikan upstream membutuhkan keputusan kontrak terpisah. Guard/persiapan juga tidak mengamankan handler/logging caller yang sengaja melakukan dump raw HTTP.

Load A dan boundary A tetap memakai key A. Load B dan boundary B memakai key B dengan token gateway serta request client yang sama. Bukti ini adalah pergantian instance/snapshot, bukan hot reload, restart proses atau drain terkontrol. Restart nyata dan kelanjutan strict tetap mengikuti task perakitan/lifecycle dan EP-04.

## Bukti dan pekerjaan yang tersisa

| AC S02 | Bukti komponen/fixture | Penerimaan produk |
|---|---|---|
| AC1 | Gateway token membuka handler fixture empat path model; request generation fixture memakai key server sesuai upstream | Katalog/generasi produk belum tersedia; parsial |
| AC2 | Missing/invalid/wrong-role/duplicate auth menghasilkan 401; next, dispatch, CONNECT dan upstream tetap nol | Pemasangan guard seluruh endpoint produk belum dibuktikan |
| AC3 | Gateway token ditolak pada fixture status/metrics/operasi admin; token admin tidak membuka model | Listener privat/admin produk dan sesi browser belum dibuktikan |
| AC4 | Snapshot/instance baru memakai key baru; client token dan request tetap | Restart/drain proses belum dibuktikan |
| AC5 | Header/key server tepat; error/format/response fixture/log tidak membocorkan credential | Batas own-output di atas berlaku; seluruh alur generasi/error produk belum dibuktikan |

Tes berada di `internal/access/token_test.go`, `internal/httpapi/auth_test.go`, `internal/transport/request_test.go`, serta `tests/integration/credential_test.go`. Unit spy body membuktikan persiapan/guard tidak membaca atau menutup body. Fixture HTTPS/SOCKS5 membuktikan jalur dan key yang benar; raw TCP fixture menguji Authorization ganda pada wire. Harness bukan router/gateway produk dan tidak memetakan model atau meneruskan SSE produk.

Perintah verifikasi: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go test -race -count=30 ./internal/access ./internal/httpapi ./internal/transport`, `go vet ./...`, serta fuzz `FuzzBearerHeader` 30 detik dengan dua worker. Seluruh data sintetis, network hanya fixture loopback, native/provider/WARP/deployment aktif tidak dijalankan. Hasil aktual: 71 tes utama lulus, satu native smoke dilewati; root race dan 30 pengulangan package S02 lulus tanpa kegagalan. Fuzz Bearer mencapai 722.564 eksekusi selama 30 detik tanpa kegagalan. Vet/format/diff serta 786 tautan lokal dan 332 anchor lulus. Toolchain yang dipakai Go distro `go1.27.4-X:nodwarf5` Linux amd64; tidak mengklaim provenance archive resmi T02. Tidak ada dependency baru atau perubahan schema T01.

Catatan baseline: satu run final biasa menemukan `TestMalformedSOCKS5/bad-atyp` gagal pada `CloseWrite` dengan `transport endpoint is not connected` di helper yang tidak berubah. Pemeriksaan terpisah 50 kali lulus; overlay sementara yang menunggu peer menutup koneksi sebelum assertion mereproduksi error yang sama 20/20 kali. Penyebabnya urutan penutupan TCP pada tes malformed lama, bukan jalur autentikasi/credential. Tidak ada perubahan atau pelemahan assertion helper pada S02. Kelulusan race/fuzz/S02 tetap dibedakan dari fluktuasi fixture ini; perbaikannya menjadi tindak lanjut terpisah.
