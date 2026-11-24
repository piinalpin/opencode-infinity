# M0 — Bukti kemasan UI bersama aplikasi gateway

Tanggal pemeriksaan: 24 November 2026  
Task: [EP-01-T05](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t05)  
Status: AC1–AC4 terbukti pada fixture packaging independen; belum menjadi kelulusan M0.

## Lingkup dan bentuk artefak

Spike ini memakai [stack dan pin T02](M0-stack-dan-versi.md). Source berada pada [fixture T05](../tests/spike/ui-packaging/), terpisah dari aplikasi produk yang belum dibangun. Nested module `backend/` memakai Go standard library; dependency YAML/Prometheus fixture T02 tidak dibawa. Frontend memakai manifest dan lock T02 persis, bukan dependency aplikasi produk baru.

Pipeline menghasilkan `web/dist/`, menyalinnya ke `backend/internal/webui/dist/`, lalu menanam HTML/JS/CSS ke binary lewat `go:embed`. Pola embed mewajibkan ketiga jenis aset tersedia ketika kompilasi. File handler hanya menyajikan halaman utama dan aset `.js`/`.css` hasil build; direktori, source, sourcemap, serta aset hilang tidak mendapat fallback HTML.

Builder UI menggunakan Node 26.13.1 dan npm 12.2.2; Svelte 5.59.1, Vite 8.3.8, TypeScript 6.0.3, serta Playwright 1.65.1 mengikuti lock T02. Builder backend memakai Go resmi 1.27.4 dan menghasilkan binary statis `linux/amd64`. Image runtime Distroless nonroot membawa binary dengan aset tertanam. Browser pengujian berjalan di host, bukan runtime atau service frontend tambahan.

## Akses dan batas fixture

Compose hanya membuat satu service aplikasi pada bridge khusus fixture, dengan publish host `127.0.0.1:<port-sementara>` ke container `0.0.0.0:8788`. Port dialokasikan Docker setiap run dan dicatat pada bukti; port/listener produk serta tailnet final tetap TBD. Tailscale host tidak diubah. Revisi bridge ini disetujui pengguna setelah smoke menunjukkan jaringan `internal: true` tidak mendapat port host pada daemon uji. Bridge tidak mengklaim blokir egress; aplikasi fixture tidak memiliki jalur upstream.

Seluruh HTML/aset memakai Basic auth stateless khusus fixture, dengan username `admin` dan password acak sementara. Password dibaca dari file runtime, tidak ditanam dalam image/bundle atau ditaruh dalam URL, argv, localStorage, maupun laporan. Token gateway sintetis berbeda digunakan sebagai kontrol penolakan. HTTP pada smoke ini memakai host loopback dan bridge khusus fixture. Ini bukan kontrak sesi browser produksi; TTL/logout/perlindungan sesi tetap EP-05-T01.

Direktori staging host berizin `0700`. File password sintetis `0444` dipasang read-only agar UID runtime `65532` dapat membacanya; direktori parent tidak dibagikan ke container. File dihapus saat cleanup. Container memakai user `65532:65532`, root filesystem read-only, tanpa capability atau Docker socket/NET_ADMIN, batas 1 CPU/128 MiB/64 PID, dan delapan koneksi HTTP aktif maksimum. Batas waktu HTTP untuk spike statis berbeda dari kontrak drain/stream gateway produk.

## Reproduksi

Prasyarat: host Linux amd64 dengan Docker/Compose yang dapat dipakai, Node 26.13.1, npm 12.2.2, Go 1.27.4, Python 3, serta dependency sistem Chromium. Build/install membutuhkan akses registry/CDN. Builder serta runtime dipin digest T02; browser diunduh sesuai revision Playwright bila belum tersedia.

Dari root repository, pilih direktori sementara **yang belum ada**:

```bash
python3 -B tests/spike/ui-packaging/run.py --work-dir /tmp/opencode-ui-t05-ulang
```

Runner memulihkan manifest/lock T02 ke staging bersih, menjalankan pemeriksaan tipe/build dan tes backend, membuktikan build negatif, membangun image terpin, menjalankan Compose dengan project unik, menguji browser terhadap image tersebut, memeriksa filesystem/proses runtime, lalu membersihkan container/network/tag image milik run. Source, dependency directory, browser, cache, binary, dan ekspor filesystem tetap di direktori sementara. Runner tidak memakai dist atau mount source dari host ketika runtime diuji.

Hasil tersedia pada `<work-dir>/checks.json`. Log mentah hanya berada di staging privat; artefak permanen berupa ringkasan teredaksi dan provenance/checksum. Jangan menjalankan `go test ./...` langsung dalam nested backend sebelum menyiapkan dist. Untuk pengujian terfokus setelah runner mempersiapkan staging:

```bash
cd /tmp/opencode-ui-t05-ulang/context/backend
go test -count=1 ./...
go test -race -count=20 -run '^TestConnectionBoundAndRelease$' ./...
go vet ./...
```

Root `go test ./...` tidak mencakup nested module T05. Validasi root dan fixture dilakukan secara eksplisit; runner tidak mengeksekusi native client, WARP, provider, atau generasi.

## Hasil dan acceptance criteria

Smoke final pada Docker host `linux/amd64` (Docker 29.9.3, Compose 5.7.1) lulus. Browser Chromium `159.0.8154.6`, revision `1255`, dipakai bersama Playwright `1.65.1`. Listener uji final adalah `http://127.0.0.1:32769`, hanya selama fixture berjalan. Container, network, tag image fixture, dan file credential sudah dibersihkan; alamat tersebut bukan layanan yang tetap aktif.

Image lokal yang benar-benar diuji memiliki ID `sha256:cf3c39da982c282be253fbc5d20a8903858ef22905fc5b7786f2516fc8dba2c1`. Ini ID image lokal, bukan digest publikasi registry. Binary hasil ekspor runtime memiliki SHA-256 `f92a437db655c8a14191497295bf7c00fc682e7bd79e1bc0de09120ecf3f53b5`; `file` membuktikan ELF x86-64 statis, dan `go version -m` mencatat Go `1.27.4`, `CGO_ENABLED=0`, serta target `linux/amd64`.

| AC T05 | Hasil | Bukti |
|---|---|---|
| ✅ AC1 | Lulus pada fixture | Browser menampilkan komponen Svelte, interaksi lokal berfungsi, CSS diterapkan, JS/CSS berasal dari origin admin yang sama; byte HTTP cocok dengan hasil build |
| ✅ AC2 | Lulus pada image final | Ekspor 1.405 entri filesystem diperiksa; toolchain/Node/npm/Go/browser/shell tidak ditemukan; `docker top` menunjukkan satu proses `gateway`; hanya file credential dipasang, tanpa source/dist host |
| ✅ AC3 | Lulus pada host target | Versi, target amd64, pin builder/runtime, image ID, checksum aset/binary, akses loopback, batas container, dan reproduksi dicatat |
| ✅ AC4 | Batas klaim dipertahankan | Label terlihat menyatakan data sintetis; tidak ada state pool, probe, polling, usage, atau klaim AC-20–AC-22 lulus |

[Checks](m0/ep-01-t05/checks.json), [provenance](m0/ep-01-t05/provenance.json), dan [checksum](m0/ep-01-t05/SHA256SUMS) menyimpan ringkasan teredaksi. Provenance memuat hash seluruh source fixture dan manifest/lock T02. Generated assets, dependency directory, binary, browser, ekspor filesystem, serta log mentah tidak ditambahkan ke repository.

| Pemeriksaan | Hasil |
|---|---|
| Frontend strict peer, checker, TypeScript, build | Lulus; 0 error/0 warning |
| Backend fixture | 7 tes utama lulus; race dan vet lulus; format bersih; builder Go resmi juga menjalankan tes/race/vet |
| Release slot saat Close bersamaan | 20 pengulangan dengan race detector lulus; nol kegagalan/data race |
| Build negatif | Tanpa seluruh dist, index, JS, atau CSS: masing-masing gagal karena pola embed tidak menemukan aset |
| Auth runtime | Credential hilang/salah/malformed/gateway ditolak untuk halaman dan seluruh aset; credential admin valid mendapat aset |
| Browser image final | 2 tes lulus: mount/interaksi/aset lokal dan penolakan browser tanpa credential |
| Root repository | 27 tes utama lulus, 1 native opt-in dilewati; race/vet/format lulus; 2 tes Python existing lulus |
| Regresi runner | 1 tes lulus: kegagalan cleanup tetap menghapus credential dan mencatat status gagal |
| Review strict | APPROVE; 1 temuan MEDIUM cleanup diperbaiki dan ditinjau ulang; tidak ada temuan terbuka |

Tiga smoke awal belum lulus karena konfigurasi kompresi log, port jaringan internal yang tidak dipasang, dan format `docker top` tanpa kolom PID. Perbaikan tidak melemahkan assertion; revisi bridge disetujui pengguna. Smoke final memeriksa image yang sama dengan browser dan inventaris runtime. Build cache Docker digunakan; bukti ini tidak mengklaim reproduksi bit-identik dari builder remote kosong.

## Batas pembuktian

Kelulusan packaging tidak menyatakan kelulusan dashboard AC-20–AC-22, autentikasi/sesi produksi, integrasi client/provider/WARP, benchmark performa, T01, atau gerbang M0. Dependensi formal T05 terhadap T02 dan T02 terhadap T01 tetap dicatat. Hasil ini merupakan pembuktian teknis independen menggunakan pilihan stack T02; status persetujuan tiket tetap `approved`.

## Sumber teknis

- [Go embed](https://pkg.go.dev/embed): aset dibaca saat kompilasi, relatif terhadap package.
- [Vite static deployment](https://vite.dev/guide/static-deploy.html): build aset statis; `vite preview` bukan server produksi. Dokumentasi diambil lewat Context7 `/vitejs/vite`; versi 8.3.8 belum memiliki indeks khusus, sehingga kompatibilitas exact version dibuktikan dengan build.
- [Docker multi-stage](https://docs.docker.com/build/building/multi-stage/): hanya artefak akhir disalin ke runtime.
- [Playwright HTTP credentials](https://playwright.dev/docs/test-use-options#http-credentials): konteks browser khusus fixture; trace/HAR/video dinonaktifkan.
- [Docker local logging](https://docs.docker.com/engine/logging/drivers/local/): log dibatasi 1 MiB/satu file dengan kompresi dinonaktifkan.
