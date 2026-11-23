---
title: "Agent dapat memakai Responses API yang diuji"
date: 2026-11-22
version: 2
status: approved
approved_by: pengguna
approved_at: 2026-11-23
planner: none
project: ""
hierarchy: epic
epic_id: EP-07
milestone: "P1"
priority: P1
capabilities: ["C-11", "C-12", "C-13", "C-15", "C-16"]
---

# EP-07 Agent dapat memakai Responses API yang diuji

**Milestone:** P1 · **Prioritas:** P1 sesuai pembagian docs · **Status:** approved · **Pemilik:** pengguna · **Tanggal:** TBD

## Ringkasan

Kontrak Responses yang dipilih produk berjalan melalui gateway dengan pengujian kompatibilitas tersendiri.

## Masalah

Sebagian client/model memerlukan Responses API, sedangkan MVP hanya menerima Chat Completions.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-11, C-12, C-13, C-15, C-16.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone P1.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), kriteria P1 dikembangkan pada tiket di bawah; bukan perluasan AC MVP yang sudah ada.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- Bagian kontrak Responses yang disepakati mempunyai uji dan integrasi client lulus.
- Kontrak Chat Completions yang sudah diterima tidak mengalami regresi.
- Baseline yang belum terukur tetap TBD; metode dan pelaksana pengukuran mengikuti D-005 (angka aktual menyusul M0).

## Scope

- Task bersama untuk urutan backlog P1
- Kontrak Responses yang disepakati
- Jalur request/stream Responses dan bukti client

## Di luar scope

Perubahan cakupan MVP secara retroaktif, translasi Anthropic/Gemini native, dan klaim seluruh fitur Responses tanpa pengujian.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-07-T01](#ep-07-t01) | Tetapkan urutan backlog P1 berdasarkan kebutuhan produk | Task | 4 | [EP-06-T05](2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t05) |
| [EP-07-T02](#ep-07-t02) | Tetapkan kontrak Responses dan client yang didukung | Task | 4 | [EP-07-T01](#ep-07-t01) |
| [EP-07-S03](#ep-07-s03) | Kirim request Responses melalui relay yang dipilih gateway | Story | 4 | [EP-07-T02](#ep-07-t02) |
| [EP-07-T04](#ep-07-t04) | Buktikan integrasi Responses dan regresi protokol lama | Task | 4 | [EP-07-S03](#ep-07-s03) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-06](2026-11-22-ep-06-integrasi-dan-rilis-mvp.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Client dapat memakai varian kontrak berbeda; tentukan versi dan bagian API yang dijamin.
- Adapter baru dapat mengubah semantik retry; jalankan kembali skenario transport yang relevan.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC berasal dari usulan agent dan telah disetujui pengguna pada 23 November 2026 ([D-015](../DECISIONS.md#d-015-persetujuan-seluruh-tiket-mvp-dan-p1)).
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan mencakup epic dan seluruh child story/task pada file ini. Status `approved` menyatakan persetujuan tiket; implementasi, pemenuhan dependensi, dan kelulusan AC tetap memerlukan bukti. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-07-t01"></a>

## [TASK] EP-07-T01 Tetapkan urutan backlog P1 berdasarkan kebutuhan produk

**Status:** approved · **Parent:** EP-07 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

O-9 mengatur semua epic P1 dan tidak boleh memaksa Responses API menjadi fitur pertama.

- [PRD](../PRD.md): C-11, C-12, C-13, C-15, C-16.
- Kualitas terkait: kriteria keputusan dan reproduksibilitas pada ROADMAP.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- Keputusan yang ditangani task ini: O-9.

D-012 menetapkan urutan C-11 → C-15 → C-16 → C-13 → C-12 dan menunda hot reload; task ini mencatat alasan, dependensi kontrak, serta penanggung jawab tiap epic saat ditutup setelah MVP diterima.

### Acceptance criteria

- **AC1:** **Given** MVP telah diterima, **When** pemilik produk meninjau C-11/C-12/C-13/C-15/C-16, **Then** urutan D-012 (C-11 → C-15 → C-16 → C-13 → C-12) beserta alasannya tercatat dan dikonfirmasi ulang.
- **AC2:** **Given** kemampuan P1 dipilih untuk dimulai, **When** input dan penanggung jawabnya ditetapkan, **Then** dependensi serta keputusan kontrak yang perlu ditutup mempunyai pemilik.
- **AC3:** **Given** hot reload dibahas, **When** prioritas ditentukan, **Then** keputusan D-012 dicatat: tidak diprioritaskan dan tetap tanpa tiket sampai ada keputusan cakupan baru.
- **AC4 (negatif/edge):** **Given** produk memilih epic selain Responses lebih dahulu, **When** task ini ditutup, **Then** epic yang dipilih dapat dimulai tanpa menunggu EP-07-T02/S03/T04.

### Batas dan dependensi

- Di luar scope: Implementasi kemampuan P1 dan perubahan syarat penerimaan MVP.
- Diblokir oleh: [EP-06-T05](2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t05)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-07-t02"></a>

## [TASK] EP-07-T02 Tetapkan kontrak Responses dan client yang didukung

**Status:** approved · **Parent:** EP-07 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

C-11 mensyaratkan kontrak terpisah sebelum dukungan protokol diumumkan.

- [PRD](../PRD.md): C-11.
- Kualitas terkait: Q-1, Q-2, Q-3, Q-7, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** C-11 diprioritaskan melalui O-9, **When** kontrak Responses ditinjau, **Then** versi client/upstream, operasi, mode JSON/stream, tool calling, usage, dan field yang didukung tercatat.
- **AC2:** **Given** kontrak yang didukung telah dipilih, **When** kriteria kompatibilitas dibuat, **Then** fixture, error, pembatalan, dan batas replay/affinity mempunyai hasil yang dapat diverifikasi.
- **AC3:** **Given** suatu operasi Responses tidak didukung, **When** kontrak penolakan ditetapkan, **Then** error eksplisit didefinisikan tanpa pengalihan diam-diam ke Chat Completions.
- **AC4 (negatif/edge):** **Given** dukungan baru memerlukan perubahan batas arsitektur, **When** desain ditinjau, **Then** keputusan dicatat sebelum implementasi atau klaim dukungan.

### Batas dan dependensi

- Di luar scope: Implementasi adapter dan dukungan operasi yang belum disepakati.
- Diblokir oleh: [EP-07-T01](#ep-07-t01)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-07-s03"></a>

## [STORY] EP-07-S03 Kirim request Responses melalui relay yang dipilih gateway

**Status:** approved · **Parent:** EP-07 · **Milestone:** P1 · **Prioritas:** P1

**Sebagai** developer, **saya ingin** memakai Responses API melalui gateway, **agar** client yang membutuhkan protokol tersebut dapat dipakai dengan pool WARP.

### Konteks

Client yang memerlukan Responses membutuhkan jalur end to end yang mempertahankan aturan transport gateway.

- [PRD](../PRD.md): C-11.
- Kualitas terkait: Q-1, Q-2, Q-3, Q-6, Q-7.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** request sesuai EP-07-T02 dan token valid, **When** client meminta Responses, **Then** model/upstream yang dikonfigurasi dipakai melalui relay eligible.
- **AC2:** **Given** upstream mengirim respons JSON atau stream yang disepakati, **When** client menerima hasil, **Then** content, tool-call data, dan usage yang didukung mempertahankan urutan serta semantik.
- **AC3:** **Given** client disconnect atau pengiriman ambigu, **When** request diproses, **Then** pembatalan dan larangan replay mengikuti kontrak gateway.
- **AC4 (negatif/edge):** **Given** request memakai operasi di luar kontrak, **When** gateway memprosesnya, **Then** error unsupported diberikan tanpa translasi diam-diam.

### Batas dan dependensi

- Di luar scope: Perubahan cakupan MVP secara retroaktif, translasi Anthropic/Gemini native, dan klaim seluruh fitur Responses tanpa pengujian.
- Diblokir oleh: [EP-07-T02](#ep-07-t02)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-07-t04"></a>

## [TASK] EP-07-T04 Buktikan integrasi Responses dan regresi protokol lama

**Status:** approved · **Parent:** EP-07 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

Lulus handler Responses saja tidak membuktikan client nyata atau menjaga fitur yang sudah dirilis.

- [PRD](../PRD.md): C-11.
- Kualitas terkait: Q-1, Q-2, Q-3, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** adapter Responses selesai, **When** suite fixture kontrak dijalankan, **Then** seluruh skenario EP-07-T02 mempunyai bukti lulus.
- **AC2:** **Given** versi client dan upstream yang dipilih tersedia, **When** smoke test nyata dijalankan, **Then** alur yang diklaim didukung berhasil dan versinya dicatat.
- **AC3:** **Given** fitur Responses aktif, **When** suite Chat Completions dan skenario transport relevan dijalankan, **Then** kontrak lama tetap lulus.
- **AC4 (negatif/edge):** **Given** suatu fitur atau kombinasi belum diuji atau gagal, **When** matriks dukungan diterbitkan, **Then** fitur itu tidak diiklankan sebagai kompatibel.

### Batas dan dependensi

- Di luar scope: Perubahan cakupan MVP secara retroaktif, translasi Anthropic/Gemini native, dan klaim seluruh fitur Responses tanpa pengujian.
- Diblokir oleh: [EP-07-S03](#ep-07-s03)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.
