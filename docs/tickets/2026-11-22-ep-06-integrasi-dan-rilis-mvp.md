---
title: "MVP dapat dipakai dan dioperasikan pada deployment yang teruji"
date: 2026-11-22
version: 1
status: draft
planner: none
project: ""
hierarchy: epic
epic_id: EP-06
milestone: "M3"
priority: P0
capabilities: ["C-1", "C-2", "C-3", "C-4", "C-5", "C-6", "C-7", "C-8", "C-9", "C-10", "C-14"]
---

# EP-06 MVP dapat dipakai dan dioperasikan pada deployment yang teruji

**Milestone:** M3 · **Prioritas:** P0 sesuai pembagian docs · **Status:** draft · **Pemilik/tanggal:** TBD

## Ringkasan

CLI/Desktop, dashboard, transport WARP, benchmark, dan prosedur operasional lulus pada artefak rilis yang sama.

## Masalah

Fixture dan modul yang lulus sendiri belum membuktikan OpenCode nyata, resource total, serta paket instalasi sebagai satu produk.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-1, C-2, C-3, C-4, C-5, C-6, C-7, C-8, C-9, C-10, C-14.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone M3.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), AC-01, AC-02, AC-03, AC-04, AC-05, AC-06, AC-07, AC-08, AC-09, AC-10, AC-11, AC-12, AC-13, AC-14, AC-15, AC-16, AC-17, AC-18, AC-19, AC-20, AC-21, AC-22.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- AC-01–AC-22 lulus pada artefak rilis yang dicatat versinya.
- G-1–G-7 dan Q-1–Q-11 mempunyai bukti; target resource dilaporkan dengan skenario yang ditetapkan.
- Baseline yang belum terukur tetap TBD; pemilik pengukuran ditetapkan melalui O-1.

## Scope

- Integrasi CLI dan Desktop nyata
- Benchmark serta pemeriksaan versi rilis
- Paket deployment dan panduan operasi
- Penerimaan semua AC/G/Q dan keputusan rilis

## Di luar scope

Implementasi fitur P1, klaim semua versi client kompatibel, dan SLA yang belum ditetapkan.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-06-S01](#ep-06-s01) | Pakai gateway dari OpenCode CLI yang diuji | Story | 4 | [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06) |
| [EP-06-S02](#ep-06-s02) | Pakai gateway dari OpenCode Desktop yang diuji | Story | 4 | [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06) |
| [EP-06-T03](#ep-06-t03) | Buktikan target performa dengan dashboard aktif | Task | 4 | [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06), [EP-03-S03](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s03), [EP-01-T06](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06) |
| [EP-06-T04](#ep-06-t04) | Sediakan paket deployment dan panduan yang dapat direproduksi | Task | 5 | [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06), [EP-06-S01](#ep-06-s01), [EP-06-S02](#ep-06-s02) |
| [EP-06-T05](#ep-06-t05) | Tetapkan penerimaan rilis berdasarkan bukti lengkap | Task | 5 | [EP-06-S01](#ep-06-s01), [EP-06-S02](#ep-06-s02), [EP-06-T03](#ep-06-t03), [EP-06-T04](#ep-06-t04) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-01](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-03](2026-11-22-ep-03-pool-routing-dan-kegagalan.md), [EP-04](2026-11-22-ep-04-affinity-dan-maintenance.md), [EP-05](2026-11-22-ep-05-dashboard-dan-observabilitas.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Kapasitas contoh relay dapat kurang untuk 20 stream; pisahkan konfigurasi contoh dan benchmark.
- Ketidaksesuaian client/protokol tidak boleh disembunyikan dengan menghapus skenario wajib.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC adalah **(agent's call)** berdasarkan docs dan tetap draft.
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan draft tidak diasumsikan. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-06-s01"></a>

## [STORY] EP-06-S01 Pakai gateway dari OpenCode CLI yang diuji

**Status:** draft · **Parent:** EP-06 · **Milestone:** M3 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** memakai gateway dari OpenCode CLI, **agar** pekerjaan coding dapat berjalan lewat satu custom provider.

### Konteks

Developer membutuhkan bukti integrasi client nyata, bukan hanya kompatibilitas protokol fixture.

- [PRD](../PRD.md): C-1, C-2, C-3.
- Kualitas terkait: Q-1, Q-10.
- [ROADMAP](../ROADMAP.md): M3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-01. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** versi CLI/runtime provider ditetapkan dan gateway tersedia, **When** developer menambah custom provider dengan endpoint serta token gateway, **Then** alias model dapat dipilih.
- **AC2:** **Given** model yang diuji mendukung percakapan, **When** CLI mengirim request biasa, **Then** percakapan selesai melalui relay yang dapat ditelusuri.
- **AC3:** **Given** model mendukung tool calling, **When** CLI menjalankan satu siklus tool call, **Then** siklus selesai melalui gateway dan tool tetap dieksekusi oleh agent.
- **AC4 (negatif/edge):** **Given** model/client membutuhkan protokol atau sinyal sesi yang belum didukung, **When** kompatibilitas dilaporkan, **Then** versi atau kombinasi tidak diklaim lulus dan kegagalan tercatat.

### Batas dan dependensi

- Di luar scope: Implementasi fitur P1, klaim semua versi client kompatibel, dan SLA yang belum ditetapkan.
- Diblokir oleh: [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-06-s02"></a>

## [STORY] EP-06-S02 Pakai gateway dari OpenCode Desktop yang diuji

**Status:** draft · **Parent:** EP-06 · **Milestone:** M3 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** memakai gateway dari OpenCode Desktop, **agar** jalur yang sama tersedia pada client desktop yang diuji.

### Konteks

Integrasi CLI tidak otomatis membuktikan Desktop dengan runtime/config berbeda bekerja.

- [PRD](../PRD.md): C-1, C-2, C-3.
- Kualitas terkait: Q-1, Q-10.
- [ROADMAP](../ROADMAP.md): M3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-02. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** versi Desktop/runtime provider dan model dicatat, **When** developer menambah provider gateway, **Then** alias model dapat dipilih dari Desktop.
- **AC2:** **Given** konfigurasi Desktop valid, **When** percakapan biasa dijalankan, **Then** respons berhasil diterima melalui relay gateway.
- **AC3:** **Given** model mendukung tool calling, **When** Desktop menjalankan satu siklus tool call, **Then** siklus selesai dan streaming yang digunakan tidak rusak.
- **AC4 (negatif/edge):** **Given** Desktop tidak mendukung konfigurasi/protokol yang diperlukan, **When** integrasi dinilai, **Then** hasil gagal serta batas dukungan dicatat tanpa menganggap kelulusan CLI sebagai pengganti.

### Batas dan dependensi

- Di luar scope: Implementasi fitur P1, klaim semua versi client kompatibel, dan SLA yang belum ditetapkan.
- Diblokir oleh: [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-06-t03"></a>

## [TASK] EP-06-T03 Buktikan target performa dengan dashboard aktif

**Status:** draft · **Parent:** EP-06 · **Milestone:** M3 · **Prioritas:** P0

### Konteks

Target resource perlu dibuktikan pada artefak dan kapasitas relay yang cukup, termasuk monitoring aktif.

- [PRD](../PRD.md): C-2, C-4, C-9, C-10, C-14.
- Kualitas terkait: Q-4, Q-5, Q-6, Q-10, Q-11.
- [ROADMAP](../ROADMAP.md): M3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-19. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** target O-6 disahkan dan fixture pembanding siap, **When** latensi gateway dibanding jalur SOCKS5 yang sama, **Then** p95 tambahan ≤50 ms sesuai Q-4 dan command/konfigurasi tersimpan.
- **AC2:** **Given** host uji 2 vCPU/2 GiB dan kapasitas relay memadai, **When** 20 stream berjalan 30 menit dengan dashboard aktif, **Then** RSS gateway ≤256 MiB sesuai Q-5.
- **AC3:** **Given** benchmark selesai, **When** laporan disusun, **Then** ukuran aset/image, konsumsi WARP, dan total deployment dilaporkan terpisah dari RSS gateway.
- **AC4 (negatif/edge):** **Given** kapasitas kurang, target terlampaui, atau fixture tidak sesuai, **When** hasil ditinjau, **Then** AC-19 tidak diberi status lulus dan hambatan dicatat.

### Batas dan dependensi

- Di luar scope: Implementasi fitur P1, klaim semua versi client kompatibel, dan SLA yang belum ditetapkan.
- Diblokir oleh: [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06), [EP-03-S03](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s03), [EP-01-T06](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-06-t04"></a>

## [TASK] EP-06-T04 Sediakan paket deployment dan panduan yang dapat direproduksi

**Status:** draft · **Parent:** EP-06 · **Milestone:** M3 · **Prioritas:** P0

### Konteks

Operator harus dapat memasang dan memelihara paket yang sama dengan yang diuji.

- [PRD](../PRD.md): C-8, C-10, C-14.
- Kualitas terkait: Q-7, Q-8, Q-10.
- [ROADMAP](../ROADMAP.md): M3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** versi kandidat rilis telah diperiksa kembali, **When** paket dibuat, **Then** binary/image+UI, commit/digest, Compose minimal dua relay, dan contoh konfigurasi tersedia.
- **AC2:** **Given** operator mengikuti panduan dengan secret miliknya, **When** pemasangan bersih dijalankan, **Then** startup, status, akses dashboard, serta integrasi provider dapat direproduksi.
- **AC3:** **Given** relay atau gateway perlu dipelihara, **When** prosedur drain/maintenance/restart/shutdown diikuti, **Then** hasil sesuai kebijakan binding dan termination yang sudah diuji.
- **AC4:** **Given** integrasi CLI/Desktop dan skenario gangguan telah diuji, **When** panduan rilis disusun, **Then** panduan mencatat versi client serta troubleshooting IPv4/IPv6, quota, WARP unreachable, dan stream terputus.
- **AC5 (negatif/edge):** **Given** contoh konfigurasi serta privilege diperiksa, **When** paket ditinjau, **Then** hanya placeholder tersedia; admin privat dan gateway tidak mendapat Docker socket/NET_ADMIN.

### Batas dan dependensi

- Di luar scope: Implementasi fitur P1, klaim semua versi client kompatibel, dan SLA yang belum ditetapkan.
- Diblokir oleh: [EP-05-T06](2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06), [EP-06-S01](#ep-06-s01), [EP-06-S02](#ep-06-s02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-06-t05"></a>

## [TASK] EP-06-T05 Tetapkan penerimaan rilis berdasarkan bukti lengkap

**Status:** draft · **Parent:** EP-06 · **Milestone:** M3 · **Prioritas:** P0

### Konteks

Keputusan rilis harus menggabungkan bukti client, pool, dashboard, performa, dan batas dukungan.

- [PRD](../PRD.md): C-1, C-2, C-3, C-4, C-5, C-6, C-7, C-8, C-9, C-10, C-14.
- Kualitas terkait: Q-1, Q-2, Q-3, Q-4, Q-5, Q-6, Q-7, Q-8, Q-9, Q-10, Q-11.
- [ROADMAP](../ROADMAP.md): M3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-01, AC-02, AC-03, AC-04, AC-05, AC-06, AC-07, AC-08, AC-09, AC-10, AC-11, AC-12, AC-13, AC-14, AC-15, AC-16, AC-17, AC-18, AC-19, AC-20, AC-21, AC-22. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

- Keputusan yang ditangani task ini: O-7, O-8.

### Acceptance criteria

- **AC1:** **Given** artefak rilis yang sama tersedia, **When** suite penerimaan ditinjau, **Then** AC-01–AC-22 mempunyai hasil dan bukti lulus yang dapat ditelusuri.
- **AC2:** **Given** hasil integrasi dan benchmark tersedia, **When** G-1–G-7 serta Q-1–Q-11 dievaluasi, **Then** laporan mencatat hasil, versi, batas kompatibilitas, serta baseline aktual atau TBD tanpa angka rekaan.
- **AC3:** **Given** pemilik produk meninjau kesiapan rilis, **When** O-8 diselesaikan, **Then** penanggung jawab, pengguna awal, target tanggal, dan anggaran rilis ditetapkan.
- **AC4:** **Given** SLA tidak diperlukan atau belum ditetapkan pada O-7, **When** klaim rilis disusun, **Then** tidak ada SLA numerik atau jaminan kompatibilitas/IP yang tidak memiliki bukti.
- **AC5 (negatif/edge):** **Given** satu skenario wajib gagal atau blocker rilis belum selesai, **When** rilis dinilai, **Then** MVP tidak dinyatakan diterima dengan menghilangkan skenario atau menunda dashboard ke P1.

### Batas dan dependensi

- Di luar scope: Implementasi fitur P1, klaim semua versi client kompatibel, dan SLA yang belum ditetapkan.
- Diblokir oleh: [EP-06-S01](#ep-06-s01), [EP-06-S02](#ep-06-s02), [EP-06-T03](#ep-06-t03), [EP-06-T04](#ep-06-t04)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.
