---
title: "Maintenance WARP berjalan otomatis tanpa melanggar binding"
date: 2026-11-22
version: 1
status: draft
planner: none
project: ""
hierarchy: epic
epic_id: EP-09
milestone: "P1"
priority: P1
capabilities: ["C-7", "C-9", "C-13"]
---

# EP-09 Maintenance WARP berjalan otomatis tanpa melanggar binding

**Milestone:** P1 · **Prioritas:** P1 sesuai pembagian docs · **Status:** draft · **Pemilik/tanggal:** TBD

## Ringkasan

Controller terpisah melakukan maintenance sesuai kebijakan serta menghentikan tindakan yang tidak aman atau tidak dapat diverifikasi.

## Masalah

Otomasi dari stack referensi perlu memahami drain, binding, dan generasi agar tidak mengganti identitas yang masih digunakan.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-7, C-9, C-13.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone P1.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), kriteria P1 dikembangkan pada tiket di bawah; bukan perluasan AC MVP yang sudah ada.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- Maintenance terkontrol tidak memutus stream relay lain atau memindahkan binding aktif diam-diam.
- Relay hanya kembali menerima assignment setelah kondisi re-enable terpenuhi.
- Baseline yang belum terukur tetap TBD; pemilik pengukuran ditetapkan melalui O-1.

## Scope

- Kebijakan trigger dan otorisasi controller
- Drain dan maintenance satu relay
- Pemeriksaan identitas serta penanganan kegagalan controller

## Di luar scope

Docker socket pada gateway, rotasi key/akun, jaminan IP unik, dan restart akibat 429 tanpa kebijakan yang dibuktikan.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-09-T01](#ep-09-t01) | Tetapkan trigger dan kontrak maintenance controller | Task | 4 | [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06) |
| [EP-09-S02](#ep-09-s02) | Lakukan maintenance relay melalui controller terpisah | Story | 4 | [EP-09-T01](#ep-09-t01) |
| [EP-09-S03](#ep-09-s03) | Ketahui maintenance yang tertahan atau gagal diverifikasi | Story | 4 | [EP-09-S02](#ep-09-s02) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-04](2026-11-22-ep-04-affinity-dan-maintenance.md), [EP-07](2026-11-22-ep-07-responses-api.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Trigger 429 tidak otomatis berarti masalah IP; scope harus dibuktikan.
- Perubahan IP pada probe tidak menjamin identitas untuk semua tujuan; dokumentasikan batas.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC adalah **(agent's call)** berdasarkan docs dan tetap draft.
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan draft tidak diasumsikan. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-09-t01"></a>

## [TASK] EP-09-T01 Tetapkan trigger dan kontrak maintenance controller

**Status:** draft · **Parent:** EP-09 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

C-13 memerlukan controller dengan kebijakan yang berbeda dari rotasi otomatis tanpa koordinasi gateway.

- [PRD](../PRD.md): C-7, C-13.
- Kualitas terkait: Q-3, Q-6, Q-7, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** C-13 diprioritaskan dan prosedur manual telah lulus, **When** kebijakan controller disepakati, **Then** trigger, otorisasi, batas attempt, timeout, dan kondisi berhenti tercatat.
- **AC2:** **Given** binding atau stream masih aktif, **When** precondition maintenance ditetapkan, **Then** kontrak mewajibkan drain serta pemeriksaan sebelum perubahan identitas.
- **AC3:** **Given** controller diberi akses tindakan relay, **When** batas akses ditinjau, **Then** gateway tetap tanpa Docker socket dan akses controller dipisahkan serta dibatasi pada tindakan yang disetujui.
- **AC4 (negatif/edge):** **Given** 429 berscope akun/key atau belum diketahui, **When** trigger dievaluasi, **Then** status itu tidak otomatis memberi izin restart/rekey untuk menambah kuota.

### Batas dan dependensi

- Di luar scope: Docker socket pada gateway, rotasi key/akun, jaminan IP unik, dan restart akibat 429 tanpa kebijakan yang dibuktikan.
- Diblokir oleh: [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01), [EP-04-S06](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-09-s02"></a>

## [STORY] EP-09-S02 Lakukan maintenance relay melalui controller terpisah

**Status:** draft · **Parent:** EP-09 · **Milestone:** P1 · **Prioritas:** P1

**Sebagai** operator, **saya ingin** menjalankan maintenance WARP otomatis yang terkoordinasi, **agar** operasi rutin tetap mematuhi aturan percakapan.

### Konteks

Otomasi harus menjalankan prosedur drain dan identitas yang sudah terbukti secara manual.

- [PRD](../PRD.md): C-7, C-13.
- Kualitas terkait: Q-3, Q-6, Q-7.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** trigger sah dan relay memenuhi kebijakan, **When** controller memulai maintenance, **Then** relay menolak assignment baru dan pekerjaan yang sudah di-admit mengikuti drain.
- **AC2:** **Given** stream dan binding telah memenuhi precondition, **When** controller menjalankan tindakan yang disetujui, **Then** generasi gateway diperbarui sesuai perubahan identitas yang diketahui.
- **AC3:** **Given** maintenance selesai dan health/identitas diperiksa, **When** controller meminta re-enable, **Then** relay baru menerima assignment setelah kondisi kontrak terpenuhi.
- **AC4 (negatif/edge):** **Given** binding aktif belum boleh dibatalkan atau precondition gagal, **When** controller melanjutkan siklus, **Then** restart/rekey ditahan tanpa tindakan paksa diam-diam.

### Batas dan dependensi

- Di luar scope: Docker socket pada gateway, rotasi key/akun, jaminan IP unik, dan restart akibat 429 tanpa kebijakan yang dibuktikan.
- Diblokir oleh: [EP-09-T01](#ep-09-t01)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-09-s03"></a>

## [STORY] EP-09-S03 Ketahui maintenance yang tertahan atau gagal diverifikasi

**Status:** draft · **Parent:** EP-09 · **Milestone:** P1 · **Prioritas:** P1

**Sebagai** operator, **saya ingin** mengetahui maintenance yang belum aman diselesaikan, **agar** kegagalan otomasi tidak tersembunyi di belakang status sehat.

### Konteks

Controller tidak boleh mengulang tindakan tanpa batas atau menyatakan identitas aman tanpa bukti.

- [PRD](../PRD.md): C-9, C-13.
- Kualitas terkait: Q-3, Q-9, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** maintenance sedang berlangsung, **When** operator membaca status, **Then** tahap, relay/generasi, alasan, dan hasil tindakan dapat ditelusuri.
- **AC2:** **Given** health atau pemeriksaan perubahan identitas gagal, **When** controller menyelesaikan attempt, **Then** relay tidak diaktifkan sebagai sehat secara palsu.
- **AC3:** **Given** batas attempt atau deadline tercapai, **When** controller menentukan langkah berikutnya, **Then** tindakan berhenti sesuai kebijakan dan kondisi meminta tindakan operator terlihat.
- **AC4 (negatif/edge):** **Given** controller restart ketika maintenance tertunda, **When** siklus dipulihkan, **Then** controller merekonsiliasi state aktual tanpa menggandakan tindakan yang belum diketahui hasilnya.

### Batas dan dependensi

- Di luar scope: Docker socket pada gateway, rotasi key/akun, jaminan IP unik, dan restart akibat 429 tanpa kebijakan yang dibuktikan.
- Diblokir oleh: [EP-09-S02](#ep-09-s02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.
