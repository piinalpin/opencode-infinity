# M0 — Stack, versi, dan bukti kompatibilitas

Tanggal verifikasi: 23 November 2026  
Status: Kombinasi teknis dipilih untuk spike M0; dependensi EP-01-T01 masih terbuka  
Task: [EP-01-T02](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t02)

Pengguna menyetujui pelaksanaan validasi stack yang independen dari hambatan T01 pada D-017. Hasil ini menetapkan kombinasi Go + Svelte untuk pengerjaan berikutnya, dengan bukti kompilasi serta provenance. Ini bukan kelulusan integrasi client, transport WARP, atau seluruh gerbang M0. Batas A-1–A-8 pada [ARCHITECTURE](ARCHITECTURE.md) tetap berlaku.

## 1. Pilihan dan pemetaan arsitektur

| Acuan | Pilihan implementasi | Konsekuensi dan batas |
|---|---|---|
| A-1, modular monolith | Go, standard library, satu binary | Perakitan di `internal/app`; tidak menambah framework server atau service aplikasi |
| A-2, listener terpisah | `net/http`, `http.ServeMux` | Listener generasi dan admin terpisah; kontrak auth browser tetap ditetapkan EP-05-T01 |
| A-3, SOCKS5 langsung | `http.Transport` + `httputil.ReverseProxy` | Konstruktor berhasil dikompilasi; DNS proxy, TLS, flushing, cancellation, dan batas attempt tetap harus dibuktikan T03/T04 |
| A-4, transaksi state atomik | Package `internal/pool` dengan koordinasi internal Go | Tidak menambah cache/database; desain serta pengujian konkurensi tetap EP-03/EP-04 |
| A-5, snapshot server | JSON snapshot; Svelte + TypeScript pada browser | UI membaca kontrak backend, polling 5 detik/stale >15 detik sesuai D-009; tidak menentukan eligibility |
| A-6, aset bersama gateway | Vite menghasilkan aset statis, `embed.FS` memasukkannya ke binary | Fixture membuktikan kompilasi embed; packaging aplikasi dan uji browser tetap T05/EP-05 |
| A-7, state sementara dibatasi | YAML v4 untuk input, memory untuk runtime, `log/slog`, registry Prometheus tersendiri | Decoder strict dan metrik sintetis diuji; schema produk, batas kapasitas, redaksi, dan lifecycle belum diimplementasikan |
| A-8, lifecycle relay oleh operator | MicroWARP dari source yang dipin, Docker Compose pada host `linux/amd64` | Gateway tanpa Docker socket/NET_ADMIN. Tailscale tetap di host; relay tidak dijalankan pada validasi ini |

Svelte memakai scoped CSS dan CSS variables seperti rancangan awal; tidak ditambah UI kit, router, state library, SvelteKit, database, atau server frontend runtime. `prometheus/client_golang` hanya menyediakan metrik dalam aplikasi; server Prometheus/Grafana bukan dependensi MVP.

Alternatif Go + template/JavaScript tetap lebih kecil dari sisi toolchain, tetapi Svelte dipilih untuk interaksi dashboard yang sudah direncanakan. Node.js + Undici tidak dipilih karena source relay HTTP bukan jalur generasi MVP. Pilihan ini tidak didasarkan pada klaim CPU/RAM lebih baik; benchmark produk tetap M3.

## 2. Versi exact yang dipilih

Versi berasal dari metadata rilis resmi yang diperiksa pada tanggal di atas. Semua versi direct dependency pada manifest fixture dipin tanpa `^`, `~`, atau `latest`. Dependency transitif dicatat dalam lock/checksum, bukan ditebak dari versi direct dependency.

| Komponen | Pin | Sumber dan hasil |
|---|---|---|
| Go | `1.27.4` | [Rilis resmi](https://go.dev/dl/?mode=json); archive resmi `linux-amd64` diverifikasi SHA-256, dipakai untuk seluruh pemeriksaan Go |
| Node.js | `26.13.1` | [Indeks rilis](https://nodejs.org/dist/index.json); LTS Lithium menurut [jadwal dukungan](https://github.com/nodejs/Release/blob/main/schedule.json); archive resmi diverifikasi checksum |
| npm | `12.2.2` | [Metadata paket](https://registry.npmjs.org/npm/12.2.2); dipasang pada prefix sementara terpisah, integrity cocok dengan install lock; `packageManager` dan engine fixture dipin |
| Svelte | `5.59.1` | [Metadata paket](https://registry.npmjs.org/svelte/5.59.1); checker dan build lulus |
| TypeScript | `6.0.3` | [Metadata paket](https://registry.npmjs.org/typescript/6.0.3); dipilih karena checker belum menerima major 7 |
| Vite | `8.3.8` | [Metadata paket](https://registry.npmjs.org/vite/8.3.8); engine Node dan peer plugin dipenuhi |
| `@sveltejs/vite-plugin-svelte` | `7.3.1` | [Metadata paket](https://registry.npmjs.org/@sveltejs/vite-plugin-svelte/7.3.1); peer Vite 8/Svelte ≥5.46.4 dipenuhi |
| `svelte-check` | `4.7.8` | [Metadata paket](https://registry.npmjs.org/svelte-check/4.7.8); peer TypeScript `^5.0.0 || ^6.0.0` dipenuhi |
| `@types/node` | `26.6.6` | [Metadata paket](https://registry.npmjs.org/@types/node/26.6.6); pemeriksaan tipe fixture lulus |
| `@playwright/test` | `1.65.1` | [Metadata paket](https://registry.npmjs.org/@playwright/test/1.65.1); instalasi/CLI diverifikasi, browser belum diunduh atau diuji |
| YAML | `go.yaml.in/yaml/v4 v4.0.0` | [API v4](https://pkg.go.dev/go.yaml.in/yaml/v4@v4.0.0); decode valid, unknown field, dan duplicate key diuji |
| Metrik | `github.com/prometheus/client_golang v1.25.0` | [API module](https://pkg.go.dev/github.com/prometheus/client_golang@v1.25.0); registry/gauge/handler sintetis diuji |
| Compose host | `5.7.1` | Lingkungan D-006 dan [hasil T01](M0-lingkungan-dan-integrasi.md); verifikasi ini tidak mengubah instalasi host |

Go pada host teramati sebagai `1.27.4-X:nodwarf5`. Eksperimen memakai distribusi resmi Go `1.27.4` yang diunduh ke `/tmp`, sehingga bukti tidak menyamakan build distro dengan archive upstream. Node resmi juga diekstrak terpisah; instalasi host tidak diganti. Archive Node tersebut membawa npm `11.20.0`; percobaan awal menghasilkan peringatan engine terhadap pin npm `12.2.2`. Setelah npm `12.2.2` dipasang terpisah di `/tmp`, seluruh pemeriksaan frontend diulang tanpa peringatan engine. Hasil aset identik dengan yang sudah diverifikasi melalui embed Go.

Checksum archive yang benar-benar diverifikasi:

| Archive | SHA-256 |
|---|---|
| `go1.27.4.linux-amd64.tar.gz` | `7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840` |
| `node-v26.13.1-linux-x64.tar.xz` | `5c063e46bab0fbc1c17cc445d7ff129cbdcd0471be6ed83b6ab864f8143bc864` |

Go didukung menurut [kebijakan dua major release](https://go.dev/doc/devel/release). Node 26 tercatat LTS sejak 28 Oktober 2026 dan berakhir dukungan 30 April 2029 pada jadwal yang diperiksa. Status dukungan harus diperiksa kembali saat rilis; pin tanggal ini bukan janji kompatibilitas masa depan.

## 3. Pemilihan yang tidak mengikuti latest

- **TypeScript 7.0.2 ditolak untuk kombinasi ini.** Metadata `svelte-check 4.7.8` hanya menerima major 5/6. Percobaan `npm install --package-lock-only --ignore-scripts --strict-peer-deps` pada fixture dengan TypeScript 7 gagal `ERESOLVE`; versi 6.0.3 lolos tanpa `--force`, `--legacy-peer-deps`, atau override peer.
- **YAML v3 ditinjau dan diganti dengan v4.0.0 untuk implementasi baru.** [README pada tag v4.0.0](https://github.com/yaml/go-yaml/blob/v4.0.0/README.md) menyatakan v2/v3 stabil tetapi API dibekukan dan hanya menerima perbaikan keamanan; pengembangan aktif berada di v4. API decoder strict yang diperlukan berhasil diuji. Tidak ada migrasi konfigurasi/kode produk lama karena implementasi belum ada.
- **Alpine 3.21 pada source referensi tidak dipilih sebagai base target.** [Metadata rilis Alpine](https://alpinelinux.org/releases.json) mencatat akhir dukungan branch itu pada 1 November 2026. Target memakai `3.24.4` dan repo `main`, yang tercatat didukung sampai 1 Juni 2028. Seluruh delapan nama paket build/runtime referensi ditemukan pada katalog `main` yang diperiksa; paket belum diinstal atau diuji sebagai relay. Dukungan `community` tidak diasumsikan sama dengan `main`.

Tidak ada downgrade target PRD, pemilihan prerelease, atau klaim bahwa tag `latest` membuktikan kompatibilitas.

## 4. Pin image dan provenance relay

Manifest dibaca dengan `docker buildx imagetools inspect --raw`. Digest berikut menunjuk manifest **`linux/amd64`**, bukan indeks multi-arsitektur. Operasi ini tidak menarik seluruh layer, membangun image aplikasi, atau menjalankan container.

| Peran | Tag saat pemeriksaan | Digest manifest `linux/amd64` |
|---|---|---|
| Builder Go | `docker.io/library/golang:1.27.4-trixie` | `sha256:2cd7e1ebf475c6208122d2b9cefc638d98fcc8dfd8554c8872c9dd0c21f4ac61` |
| Builder UI | `docker.io/library/node:26.13.1-trixie-slim` | `sha256:a5a5adfee3013de54504e0e6aff49993acf7daed072e86be75b8818c0f6ac258` |
| Runtime gateway | `gcr.io/distroless/static-debian13:nonroot` | `sha256:89e00a7e78df216bcbf495e7f43135f03ddbd3e98024a4104731872d6c8cf3c6` |
| Base target relay | `docker.io/library/alpine:3.24.4` | `sha256:0f49f331d9212aff11b932863be4eb01cdf962969aa86dab7d1ee21ac6f5f52b` |

Referensi lengkap `tag@digest` tersedia di [pins.json](m0/ep-01-t02/pins.json). Config runtime pada digest terpilih mencatat OS Linux, arsitektur amd64, dan user `65532`; ini pemeriksaan metadata, bukan uji eksekusi dalam Distroless. Binary fixture yang dibangun dengan `CGO_ENABLED=0` teridentifikasi statically linked. [Distroless](https://github.com/GoogleContainerTools/distroless) dipilih untuk runtime tanpa toolchain atau shell; health subcommand produk tetap pekerjaan implementasi.

MicroWARP diambil sebagai referensi source dari [warp-relay-pool](https://github.com/trefeon/warp-relay-pool/tree/1a58c904d6a99463f2c48cb02234f44eca7646e6):

| Komponen | Identitas sumber | Lisensi/scope |
|---|---|---|
| MicroWARP | Commit `1a58c904d6a99463f2c48cb02234f44eca7646e6` | MIT; Dockerfile, entrypoint, dan LICENSE diperiksa; URL immutable dan checksum ada di pins.json |
| microsocks | Tag `v1.0.5` terselesaikan ke commit `98421a21c4adc4c77c0cf3a5d650cc28ad3e0107` | MIT menurut [COPYING pada commit](https://github.com/rofl0r/microsocks/blob/98421a21c4adc4c77c0cf3a5d650cc28ad3e0107/COPYING) |

Belum ada source relay yang disalin/adopsi sebagai kode produk. Saat T04 mengadopsinya, perubahan lokal yang diperlukan adalah penggantian base Alpine ke digest terpilih dan pin microsocks ke commit, disertai catatan provenance. Source referensi memiliki `STOPSIGNAL SIGKILL`; perilaku relay itu harus ditinjau terhadap prosedur maintenance dan tidak disalin ke gateway yang mengikuti D-014.

Digest final image gateway/relay, graph APK yang benar-benar dipasang, serta perubahan entrypoint belum tersedia karena image produk belum dibangun. Pin base/source dan snapshot katalog paket memberi keterlacakan awal; keduanya tidak menggantikan identitas artefak hasil build atau pengujian WARP. Katalog APK pada pins.json adalah observasi versi dan checksum indeks, bukan lock lengkap dependency APK atau hasil verifikasi signature paket.

## 5. Pemeriksaan yang dijalankan

Semua source fixture, dependency directory, binary, dan hasil build dijalankan di `/tmp/implement-task/ep-01-t02`. Hanya manifest, lock/checksum, source fixture sintetis, serta ringkasan pemeriksaan disimpan sebagai bukti dokumentasi; tidak ada kerangka aplikasi produk yang dibuat.

| Pemeriksaan | Hasil | Batas pembuktian |
|---|---|---|
| Archive toolchain resmi | SHA-256 Go dan Node cocok | Integritas terhadap checksum sumber resmi; tidak menyatakan verifikasi signature archive |
| Resolusi TypeScript 7 | Gagal `ERESOLVE` seperti yang diharapkan | Kasus negatif kompatibilitas dependency, bukan kegagalan build produk |
| Resolusi/instalasi TypeScript 6 | `npm install --package-lock-only` dan `npm ci`, keduanya strict peer, lulus | Lifecycle script npm dinonaktifkan; tidak ada pelonggaran peer dependency |
| Frontend | `npm run check` lulus: svelte-check 0 error/0 warning dan `tsc --noEmit` lulus; `npm run build` lulus | Komponen fixture bertipe dan aset Vite, bukan dashboard AC-20–AC-22 |
| Dependency frontend | `npm ls --depth=0` lulus; CLI Playwright `1.65.1` tersedia | Tidak ada browser/E2E dijalankan |
| Modul Go | `go mod tidy`, `go mod verify`, `go vet ./...` lulus | Memakai proxy dan checksum database resmi, cache khusus fixture |
| Tes Go | `go test -count=1 -v ./...` lulus: 3 tes utama, termasuk 3 subkasus YAML | YAML valid/unknown/duplicate, gauge/handler Prometheus, dan embed index hasil build |
| Build Go | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath` dan eksekusi fixture lulus | Binary statis terverifikasi `file`; tidak ada dispatch jaringan |
| Image | Empat manifest amd64 dan metadata user runtime diperiksa | Tidak ada image produk/relay dibangun atau container dijalankan |

Detail command dan exit code tersedia di [checks.json](m0/ep-01-t02/checks.json). Ukuran bundle fixture bukan ukuran dashboard produk; tidak ada benchmark RSS, overhead, atau kapasitas yang diklaim.

## 6. Artefak dan reproduksi

Artefak pada [direktori bukti T02](m0/ep-01-t02/) meliputi:

- [pins.json](m0/ep-01-t02/pins.json): versi, integrity paket npm, checksum archive/source, commit, manifest image, dan batas bukti.
- [frontend-package.json](m0/ep-01-t02/frontend-package.json) dan [frontend-package-lock.json](m0/ep-01-t02/frontend-package-lock.json): seluruh resolusi direct/transitive yang dipakai.
- [backend-go.mod](m0/ep-01-t02/backend-go.mod) dan [backend-go.sum](m0/ep-01-t02/backend-go.sum): module graph dan checksum fixture Go.
- [fixture-sources.json](m0/ep-01-t02/fixture-sources.json): source minimal untuk mereproduksi pemeriksaan; bukan implementasi gateway atau dashboard.
- [checks.json](m0/ep-01-t02/checks.json) dan [SHA256SUMS](m0/ep-01-t02/SHA256SUMS): hasil serta checksum artefak bukti.

Jalankan dari root repository untuk memulihkan fixture ke direktori sementara:

```bash
TASK_CHECK_ROOT="$(mktemp -d /tmp/opencode-stack-check.XXXXXX)"
python3 - "$TASK_CHECK_ROOT" <<'PY'
from pathlib import Path
import json, shutil, sys
source = Path('docs/m0/ep-01-t02')
target = Path(sys.argv[1])
for relative, content in json.loads((source / 'fixture-sources.json').read_text()).items():
    destination = target / relative
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(content)
for original, relative in [
    ('frontend-package.json', 'frontend/package.json'),
    ('frontend-package-lock.json', 'frontend/package-lock.json'),
    ('backend-go.mod', 'backend/go.mod'),
    ('backend-go.sum', 'backend/go.sum'),
]:
    shutil.copyfile(source / original, target / relative)
PY
```

Gunakan Go `1.27.4`, Node `26.13.1`, dan npm `12.2.2` yang checksum-nya telah diverifikasi pada pins.json. Di direktori `frontend` fixture, jalankan `npm ci --ignore-scripts --strict-peer-deps --no-audit --no-fund`, `npm run check`, dan `npm run build`. Salin hasil `frontend/dist/` ke `backend/ui/`, lalu jalankan pemeriksaan modul, vet, tes, dan build Go dengan target `linux/amd64` serta `CGO_ENABLED=0`. Untuk kasus negatif, buat salinan manifest terpisah dengan TypeScript `7.0.2`; resolusi strict peer harus gagal `ERESOLVE`. Jangan mengubah lock yang telah lulus untuk menyembunyikan konflik.

## 7. Status acceptance dan pekerjaan tersisa

| AC T02 | Hasil teknis independen | Bukti |
|---|---|---|
| AC1 | Tersedia | Pemetaan A-1–A-8, pilihan, alternatif, dan konsekuensi pada bagian 1 |
| AC2 | Lulus pada fixture | Versi exact, sumber resmi, dukungan engine/peer, kompilasi, dan checker pada bagian 2/5 serta lock |
| AC3 | Tersedia untuk source/base/fixture | Commit MicroWARP/microsocks, empat digest base/builder/runtime, checksum/lock pada bagian 4/6; digest final image produk menunggu build pada task terkait |
| AC4 | Terbukti pada kasus dependency | TypeScript 7 ditolak dan 6 diterima tanpa override; YAML v3 ditinjau; Alpine EOL tidak dipilih |

Status tiket tetap `approved`. Dependensi T01 belum ditutup, terutama runtime provider aktif dan detail akses. Batas OpenCode Free untuk pemakaian dari dalam OpenCode serta risiko `forceStream` tetap mengikuti D-016 dan [hasil T01](M0-lingkungan-dan-integrasi.md); pemilihan stack tidak menyelesaikan kedua hal itu.

T03 harus membuktikan DNS melalui SOCKS5, TLS, fragmentasi/flushing SSE, cancellation, dan jumlah attempt. T04 membuktikan relay pada host target dengan otorisasi pengujian; T05 membuktikan packaging nyata. Pemilihan ini tidak memberi izin mengubah semantik request, melemahkan auth, menurunkan target, atau menyatakan M0 lulus.
