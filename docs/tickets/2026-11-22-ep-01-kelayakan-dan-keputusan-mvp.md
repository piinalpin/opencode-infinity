---
title: "Gateway layak dibangun pada lingkungan target"
date: 2026-11-22
version: 1
status: draft
planner: none
project: ""
hierarchy: epic
epic_id: EP-01
milestone: "M0"
priority: P0
capabilities: ["C-1", "C-2", "C-4", "C-5", "C-8", "C-9", "C-10", "C-14"]
---

# EP-01 Gateway layak dibangun pada lingkungan target

**Milestone:** M0 · **Prioritas:** P0 sesuai pembagian docs · **Status:** draft · **Pemilik/tanggal:** TBD

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
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC adalah **(agent's call)** berdasarkan docs dan tetap draft.
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan draft tidak diasumsikan. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-01-t01"></a>

## [TASK] EP-01-T01 Tetapkan lingkungan dan kebutuhan integrasi awal

**Status:** draft · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

O-1–O-3 perlu diubah menjadi input konkret agar spike menilai lingkungan yang benar.

Input terkonfirmasi melalui [D-001–D-003](../DECISIONS.md): penggunaan pribadi, gateway/pool WARP siap container di lokal atau homelab, serta akses homelab melalui Tailscale yang disiapkan pengguna. [D-006](../DECISIONS.md) menetapkan host uji amd64, Docker Compose, dan Tailscale di host; [D-007](../DECISIONS.md) menetapkan client utama OpenCode Desktop dan upstream Chat Completions. Provider/model, versi Desktop, listener final, dan kebutuhan affinity dicatat saat M0. Task ini tetap draft dan belum dinyatakan selesai.

[D-004](../DECISIONS.md) menetapkan pengguna sendiri sebagai pemilik keputusan produk dan penentu target keberhasilan. [D-005](../DECISIONS.md) menetapkan baseline dua tahap (catatan manual lalu pengukuran M0) dan pelaksana pengukurannya; angka baseline serta format pencatatan masih perlu ditetapkan.

- [PRD](../PRD.md): C-1, C-4, C-10.
- Kualitas terkait: kriteria keputusan dan reproduksibilitas pada ROADMAP.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- Keputusan yang ditangani task ini: O-1, O-2, O-3.

### Acceptance criteria

- **AC1:** **Given** host dan calon upstream/client belum ditetapkan, **When** pemilik produk bersama operator meninjau kebutuhan, **Then** matriks mencatat host, arsitektur CPU, akses jaringan, upstream/model, versi CLI/Desktop, dan runtime provider.
- **AC2:** **Given** model awal telah dipilih, **When** protokol dan kebutuhan affinity diperiksa, **Then** keputusan menjelaskan kesesuaian Chat Completions dan kebutuhan konsistensi jalurnya.
- **AC3:** **Given** data baseline belum tersedia, **When** pemilik pengukuran ditetapkan, **Then** setiap G-1–G-7 mempunyai sumber pengukuran serta baseline aktual atau TBD berpenanggung jawab.
- **AC4 (negatif/edge):** **Given** model awal membutuhkan Responses API atau jaminan yang belum terpenuhi, **When** hasil validasi ditinjau, **Then** ketidakcocokan dicatat untuk keputusan produk dan M0 tidak ditandai lulus diam-diam.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: Tidak ada; dapat dimulai sebagai task keputusan awal.
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-01-t02"></a>

## [TASK] EP-01-T02 Tetapkan stack dan versi yang kompatibel dengan arsitektur

**Status:** draft · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

Kandidat TECHSTACK belum menjadi pilihan final atau kumpulan versi yang terbukti kompatibel.

- [PRD](../PRD.md): C-4, C-8, C-10, C-14.
- Kualitas terkait: Q-2, Q-3, Q-10.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** A-1–A-8 dan lingkungan target tersedia, **When** kandidat teknologi dievaluasi, **Then** keputusan memetakan pilihan ke modul arsitektur dan mencatat konsekuensinya.
- **AC2:** **Given** pilihan toolchain dan paket diajukan, **When** rilis stabil serta dukungan silang diperiksa, **Then** matriks mencatat exact version, sumber resmi, dan hasil kompatibilitas termasuk compiler/checker frontend.
- **AC3:** **Given** komponen referensi dan base image dipilih, **When** identitas build dicatat, **Then** commit MicroWARP, digest image, dan checksum atau lock dependensi dapat ditelusuri.
- **AC4 (negatif/edge):** **Given** versi terbaru suatu paket tidak kompatibel atau kandidat YAML v3 sudah perlu ditinjau, **When** versi diputuskan, **Then** alasan pemilihan dicatat dan label latest tidak dianggap bukti kompatibilitas.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T01](#ep-01-t01)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-01-t03"></a>

## [TASK] EP-01-T03 Sediakan fixture yang membuktikan jalur dan attempt request

**Status:** draft · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

Pengujian deterministik perlu membedakan koneksi proxy, request yang terkirim, dan potongan stream.

- [PRD](../PRD.md): C-2, C-4, C-5.
- Kualitas terkait: Q-1, Q-2, Q-3.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-03, AC-05, AC-06, AC-07, AC-08. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** fixture upstream dan SOCKS5 dijalankan, **When** request kontrol melewati proxy, **Then** rekaman menunjukkan relay yang dipakai dan hostname tujuan yang diterima proxy.
- **AC2:** **Given** fixture streaming diatur untuk memecah byte, **When** skenario dijalankan ulang, **Then** content dan tool-call deltas dapat diverifikasi menurut urutan yang sama.
- **AC3:** **Given** fixture memutus koneksi pada fase berbeda, **When** skenario gagal dijalankan, **Then** bukti membedakan sebelum dispatch, pengiriman ambigu, dan stream yang sudah dimulai.
- **AC4:** **Given** client kontrol memutus stream, **When** fixture mengamati lifecycle request upstream, **Then** bukti dapat membedakan pembatalan request dari respons sukses dan mencatat jumlah dispatch.
- **AC5 (negatif/edge):** **Given** fixture proxy tidak tersedia, **When** request kontrol dijalankan, **Then** pengujian mendeteksi koneksi langsung jika terjadi dan gagal bila jalur itu digunakan.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T02](#ep-01-t02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-01-t04"></a>

## [TASK] EP-01-T04 Buktikan satu request melalui WARP pada host target

**Status:** draft · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

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

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T01](#ep-01-t01), [EP-01-T02](#ep-01-t02), [EP-01-T03](#ep-01-t03)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-01-t05"></a>

## [TASK] EP-01-T05 Buktikan UI dapat dikemas bersama aplikasi gateway

**Status:** draft · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

A-1/A-6 memerlukan satu artefak aplikasi yang dapat menyajikan UI tanpa service frontend produksi terpisah.

- [PRD](../PRD.md): C-10, C-14.
- Kualitas terkait: Q-10, Q-11.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** stack dan bentuk packaging dipilih, **When** aset UI contoh dibangun dan disajikan aplikasi, **Then** browser menerima halaman serta aset lokal dari listener admin.
- **AC2:** **Given** artefak runtime dibentuk, **When** isi dan proses runtime diperiksa, **Then** toolchain build atau browser pengujian tidak diperlukan untuk menyajikan UI.
- **AC3:** **Given** aplikasi dimulai pada host target, **When** smoke test kemasan dijalankan, **Then** hasil mencatat versi, arsitektur, konfigurasi akses, serta cara reproduksi.
- **AC4 (negatif/edge):** **Given** aset contoh belum memantau state pool, **When** hasil spike dicatat, **Then** bukti tidak dilabeli sebagai kelulusan dashboard AC-20–AC-22.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T02](#ep-01-t02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-01-t06"></a>

## [TASK] EP-01-T06 Tetapkan gerbang M0 dan rencana pengukuran MVP

**Status:** draft · **Parent:** EP-01 · **Milestone:** M0 · **Prioritas:** P0

### Konteks

Hasil spike perlu menjadi batas keputusan dan rencana pembuktian, bukan klaim MVP selesai.

- [PRD](../PRD.md): C-4, C-9, C-10.
- Kualitas terkait: Q-4, Q-5, Q-6, Q-10.
- [ROADMAP](../ROADMAP.md): M0; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- Keputusan yang ditangani task ini: O-6.

D-008 sudah menetapkan target Q-4/Q-5 provisional dan fixture benchmark 5 relay × 5 slot; task ini merinci anggaran buffer/state serta default timeout/probe yang belum ditetapkan.

### Acceptance criteria

- **AC1:** **Given** seluruh hasil spike tersedia, **When** gerbang M0 ditinjau, **Then** laporan menghubungkan host/client/model, versi, bukti WARP, dan kendala ke keputusan arsitektur.
- **AC2:** **Given** target Q-4–Q-6 sebagian sudah ditetapkan D-008 sebagai provisional, **When** rencana penerimaan ditetapkan, **Then** target, anggaran buffer/state, serta kapasitas fixture benchmark 5×5 dikonfirmasi dan default yang belum diputuskan dicatat penanggung jawabnya.
- **AC3:** **Given** AC-01–AC-22 perlu dibuktikan, **When** rencana pengujian dibakukan, **Then** setiap AC mempunyai metode, lingkungan, serta bukti yang harus disimpan pada milestone terkait.
- **AC4 (negatif/edge):** **Given** spike gagal atau membutuhkan perubahan scope, **When** M0 dinilai, **Then** milestone tetap belum lulus sampai keputusan produk dan dokumen terkait diperbarui.

### Batas dan dependensi

- Di luar scope: Gateway produksi lengkap, dashboard operasional lengkap, dan implementasi kemampuan P1.
- Diblokir oleh: [EP-01-T04](#ep-01-t04), [EP-01-T05](#ep-01-t05)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.
