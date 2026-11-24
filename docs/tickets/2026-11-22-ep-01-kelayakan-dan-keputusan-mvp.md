---
title: "Gateway layak dibangun pada lingkungan target"
date: 2026-11-22
version: 6
updated_at: 2026-11-24
status: approved
approved_by: pengguna
approved_at: 2026-11-23
planner: none
project: ""
hierarchy: epic
epic_id: EP-01
milestone: "M0"
priority: P0
capabilities: ["C-1", "C-2", "C-4", "C-5", "C-8", "C-9", "C-10", "C-14"]
---

# EP-01 Gateway layak dibangun pada lingkungan target

**Milestone:** M0 · **Prioritas:** P0 sesuai pembagian docs · **Status:** approved · **Pemilik:** pengguna · **Tanggal:** TBD

## Ringkasan

Tim memiliki keputusan arsitektur dan stack, fixture, serta bukti satu jalur WARP sebelum membangun gateway lengkap.

## Masalah

Host, upstream/client, kontrak transport, dan versi kandidat belum dipastikan sebagai satu deployment yang dapat direproduksi.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-1, C-2, C-4, C-5, C-8, C-9, C-10, C-14.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone M0.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), AC-03, AC-05, AC-06, AC-07, AC-08.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- Gerbang M0 pada ROADMAP terpenuhi dengan bukti yang dapat dijalankan ulang.
- Pilihan host/client/model, exact version, dan sumber artefak tercatat tanpa secret.
- Baseline yang belum terukur tetap TBD; metode dan pelaksana pengukuran mengikuti D-005 (angka aktual menyusul M0).

## Scope

- Keputusan lingkungan dan baseline
- Validasi arsitektur serta kompatibilitas versi
- Fixture dan spike WARP/UI
- Gerbang M0 dan rencana bukti penerimaan

## Di luar scope

Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

✅ berarti AC memiliki bukti pada lingkup yang disebutkan di hasil pengerjaan (dokumentasi, validasi stack independen, atau fixture lokal). Penanda ini tidak menyatakan dependensi task, integrasi nyata, atau gerbang M0 selesai.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-01-T01](#ep-01-t01) | Tetapkan lingkungan dan kebutuhan integrasi awal | Task | 4 | Tidak ada |
| [EP-01-T02](#ep-01-t02) | Tetapkan stack dan versi yang kompatibel dengan arsitektur | Task | 4 | [EP-01-T01](#ep-01-t01) |
| [EP-01-T03](#ep-01-t03) | Sediakan fixture yang membuktikan jalur dan attempt request | Task | 5 | [EP-01-T02](#ep-01-t02) |
| [EP-01-T04](#ep-01-t04) | Buktikan satu request melalui WARP pada host target | Task | 4 | [EP-01-T01](#ep-01-t01), [EP-01-T02](#ep-01-t02), [EP-01-T03](#ep-01-t03) |
| [EP-01-T05](#ep-01-t05) | Buktikan UI dapat dikemas bersama aplikasi gateway | Task | 4 | [EP-01-T02](#ep-01-t02) |
| [EP-01-T06](#ep-01-t06) | Tetapkan gerbang M0 dan rencana pengukuran MVP | Task | 4 | [EP-01-T04](#ep-01-t04), [EP-01-T05](#ep-01-t05) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: tidak ada. Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Model awal mungkin memerlukan protokol di luar MVP; putuskan O-3 sebelum menutup M0.
- Pilihan library dapat mengulang request atau menyelesaikan DNS lokal; buktikan dengan fixture.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC berasal dari usulan agent dan telah disetujui pengguna pada 23 November 2026 ([D-015](../DECISIONS.md#d-015-persetujuan-seluruh-tiket-mvp-dan-p1)).
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan mencakup epic dan seluruh child story/task pada file ini. Status `approved` menyatakan persetujuan tiket; implementasi, pemenuhan dependensi, dan kelulusan AC tetap memerlukan bukti. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-01-t01"></a>

## [TASK] EP-01-T01 Tetapkan lingkungan dan kebutuhan integrasi awal

**Status:** approved · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

O-1–O-3 perlu diubah menjadi input konkret agar spike menilai lingkungan yang benar.

Input terkonfirmasi melalui [D-001–D-003](../DECISIONS.md): penggunaan pribadi, gateway/pool WARP siap container di lokal atau homelab, serta akses homelab melalui Tailscale yang disiapkan pengguna. [D-006](../DECISIONS.md) menetapkan host uji amd64, Docker Compose, dan Tailscale di host; [D-007](../DECISIONS.md) menetapkan client utama OpenCode Desktop dan upstream Chat Completions. D-016 memilih OpenCode Free dan `deepseek-v4.1-flash-free`; versi Desktop/CLI bawaan `2.3.0` tercatat pada hasil M0. Listener final, exact version SDK aktif, dan bukti kelayakan gateway masih terbuka. Task ini telah disetujui melalui D-015 dan belum dinyatakan selesai.

[D-004](../DECISIONS.md) menetapkan pengguna sendiri sebagai pemilik keputusan produk dan penentu target keberhasilan. [D-005](../DECISIONS.md) menetapkan baseline dua tahap (catatan manual lalu pengukuran M0) dan pelaksana pengukurannya; angka baseline serta format pencatatan masih perlu ditetapkan.

- [PRD](../PRD.md): C-1, C-4, C-10.
- Kualitas terkait: kriteria keputusan dan reproduksibilitas pada ROADMAP.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- Keputusan yang ditangani task ini: O-1, O-2, O-3.

### Acceptance criteria

- **AC1:** **Given** host dan calon upstream/client belum ditetapkan, **When** pemilik produk bersama operator meninjau kebutuhan, **Then** matriks mencatat host, arsitektur CPU, akses jaringan, upstream/model, versi CLI/Desktop, dan runtime provider.
- **AC2:** **Given** model awal telah dipilih, **When** protokol dan kebutuhan affinity diperiksa, **Then** keputusan menjelaskan kesesuaian Chat Completions dan kebutuhan konsistensi jalurnya.
- ✅ **AC3:** **Given** data baseline belum tersedia, **When** pemilik pengukuran ditetapkan, **Then** setiap G-1–G-7 mempunyai sumber pengukuran serta baseline aktual atau TBD berpenanggung jawab.
- ✅ **AC4 (negatif/edge):** **Given** model awal membutuhkan Responses API atau jaminan yang belum terpenuhi, **When** hasil validasi ditinjau, **Then** ketidakcocokan dicatat untuk keputusan produk dan M0 tidak ditandai lulus diam-diam.

### Hasil pengerjaan

Pemeriksaan 23 November 2026 dan matriks input tersedia pada [M0 — Lingkungan dan kebutuhan integrasi awal](../M0-lingkungan-dan-integrasi.md). Rencana pengerjaan disetujui pengguna; status task tetap `approved`, dan pekerjaan belum dinyatakan selesai.

| AC | Hasil saat ini | Bukti/tindak lanjut |
|---|---|---|
| AC1 | Sebagian | Host/client, OpenCode Free, model `deepseek-v4.1-flash-free`, dan transport D-016 tercatat; exact version SDK aktif serta detail akses final belum lengkap |
| AC2 | Penilaian dokumentasi tersedia | Endpoint Chat Completions terkonfirmasi pada katalog resmi; batas akses Free, kebutuhan pembuktian sesi, dan JSON/non-stream dijelaskan tanpa klaim gateway kompatibel |
| ✅ AC3 | Terpenuhi pada tingkat dokumentasi | G-1–G-7 memiliki sumber manual/fixture, pemilik pengukuran, dan baseline TBD; pengukuran aktual mengikuti D-005 |
| ✅ AC4 | Terpenuhi pada tingkat dokumentasi | Pembatasan Free untuk penggunaan dalam OpenCode dan risiko `forceStream` tercatat sebagai hambatan M0; tidak ada hasil generasi nyata atau kelulusan gateway yang diklaim |

Pilihan provider/model dan input transport sudah dikonfirmasi pengguna melalui D-016. Detail yang belum terbukti tidak diasumsikan terjawab oleh persetujuan rencana. Tidak ada generasi berbayar atau perubahan konfigurasi/lifecycle deployment pada pekerjaan ini.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: Tidak ada; dapat dimulai sebagai task keputusan awal.
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-01-t02"></a>

## [TASK] EP-01-T02 Tetapkan stack dan versi yang kompatibel dengan arsitektur

**Status:** approved · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

Kandidat awal TECHSTACK dievaluasi melalui [laporan stack dan versi M0](../M0-stack-dan-versi.md). Pengguna mengizinkan bagian validasi stack yang independen dilanjutkan dengan blocker T01 tetap tercatat (D-017); dependensi di bawah tidak dihapus.

- [PRD](../PRD.md): C-4, C-8, C-10, C-14.
- Kualitas terkait: Q-2, Q-3, Q-10.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- ✅ **AC1:** **Given** A-1–A-8 dan lingkungan target tersedia, **When** kandidat teknologi dievaluasi, **Then** keputusan memetakan pilihan ke modul arsitektur dan mencatat konsekuensinya.
- ✅ **AC2:** **Given** pilihan toolchain dan paket diajukan, **When** rilis stabil serta dukungan silang diperiksa, **Then** matriks mencatat exact version, sumber resmi, dan hasil kompatibilitas termasuk compiler/checker frontend.
- ✅ **AC3:** **Given** komponen referensi dan base image dipilih, **When** identitas build dicatat, **Then** commit MicroWARP, digest image, dan checksum atau lock dependensi dapat ditelusuri.
- ✅ **AC4 (negatif/edge):** **Given** versi terbaru suatu paket tidak kompatibel atau kandidat YAML v3 sudah perlu ditinjau, **When** versi diputuskan, **Then** alasan pemilihan dicatat dan label latest tidak dianggap bukti kompatibilitas.

### Hasil pengerjaan

Pemeriksaan 23 November 2026 tercatat pada [M0 — Stack, versi, dan bukti kompatibilitas](../M0-stack-dan-versi.md), termasuk [pin dan lock fixture](../m0/ep-01-t02/). Status tetap `approved`; hasil teknis independen tidak menyatakan dependensi T01 atau gerbang M0 selesai.

| AC | Hasil teknis independen | Bukti/batas |
|---|---|---|
| ✅ AC1 | Tersedia | Go/Svelte dipetakan ke A-1–A-8; batas modul, state, dua listener, dan operator relay tetap |
| ✅ AC2 | Lulus fixture | Exact version dan sumber tercatat; strict peer, compiler/checker frontend, verifikasi modul, vet, tes, serta binary statis amd64 lulus |
| ✅ AC3 | Source/base/fixture terlacak | Commit MicroWARP/microsocks, lisensi, empat digest manifest image, lock/checksum tersedia; image produk/relay final belum dibangun |
| ✅ AC4 | Terbukti pada kasus dependency | TypeScript 7 gagal ERESOLVE; 6.0.3 lulus; YAML v3 ditinjau dan v4 dipilih; Alpine 3.21 EOL tidak dipilih |

Seluruh eksperimen memakai data sintetis di direktori sementara. Tidak ada generasi upstream, bootstrap identitas WARP, atau perubahan container/volume aktif. T03/T04/T05 tetap membuktikan transport, deployment, dan packaging yang tidak dibuktikan oleh kompilasi.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T01](#ep-01-t01)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-01-t03"></a>

## [TASK] EP-01-T03 Sediakan fixture yang membuktikan jalur dan attempt request

**Status:** approved · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

Pengujian deterministik perlu membedakan koneksi proxy, request yang terkirim, dan potongan stream.

- [PRD](../PRD.md): C-2, C-4, C-5.
- Kualitas terkait: Q-1, Q-2, Q-3.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-03, AC-05, AC-06, AC-07, AC-08. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- ✅ **AC1:** **Given** fixture upstream dan SOCKS5 dijalankan, **When** request kontrol melewati proxy, **Then** rekaman menunjukkan relay yang dipakai dan hostname tujuan yang diterima proxy.
- ✅ **AC2:** **Given** fixture streaming diatur untuk memecah byte, **When** skenario dijalankan ulang, **Then** content dan tool-call deltas dapat diverifikasi menurut urutan yang sama.
- ✅ **AC3:** **Given** fixture memutus koneksi pada fase berbeda, **When** skenario gagal dijalankan, **Then** bukti membedakan sebelum dispatch, pengiriman ambigu, dan stream yang sudah dimulai.
- ✅ **AC4:** **Given** client kontrol memutus stream, **When** fixture mengamati lifecycle request upstream, **Then** bukti dapat membedakan pembatalan request dari respons sukses dan mencatat jumlah dispatch.
- ✅ **AC5 (negatif/edge):** **Given** fixture proxy tidak tersedia, **When** request kontrol dijalankan, **Then** pengujian mendeteksi koneksi langsung jika terjadi dan gagal bila jalur itu digunakan.

### Hasil pengerjaan — 23 November 2026

- [Fixture jalur dan attempt request](../M0-fixture-transport.md) memetakan bukti AC1–AC5 ke helper Go reusable serta tes HTTP/TLS/SOCKS5 loopback nyata. Pengguna menyetujui pengerjaan fixture secara independen dengan keterbatasan dependensi tetap dicatat.
- Bukti mencakup hostname/relay/peer, penghitung koneksi terpisah dari dispatch, fragmentasi/flushing SSE, refusal sebelum tunnel, POST ambigu pada koneksi baru/reusable, stream putus, serta cancellation sebelum handler selesai.
- Negative control membuktikan upstream dapat dicapai langsung dan oracle menolak bypass. TLS hostname, allowlist, batas body/frame/rekaman/koneksi, serta cleanup konkuren ikut diuji.
- Distribusi resmi Go 1.27.4: 18 tes utama lulus; format/vet lulus; 50 pengulangan dengan race detector menghasilkan 900 eksekusi tes utama, nol kegagalan atau laporan data race.
- Hasil ini membuktikan fixture sintetis, bukan jalur WARP/provider, retry SDK/client nyata, performa produksi, dependensi T01/T02, atau gerbang M0. Status persetujuan tiket tetap `approved`.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T02](#ep-01-t02)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-01-t04"></a>

## [TASK] EP-01-T04 Buktikan satu request melalui WARP pada host target

**Status:** approved · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

Lulus fixture saja tidak membuktikan WARP, izin jaringan, dan upstream nyata berfungsi bersama.

- [PRD](../PRD.md): C-2, C-4, C-10.
- Kualitas terkait: Q-1, Q-2, Q-3.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-05, AC-06. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** relay dan credential uji operator tersedia, **When** request nyata dikirim melalui SOCKS5, **Then** upstream memberi respons melalui relay yang dicatat pada konfigurasi uji.
- **AC2:** **Given** hostname upstream dikirim melalui proxy, **When** jalur TLS dan DNS diuji, **Then** bukti menunjukkan resolusi tujuan lewat SOCKS5 dan verifikasi hostname TLS aktif.
- **AC3:** **Given** upstream mengirim stream, **When** client menerima respons, **Then** potongan respons diteruskan sebelum keseluruhan respons selesai.
- **AC4 (negatif/edge):** **Given** relay gagal atau bukti belum terkirim tidak tersedia, **When** skenario gangguan dijalankan, **Then** tidak ada direct fallback dan attempt generasi dibatasi satu pada kondisi ambigu.

### Hasil pengerjaan — 23 November 2026

[Laporan T04](../M0-warp-request.md), [provenance](../m0/ep-01-t04/provenance.json), dan [checks](../m0/ep-01-t04/checks.json) mencatat hasil parsial. Image relay fixture terpin berhasil dibangun, kontrol transport dan bootstrap sintetis diuji. Native smoke mencapai penerus lokal tetapi ditolak dengan `tools_disabled`; nol request diteruskan, nol registrasi WARP, dan nol generasi provider. Langkah native berikutnya menunggu persetujuan revisi setelah tiga upaya GREEN belum lulus.

| AC | Hasil | Batas |
|---|---|---|
| AC1 | Belum terbukti | Belum ada respons generasi aktual melalui WARP |
| AC2 | Fixture lokal saja | Domain SOCKS5/TLS diuji secara lokal; request WARP yang sama belum tersedia |
| AC3 | Fixture lokal saja | Barrier dan fragmentasi lokal lulus; incremental stream provider belum terbukti |
| AC4 | Fixture lokal saja | Fault ambigu/dead relay lokal dan latch diuji; fault relay WARP aktual belum dijalankan |

Tidak ada ✅ pada T04, tidak ada pelonggaran wording AC/dependensi, dan status tetap `approved`. Batas otorisasi baru pada D-018 tidak mengizinkan pemakaian profil existing, retry generasi, atau perubahan model.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T01](#ep-01-t01), [EP-01-T02](#ep-01-t02), [EP-01-T03](#ep-01-t03)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-01-t05"></a>

## [TASK] EP-01-T05 Buktikan UI dapat dikemas bersama aplikasi gateway

**Status:** approved · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

A-1/A-6 memerlukan satu artefak aplikasi yang dapat menyajikan UI tanpa service frontend produksi terpisah.

- [PRD](../PRD.md): C-10, C-14.
- Kualitas terkait: Q-10, Q-11.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- ✅ **AC1:** **Given** stack dan bentuk packaging dipilih, **When** aset UI contoh dibangun dan disajikan aplikasi, **Then** browser menerima halaman serta aset lokal dari listener admin.
- ✅ **AC2:** **Given** artefak runtime dibentuk, **When** isi dan proses runtime diperiksa, **Then** toolchain build atau browser pengujian tidak diperlukan untuk menyajikan UI.
- ✅ **AC3:** **Given** aplikasi dimulai pada host target, **When** smoke test kemasan dijalankan, **Then** hasil mencatat versi, arsitektur, konfigurasi akses, serta cara reproduksi.
- ✅ **AC4 (negatif/edge):** **Given** aset contoh belum memantau state pool, **When** hasil spike dicatat, **Then** bukti tidak dilabeli sebagai kelulusan dashboard AC-20–AC-22.

### Hasil pengerjaan — 24 November 2026

[Laporan packaging T05](../M0-ui-packaging.md), [checks](../m0/ep-01-t05/checks.json), dan [provenance](../m0/ep-01-t05/provenance.json) membuktikan AC1–AC4 secara independen menggunakan stack T02. Source fixture berada di [tests/spike/ui-packaging](../../tests/spike/ui-packaging/), dengan nested module Go terpisah dari tooling aplikasi produk.

- UI Svelte hasil build ditanam dalam binary Go. Browser pada image final memverifikasi mount/interaksi lokal, CSS, dan aset same-origin melalui listener admin; seluruh halaman/aset memerlukan credential admin sintetis tersendiri.
- Image Distroless `linux/amd64` diperiksa melalui ekspor filesystem dan daftar proses: satu proses Go statis, tanpa toolchain/browser/runtime frontend atau mount source/dist host. Versi, image ID, checksum, batas akses, dan reproduksi tersedia pada laporan.
- Tujuh tes utama backend fixture, dua tes browser, empat build negatif aset, race/vet/format, dan regresi cleanup lulus. Pelepasan slot koneksi diuji 20 kali dengan race detector; review strict menyetujui setelah perbaikan satu temuan cleanup.
- Revisi dari jaringan internal ke bridge khusus fixture disetujui pengguna setelah pemetaan port host tidak tersedia. Publish tetap loopback; resource fixture serta credential sementara sudah dibersihkan. Deployment aktif, Tailscale, WARP, dan provider tidak diubah.

Status persetujuan tetap `approved`; wording AC/dependensi dipertahankan. Hasil ini tidak meluluskan dashboard AC-20–AC-22, kontrak sesi produksi, integrasi client/upstream/WARP, dependensi formal T01/T02, benchmark, atau seluruh gerbang M0.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T02](#ep-01-t02)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-01-t06"></a>

## [TASK] EP-01-T06 Tetapkan gerbang M0 dan rencana pengukuran MVP

**Status:** approved · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

Hasil spike perlu menjadi batas keputusan dan rencana pembuktian, bukan klaim MVP selesai.

- [PRD](../PRD.md): C-4, C-9, C-10.
- Kualitas terkait: Q-4, Q-5, Q-6, Q-10.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- Keputusan yang ditangani task ini: O-6.

D-008 sudah menetapkan target Q-4/Q-5 provisional dan fixture benchmark 5 relay × 5 slot; task ini merinci anggaran buffer/state serta default timeout/probe yang belum ditetapkan.

### Acceptance criteria

- **AC1:** **Given** seluruh hasil spike tersedia, **When** gerbang M0 ditinjau, **Then** laporan menghubungkan host/client/model, versi, bukti WARP, dan kendala ke keputusan arsitektur.
- ✅ **AC2:** **Given** target Q-4–Q-6 sebagian sudah ditetapkan D-008 sebagai provisional, **When** rencana penerimaan ditetapkan, **Then** target, anggaran buffer/state, serta kapasitas fixture benchmark 5×5 dikonfirmasi dan default yang belum diputuskan dicatat penanggung jawabnya.
- ✅ **AC3:** **Given** AC-01–AC-22 perlu dibuktikan, **When** rencana pengujian dibakukan, **Then** setiap AC mempunyai metode, lingkungan, serta bukti yang harus disimpan pada milestone terkait.
- ✅ **AC4 (negatif/edge):** **Given** spike gagal atau membutuhkan perubahan scope, **When** M0 dinilai, **Then** milestone tetap belum lulus sampai keputusan produk dan dokumen terkait diperbarui.

### Hasil pengerjaan — 24 November 2026

[Gerbang dan rencana pengukuran MVP](../M0-gerbang-dan-pengukuran.md) merangkum T01–T05, target, anggaran resource, metode G/Q, 22 AC sumber beserta lingkungan/milestone/artefak, serta pemilik input yang belum final. Pengguna menyetujui dokumentasi independen dan profil/metode provisional melalui [D-019](../DECISIONS.md#d-019-profil-resource-dan-metode-pengukuran-provisional).

| AC | Hasil saat ini | Batas |
|---|---|---|
| AC1 | Parsial | Rekap tersedia, tetapi hasil WARP T04 dan input T01 belum lengkap; tidak diberi ✅ |
| ✅ AC2 | Terpenuhi pada tingkat rencana dokumentasi | Buffer 64/8/16/8 MiB, kapasitas 5×5/admission 20, metode dan daftar TBD dikonfirmasi; belum bukti RSS atau default produk final |
| ✅ AC3 | Terpenuhi pada tingkat dokumentasi | Seluruh AC-01–AC-22 memiliki metode, lingkungan, milestone, artefak dan task pemilik; belum hasil acceptance produk |
| ✅ AC4 | Terpenuhi pada tingkat dokumentasi | M0 tetap belum lulus; target, prasyarat dan batas otorisasi tidak dilonggarkan |

Task T06 tetap parsial dan status persetujuan `approved` dipertahankan. Baseline aktual G-1–G-7, SDK/akses/listener T01, dan bukti WARP T04 tetap menghalangi penutupan M0/M1 sesuai sumbernya. Tidak ada benchmark, generasi, perubahan fixture/runtime, atau lifecycle deployment yang dijalankan. Laporan/indeks dan keputusan diperiksa sebagai dokumen; hasil T05 serta seluruh bukti sebelumnya dipertahankan. Setelah bukti task pemilik tersedia, AC1 dan gerbang M0 harus ditinjau ulang.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T04](#ep-01-t04), [EP-01-T05](#ep-01-t05)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.
