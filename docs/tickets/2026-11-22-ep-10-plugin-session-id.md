---
title: "Client memasok identitas sesi dinamis untuk affinity"
date: 2026-11-22
version: 1
status: draft
planner: none
project: ""
hierarchy: epic
epic_id: EP-10
milestone: "P1"
priority: P1
capabilities: ["C-6", "C-15"]
---

# EP-10 Client memasok identitas sesi dinamis untuk affinity

**Milestone:** P1 · **Prioritas:** P1 sesuai pembagian docs · **Status:** draft · **Pemilik/tanggal:** TBD

## Ringkasan

Plugin pada client yang dipilih memasok session ID sesuai lifecycle percakapan dan kontrak strict gateway.

## Masalah

Client tanpa identitas percakapan dinamis hanya mendapat fixed scope bersama dan tidak dapat membuktikan affinity per percakapan.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-6, C-15.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone P1.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), kriteria P1 dikembangkan pada tiket di bawah; bukan perluasan AC MVP yang sudah ada.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- Percakapan pada client yang diuji memperoleh ID dinamis yang mengikuti kontrak sesi.
- Kehilangan affinity tetap terlihat dan tidak ditutupi plugin dengan perpindahan jalur otomatis.
- Baseline yang belum terukur tetap TBD; pemilik pengukuran ditetapkan melalui O-1.

## Scope

- Pemilihan client/plugin dan kontrak sesi
- Pemasangan session ID per percakapan
- Pembuktian multi-turn, restart, dan dokumentasi versi

## Di luar scope

Penyimpanan riwayat oleh gateway, penebakan sesi dari prompt, dan klaim semua client mendukung plugin yang sama.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-10-T01](#ep-10-t01) | Tetapkan client dan lifecycle plugin sesi yang didukung | Task | 4 | [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01), [EP-04-T01](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-t01) |
| [EP-10-S02](#ep-10-s02) | Kirim identitas sesi yang tepat dari percakapan client | Story | 4 | [EP-10-T01](#ep-10-t01) |
| [EP-10-T03](#ep-10-t03) | Buktikan plugin pada percakapan multi-turn dan kegagalan sesi | Task | 4 | [EP-10-S02](#ep-10-s02), [EP-04-S03](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s03) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-04](2026-11-22-ep-04-affinity-dan-maintenance.md), [EP-07](2026-11-22-ep-07-responses-api.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Lifecycle plugin dapat berbeda menurut versi client; pin dan uji versi target.
- ID statis bersama akan mencampur sesi; uji percakapan serta token yang berbeda.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC adalah **(agent's call)** berdasarkan docs dan tetap draft.
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan draft tidak diasumsikan. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-10-t01"></a>

## [TASK] EP-10-T01 Tetapkan client dan lifecycle plugin sesi yang didukung

**Status:** draft · **Parent:** EP-10 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

Plugin harus sesuai kemampuan client nyata dan kontrak O-4 yang sudah selesai.

- [PRD](../PRD.md): C-6, C-15.
- Kualitas terkait: Q-3, Q-8, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** C-15 diprioritaskan, **When** client dan titik integrasi plugin dipilih, **Then** versi, cara pemasangan, serta kemampuan header/sinyal sesi dicatat.
- **AC2:** **Given** percakapan dapat dibuat, dilanjutkan, atau ditutup, **When** kontrak plugin disepakati, **Then** pembentukan ID, sinyal start, pergantian model/upstream, serta expiry mengikuti gateway.
- **AC3:** **Given** client restart atau sesi gateway hilang, **When** perilaku pemulihan ditetapkan, **Then** plugin tidak menandai percakapan lama sebagai sesi baru diam-diam.
- **AC4 (negatif/edge):** **Given** versi client tidak menyediakan hook yang diperlukan, **When** dukungan dinilai, **Then** versi tersebut dinyatakan belum kompatibel dan tidak diberi ID statis bersama sebagai pengganti.

### Batas dan dependensi

- Di luar scope: Penyimpanan riwayat oleh gateway, penebakan sesi dari prompt, dan klaim semua client mendukung plugin yang sama.
- Diblokir oleh: [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01), [EP-04-T01](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-t01)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-10-s02"></a>

## [STORY] EP-10-S02 Kirim identitas sesi yang tepat dari percakapan client

**Status:** draft · **Parent:** EP-10 · **Milestone:** P1 · **Prioritas:** P1

**Sebagai** developer, **saya ingin** mengirim identitas sesi dari client secara otomatis, **agar** affinity per percakapan tidak perlu disusun manual pada setiap request.

### Konteks

Identitas harus melekat pada percakapan, bukan pada seluruh proses client atau isi prompt.

- [PRD](../PRD.md): C-6, C-15.
- Kualitas terkait: Q-3, Q-8.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** percakapan baru dibuat, **When** plugin mengirim request pertama, **Then** session ID serta sinyal pembentukan mengikuti EP-10-T01.
- **AC2:** **Given** percakapan yang sama dilanjutkan, **When** plugin mengirim request berikutnya, **Then** identitas sesi dipertahankan sesuai kontrak.
- **AC3:** **Given** percakapan lain dimulai atau scope berubah menurut kontrak, **When** plugin mengirim request, **Then** sesi tidak tercampur dengan percakapan sebelumnya.
- **AC4 (negatif/edge):** **Given** prompt berubah atau gateway memberi affinity_lost, **When** plugin menangani kondisi, **Then** plugin tidak menebak ID dari prompt atau mengganti relay/sesi secara diam-diam.

### Batas dan dependensi

- Di luar scope: Penyimpanan riwayat oleh gateway, penebakan sesi dari prompt, dan klaim semua client mendukung plugin yang sama.
- Diblokir oleh: [EP-10-T01](#ep-10-t01)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-10-t03"></a>

## [TASK] EP-10-T03 Buktikan plugin pada percakapan multi-turn dan kegagalan sesi

**Status:** draft · **Parent:** EP-10 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

Pemasangan header saja tidak membuktikan percakapan dan binding benar-benar konsisten.

- [PRD](../PRD.md): C-6, C-15.
- Kualitas terkait: Q-3, Q-8, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** plugin dipasang pada client yang dipilih, **When** percakapan multi-turn dan tool calling dijalankan, **Then** gateway mencatat binding pada relay/generasi yang sama.
- **AC2:** **Given** beberapa percakapan/token berjalan, **When** pengujian isolasi dilakukan, **Then** session ID yang sama secara tekstual tidak mencampur scope token.
- **AC3:** **Given** gateway restart, TTL habis, atau binding invalid, **When** client melanjutkan, **Then** error dan kebutuhan memulai sesi baru terlihat sesuai kontrak.
- **AC4 (negatif/edge):** **Given** hasil integrasi didokumentasikan, **When** versi dan panduan diterbitkan, **Then** raw session ID tidak masuk log default dan versi yang belum diuji tidak diklaim didukung.

### Batas dan dependensi

- Di luar scope: Penyimpanan riwayat oleh gateway, penebakan sesi dari prompt, dan klaim semua client mendukung plugin yang sama.
- Diblokir oleh: [EP-10-S02](#ep-10-s02), [EP-04-S03](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s03)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.
