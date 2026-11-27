# Konfigurasi dan registry gateway

Tanggal: 24 November 2026  
Task: [EP-02-T01](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-t01)  
Status: modul dan pengujian offline; startup/restart gateway terintegrasi belum dibuktikan

Kontrak `version: 1` ini mengadaptasi contoh historis [catatan teknis](PRD-technical-notes.md#11-konfigurasi-dan-credential) sesuai persetujuan rencana EP-02-T01 dan [D-020](DECISIONS.md#d-020-konfigurasi-dan-registry-offline-ep-02-t01). Persetujuan pengerjaan independen tidak menghapus dependensi EP-01-T06 atau meluluskan M0. Registry yang tervalidasi tidak membuktikan ketersediaan relay/provider, autentikasi endpoint, transport, ataupun penegakan anggaran runtime.

## Pemakaian modul

`internal/config.Load(reader, sources)` memuat dan memvalidasi seluruh input, kemudian mengembalikan satu snapshot atau `nil` dan error. Urutannya adalah pembacaan terbatas, struktur/schema, versi, referensi/routing, URL/listener, batas numerik, lalu semua secret. Tidak ada DNS, dial, probe, pembacaan environment host implisit, listener, watcher atau hot reload.

Pemakai menyediakan `Sources.LookupEnv`, misalnya `os.LookupEnv` saat perakitan aplikasi kelak; tes memasok lookup sintetis. `Sources.OpenFile` yang kosong memakai pembaca file reguler bawaan. Override pembuka file adalah dependensi tepercaya untuk pengujian atau perakitan, bukan nilai dari YAML; pembaca tetap membatasi byte dan memeriksa error baca/tutup. Pemakai menutup reader konfigurasi yang dibukanya sendiri.

`internal/catalog.New(snapshot)` hanya menerima snapshot hasil loader yang berhasil, lalu membentuk indeks metadata privat. `Resolve(upstream, alias)` memakai default hanya jika parameter upstream kosong; upstream/alias yang tidak ditemukan menghasilkan `false`, tanpa fallback. `Upstream` dan `Relay` menyediakan metadata registry. Getter config/catalog mengembalikan salinan, termasuk daftar relay/model/fallback. Catalog tidak membawa credential.

Snapshot yang sudah dimuat tidak mengikuti perubahan reader, environment atau file. Pemuatan berikutnya menghasilkan snapshot baru yang lengkap; kegagalan tidak mengubah hasil lama. Perakitan aplikasi kelak harus memuatnya sebelum menerima trafik dan memakai satu versi sepanjang proses. Modul ini belum memasang gerbang startup pada server produk dan belum membuktikan restart/drain proses.

## Format dan batas input

Contoh lengkap: [examples/infinity.yaml](../examples/infinity.yaml). Angka contoh adalah profil provisional untuk pengembangan/pengujian, bukan default numerik produk final atau konfigurasi deployment pengguna.

- Satu dokumen mapping YAML dengan `version: 1`; input kosong, dokumen tambahan termasuk kosong, trailing rusak, unknown field dan duplicate key ditolak.
- Tipe scalar mengikuti field: integer untuk angka, string untuk identifier/URL/reference/mode, boolean canonical true/false untuk stateless sejak S03. Nilai `null` dan string kosong eksplisit ditolak, termasuk pada field opsional; menghilangkan field opsional berbeda dari memberi nilai kosong.
- Anchor, alias, merge key dan custom tag ditolak. Komentar/whitespace akhir diperbolehkan. Tidak ada interpolasi environment/shell dalam isi YAML.
- Maksimal 1 MiB input, dibaca paling banyak cap+1; depth 32, 20.000 node dan 4 KiB per scalar. Batas depth diberlakukan juga oleh parser sebelum konstruksi struct. Guard ini membatasi input konfigurasi, bukan membuktikan RSS runtime.
- Maksimal 128 relay, 64 upstream, 256 model per upstream, 1.024 model total, serta 128 referensi pada setiap daftar relay. Minimal satu relay, satu upstream dan satu model per upstream.
- ID relay/upstream maksimal 64 byte, alias 128 byte: ASCII huruf/angka, diikuti huruf/angka atau `.`, `_`, `-`. Nama model upstream maksimal 256 byte printable ASCII tanpa whitespace; `/` dan `:` provider diperbolehkan.

Loader memakai `go.yaml.in/yaml/v4 v4.0.0`, pin [T02](M0-stack-dan-versi.md#2-versi-exact-yang-dipilih), dengan unknown/duplicate field checking. AST diperiksa sebelum konversi struct. API `Load` mensyaratkan tepat satu dokumen; opsi `WithSingleDocument` tidak digunakan karena mengabaikan dokumen berikutnya. Rujukan: [API exact pin](https://pkg.go.dev/go.yaml.in/yaml/v4@v4.0.0), [source exact pin](https://github.com/yaml/go-yaml/blob/v4.0.0/yaml.go).

## Field dan default

| Bagian | Field | Aturan |
|---|---|---|
| Root | `version`, `server`, `limits`, `timeouts`, `relays`, `upstreams` | Wajib; versi hanya `1` |
| `server` | `listen`, `admin_listen` | Literal IPv4/IPv6 dan port 1–65535; alamat yang bertumpang tindih pada port yang sama ditolak |
| `server` | `gateway_token_env` / `gateway_token_file`, `admin_token_env` / `admin_token_file` | Tepat satu sumber tiap token, lihat kontrak secret |
| `server` | `default_upstream` | Boleh dihilangkan hanya jika satu upstream; jika lebih dari satu wajib eksplisit menunjuk ID valid |
| `server` | `max_body_bytes`, `max_active_requests` | Wajib eksplisit dan positif; body maksimal 10 MiB |
| `relays[]` | `id`, `proxy_url`, `max_inflight` | Wajib; ID unik global; slot positif |
| `upstreams[]` | `id`, `base_url`, `api_key_env` / `api_key_file`, `relays`, `models` | ID unik global; satu sumber key; allowlist relay tidak kosong dan tidak ganda |
| `models[]` | `alias`, `upstream_model`, `primary_relay` | Wajib; alias unik per upstream; primary anggota allowlist upstream |
| `models[]` | `stateless` | Opsional boolean canonical true/false, default false; least_inflight mensyaratkan true |
| `models[]` | `routing`, `affinity`, `fallback_relays` | Jika dihilangkan menjadi `fixed`, `none`, dan daftar kosong; nilai eksplisit harus valid |

Sejak [EP-03-S03/D-028](least-inflight.md), routing menerima fixed atau least_inflight, hanya dengan affinity none. Least-inflight wajib stateless true eksplisit; absent/false, strict dan mode unknown ditolak. Boolean stateless hanya true/false huruf kecil; string, yes/no/on/off, angka, null dan uppercase boolean ditolak sebelum decode typed. Fixed tetap default, termasuk jika stateless true. Kandidat least memakai urutan allowlist upstream.relays; primary tetap wajib valid, dan fallback tidak mengaktifkan retry. Semua referensi relay harus ada; fallback anggota allowlist, tidak ganda dan tidak memuat primary. Alias yang sama boleh dipakai upstream berbeda. Kontrak dan contoh fixture opt-in tersedia pada [dokumen least-inflight](least-inflight.md#konfigurasi-dan-kandidat); contoh infinity.yaml tetap fixed.


Listener contoh memakai loopback. Wildcard yang dipilih secara eksplisit hanya mengatur bind proses/container. Publish port host tetap loopback sesuai [D-006](DECISIONS.md#d-006-host-uji-amd64-dan-tailscale-di-host); parser tidak memvalidasi Compose/Tailscale atau menetapkan listener/tailnet deployment aktif.

## URL target

Upstream harus URL HTTPS absolut dengan hostname DNS ASCII atau literal IP valid. Port opsional berada pada 1–65535. Userinfo, query termasuk `?` kosong, fragment termasuk `#` kosong, opaque URL, whitespace/control mentah dan backslash ditolak. Prefix path boleh kosong atau absolut; segmen `.`/`..`, dot/separator/backslash/control terenkode serta encoding `%` bertingkat ditolak. Prefix tidak dinormalisasi diam-diam. Spasi yang di-encode sebagai `%20` dalam path diperbolehkan.

Relay hanya menerima `socks5h://host:port`, termasuk hostname container dan IPv6 berkurung. Port wajib eksplisit. Credential, path termasuk `/`, query dan fragment ditolak. Tidak ada mode koneksi langsung. Penerimaan string `socks5h` tidak membuktikan DNS melalui proxy; TLS hostname, redirect dan larangan direct fallback tetap perlu dibuktikan oleh transport pada task terkait.

## Batas numerik

Semua angka berikut wajib diisi eksplisit sebagai integer positif. Tidak ada fallback ke angka contoh jika field hilang. Count dibatasi `int32` positif; seconds harus dapat dikonversi ke `time.Duration` tanpa overflow. Jumlah bucket byte diperiksa dengan penjumlahan aman.

| Bagian | Field | Nilai contoh provisional |
|---|---|---:|
| `server` | `max_body_bytes` | 10.485.760 |
| `server` | `max_active_requests` | 20 |
| `limits` | `body_buffer_bytes` | 67.108.864 |
| `limits` | `stream_buffer_bytes` | 8.388.608 |
| `limits` | `state_bytes` | 16.777.216 |
| `limits` | `monitoring_bytes` | 8.388.608 |
| `limits` | `inbound_connections`, `outbound_connections` | 64, 32 |
| `limits` | `bindings`, `admin_sessions` | 1.024, 32 |
| `limits` | `parallel_probes`, `parallel_status` | 2, 4 |
| `limits` | `snapshot_bytes` | 262.144 |
| `timeouts` | `connect_seconds`, `response_headers_seconds` | 10, 120 |
| `timeouts` | `stream_idle_seconds`, `total_request_seconds` | 120, 1.800 |
| `relays[]` | `max_inflight` | 5 |

`body_buffer_bytes` setidaknya sebesar `max_body_bytes`. `snapshot_bytes × parallel_status` harus muat dalam `monitoring_bytes`; pengecekan memakai pembagian agar tidak overflow. `parallel_probes` tidak melebihi koneksi keluar. Tidak ada syarat bahwa batas request global harus lebih kecil dari seluruh slot atau satu primary; admission/buffer dapat membatasi pemakaian lebih dahulu.

Contoh hanya memiliki satu relay dan satu alias. Itu bukan fixture benchmark lima relay/lima alias dan tidak menjanjikan 20 stream pada satu primary berkapasitas lima. Empat bucket 64+8+16+8 MiB mengikuti [profil D-019](M0-gerbang-dan-pengukuran.md#3-profil-resource-provisional). Penerimaan konfigurasi tidak membuktikan accounting body/decode, backpressure, batas koneksi/state, RSS, performa atau shutdown. TTL binding/sesi, timeout/backoff probe, default numerik produk final dan enforcement tetap pekerjaan task pemilik.

## Secret dan redaksi

Untuk gateway/admin gunakan `*_token_env` atau `*_token_file`; upstream memakai `api_key_env` atau `api_key_file`. Tepat satu sumber per role/upstream. Tidak ada credential literal dalam YAML, fallback antar-sumber atau pemakaian token client sebagai key upstream.

Nama environment mengikuti `[A-Za-z_][A-Za-z0-9_]*`, maksimal 128 byte. Lookup yang tidak disediakan atau variable tidak tersedia menggagalkan load. Path file harus absolut dan mengarah ke file reguler; symlink mount ke file reguler diperbolehkan. Pembaca bawaan memeriksa jenis file sebelum dan setelah open, memakai `O_NONBLOCK` pada target Linux agar pergantian ke FIFO tidak menggantung pembacaan, dan membaca paling banyak 4.099 byte. Operator tetap harus memasang file read-only dan mengamankan direktori/mount; parser tidak membuktikan izin mount atau menjamin filesystem remote yang bermasalah tidak memblokir.

Nilai secret maksimal 4 KiB, printable ASCII tanpa whitespace, CR/LF/NUL. Token gateway/admin minimal 24 byte; API key minimal satu byte. File boleh satu LF atau CRLF penutup yang dilepas tepat sekali; environment tidak dinormalisasi. File tidak ditulis. Token gateway dan admin harus berbeda, dan key upstream tidak boleh sama dengan keduanya; antar-upstream boleh memakai key yang sama.

Placeholder ditolak tanpa membedakan huruf besar/kecil jika seluruh nilainya adalah `changeme`, `change-me`, `change_me`, `replace-me`, `replace_me`, `your-api-key`, `your-token`, `gateway-token`, `admin-token`, `upstream-key`, `example`, `placeholder`, `todo`, atau `tbd`; juga seluruh bentuk `<...>` atau `${...}`. Substring `test`/`example` di credential lain tidak otomatis ditolak.

`Secret` menampilkan penanda redaksi pada fmt/JSON/YAML/text. `Config` menyembunyikan keseluruhan isi pada fmt; field privat tidak diekspor melalui JSON/YAML. Nilai asli hanya melalui `Reveal()` yang dipanggil eksplisit komponen server tepercaya. Ini bukan secret store atau perlindungan terhadap kode dalam proses yang sengaja membaca nilai. Jangan membuka nilai tersebut untuk log, error, snapshot, atau UI.

Error loader memakai kode tetap, tidak membungkus error parser/OS mentah atau menampilkan raw YAML, key asing, env/path, URL maupun nilai secret. Contoh kategori: `config_invalid_yaml`, `config_invalid_version`, `config_invalid_relay_reference`, `config_invalid_upstream_url`, `config_invalid_secret_source`, `config_secret_unavailable`, `config_invalid_secret`, `config_secret_roles_overlap`. Diagnosis menggunakan konfigurasi operator secara privat; jangan menambahkan dump otomatis.

## Bukti dan batas penerimaan

| AC tiket | Bukti offline | Penerimaan terintegrasi |
|---|---|---|
| AC1 | Loader memuat registry/default/batas dan semua secret; catalog memetakan alias | Startup gateway belum tersedia; parsial |
| AC2 | Kasus negatif schema, duplikasi/referensi/default/URL menolak tanpa hasil parsial | Gerbang sebelum listener/generasi belum dipasang; parsial |
| AC3 | Environment/file kosong, placeholder, unavailable, oversize, role overlap ditolak; error dan format teredaksi | Validasi modul terbukti; tidak menyatakan endpoint auth tersedia |
| AC4 | Load A → invalid B → valid C; nested copy dan perubahan secret tidak mencampur snapshot | Restart/drain proses belum dibuktikan; parsial |

Tes berada di `internal/config/*_test.go`, `internal/catalog/registry_test.go`, dan `tests/integration/config_test.go`. Harness memakai data sintetis; fuzz mematikan pembukaan file eksternal dan hanya memakai environment lookup sintetis. Root suite transport memakai fixture loopback tanpa provider/WARP. Native smoke tetap opt-in dan tidak dijalankan oleh task ini.

Perintah pemeriksaan: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go test -race -count=30 ./internal/config ./internal/catalog`; fuzz `FuzzLoad` serta `FuzzTargetsAndLimits` masing-masing 30 detik dengan dua worker. Hasil aktual dan batas toolchain dicatat pada ringkasan pengerjaan; daftar perintah ini bukan klaim seluruh deployment telah diuji.

### Hasil verifikasi 24 November 2026

- Root suite: 51 tes utama lulus, satu native smoke dilewati karena opt-in tidak diaktifkan. Seluruh suite juga lulus dengan race detector.
- Config/catalog: 30 pengulangan dengan race detector, tanpa kegagalan; mencakup lookup/copy, pembacaan secret/snapshot dan load independen serentak.
- Fuzz loader sesudah perbaikan redaksi: 30 detik, 510.230 eksekusi; fuzz URL/limit: 30 detik, 805.336 eksekusi; tanpa kegagalan. Harness loader menonaktifkan pembukaan file eksternal. Run awal loader menghabiskan waktu pada minimisasi; run lanjutan dengan minimisasi dibatasi satu detik menghasilkan progres berkelanjutan. Tidak ada klaim cakupan menyeluruh dari fuzz.
- `go vet`, format Go, `go mod verify`, `git diff --check`, 778 tautan lokal dan 328 anchor lulus. Checksum YAML cocok dengan pin T02; tidak ada dependency produk lain yang ditambahkan.
- Toolchain yang benar-benar dipakai adalah Go distro `go1.27.4-X:nodwarf5`, Linux amd64. Archive Go resmi T02 tidak tersedia pada lingkungan sesi ini; hasil ini tidak mengklaim provenance archive tersebut. Pengujian memakai fixture lokal/sintetis tanpa provider/WARP, browser atau container deployment aktif.

Review menemukan kebocoran fmt untuk secret di field privat struct pembungkus. Regresi tersebut dibuktikan lalu diperbaiki: penyimpanan opaque tidak lagi memiliki field string mentah yang bisa ditelusuri fmt; akses nilai tetap eksplisit melalui `Reveal()`. Tes memeriksa format langsung dan pembungkus privat, bukan perlindungan terhadap pembacaan memory oleh kode tepercaya.
