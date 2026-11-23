---
title: "Percakapan mempertahankan jalur saat relay dipelihara"
date: 2026-11-22
version: 1
status: draft
planner: none
project: ""
hierarchy: epic
epic_id: EP-04
milestone: "M2.2"
priority: P0
capabilities: ["C-6", "C-7", "C-10"]
---

# EP-04 Percakapan mempertahankan jalur saat relay dipelihara

**Milestone:** M2.2 · **Prioritas:** P0 sesuai pembagian docs · **Status:** draft · **Pemilik:** pengguna · **Tanggal:** TBD

## Ringkasan

Binding strict, drain, perubahan generasi, dan shutdown mempunyai perilaku eksplisit yang dapat dibuktikan.

## Masalah

Perubahan relay atau identitas dapat merusak percakapan, sedangkan restart gateway menghapus state sementara.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-6, C-7, C-10.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone M2.2.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), AC-13, AC-14, AC-15.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- AC-13–AC-15 lulus dengan binding aktif, restart, dan perubahan identitas yang diketahui.
- Tidak ada rebinding atau penghentian paksa diam-diam akibat deadline.
- Baseline yang belum terukur tetap TBD; metode dan pelaksana pengukuran mengikuti D-005 (angka aktual menyusul M0).

## Scope

- Kontrak sesi dan restart
- Binding token/upstream/model/session
- Invalidation generasi dan maintenance
- Kebijakan serta perilaku shutdown

## Di luar scope

Plugin session ID otomatis, persistence percakapan, controller WARP otomatis, dan jaminan IP absolut yang tidak dapat dibuktikan.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-04-T01](#ep-04-t01) | Tetapkan kontrak sesi strict yang dapat dibuktikan | Task | 4 | [EP-01-T01](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t01), [EP-02-S04](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s04) |
| [EP-04-S02](#ep-04-s02) | Lanjutkan sesi pada relay dan generasi yang sama | Story | 4 | [EP-04-T01](#ep-04-t01), [EP-03-S02](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02) |
| [EP-04-S03](#ep-04-s03) | Kenali kehilangan binding setelah restart atau perubahan identitas | Story | 4 | [EP-04-S02](#ep-04-s02), [EP-03-S06](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s06) |
| [EP-04-S04](#ep-04-s04) | Pelihara satu relay tanpa memutus pekerjaan pada relay lain | Story | 4 | [EP-04-S03](#ep-04-s03), [EP-03-S02](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02) |
| [EP-04-T05](#ep-04-t05) | Tetapkan kebijakan shutdown ketika drain belum selesai | Task | 4 | [EP-04-T01](#ep-04-t01), [EP-03-S06](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s06) |
| [EP-04-S06](#ep-04-s06) | Hentikan gateway sesuai kebijakan drain yang terdokumentasi | Story | 4 | [EP-04-T05](#ep-04-t05), [EP-04-S04](#ep-04-s04) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-01](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](2026-11-22-ep-02-gateway-api-dasar.md), [EP-03](2026-11-22-ep-03-pool-routing-dan-kegagalan.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Client mungkin tidak mendukung sinyal sesi dinamis; batasi klaim kompatibilitas.
- Deadline container dapat memutus stream; selaraskan O-5 dengan prosedur operator.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC adalah **(agent's call)** berdasarkan docs dan tetap draft.
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan draft tidak diasumsikan. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-04-t01"></a>

## [TASK] EP-04-T01 Tetapkan kontrak sesi strict yang dapat dibuktikan

**Status:** draft · **Parent:** EP-04 · **Milestone:** M2.2 · **Prioritas:** P0

### Konteks

O-4 harus menghasilkan kontrak pembentukan dan lanjutan sesi sebelum implementasi binding.

- [PRD](../PRD.md): C-6.
- Kualitas terkait: Q-3, Q-10.
- [ROADMAP](../ROADMAP.md): M2.2; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-13, AC-14. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

- Keputusan yang ditangani task ini: O-4.

D-013 menetapkan MVP memakai fixed routing dan lifecycle strict dibuktikan lewat fixture; task ini menetapkan skema header sesi serta ambang TTL yang dipakai fixture tersebut.

### Acceptance criteria

- **AC1:** **Given** kemampuan client target telah dicatat, **When** kontrak sesi ditetapkan, **Then** dokumen membedakan sesi baru/lanjutan, scope token/upstream/model, TTL, expiry, dan kapasitas.
- **AC2:** **Given** gateway restart dan binding memory hilang, **When** sesi lanjutan dievaluasi menurut kontrak, **Then** dokumen menetapkan penolakan eksplisit alih-alih membuat binding baru diam-diam.
- **AC3:** **Given** client tidak menyediakan sinyal sesi yang dibutuhkan, **When** kompatibilitas dievaluasi, **Then** fixed scope bersama dan batas lifetime proses dijelaskan tanpa klaim isolasi percakapan.
- **AC4 (negatif/edge):** **Given** kebutuhan model tidak dapat dibuktikan dengan integrasi yang dipilih, **When** hasil fixture dibandingkan dengan D-013, **Then** kombinasi dinyatakan tidak kompatibel dan tidak dipromosikan sebagai strict per percakapan.

### Batas dan dependensi

- Di luar scope: Plugin session ID otomatis, persistence percakapan, controller WARP otomatis, dan jaminan IP absolut yang tidak dapat dibuktikan.
- Diblokir oleh: [EP-01-T01](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t01), [EP-02-S04](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s04)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-04-s02"></a>

## [STORY] EP-04-S02 Lanjutkan sesi pada relay dan generasi yang sama

**Status:** draft · **Parent:** EP-04 · **Milestone:** M2.2 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** melanjutkan percakapan pada jalur yang terikat, **agar** model yang memerlukan affinity menerima lanjutan secara konsisten.

### Konteks

Sesi yang memerlukan konsistensi harus terikat sebelum dispatch dan tetap terisolasi menurut token.

- [PRD](../PRD.md): C-6.
- Kualitas terkait: Q-3.
- [ROADMAP](../ROADMAP.md): M2.2; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-13. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** sesi baru valid menurut EP-04-T01, **When** request pertama di-admit, **Then** binding atomik menyimpan relay/generasi pada scope token/upstream/model/session.
- **AC2:** **Given** binding aktif dan relay masih eligible, **When** sesi mengirim request lanjutan, **Then** relay dan generasi yang sama digunakan.
- **AC3:** **Given** dua token memakai session ID tekstual yang sama, **When** masing-masing mengirim request, **Then** binding tidak tercampur antaridentitas token.
- **AC4 (negatif/edge):** **Given** batas binding tercapai atau konfigurasi strict digabung least-inflight, **When** sesi baru atau konfigurasi diproses, **Then** binding aktif tidak digusur dan kombinasi yang tidak valid ditolak.

### Batas dan dependensi

- Di luar scope: Plugin session ID otomatis, persistence percakapan, controller WARP otomatis, dan jaminan IP absolut yang tidak dapat dibuktikan.
- Diblokir oleh: [EP-04-T01](#ep-04-t01), [EP-03-S02](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-04-s03"></a>

## [STORY] EP-04-S03 Kenali kehilangan binding setelah restart atau perubahan identitas

**Status:** draft · **Parent:** EP-04 · **Milestone:** M2.2 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** mengetahui bahwa jalur sesi sudah tidak valid, **agar** percakapan tidak diteruskan dengan identitas yang berbeda tanpa pemberitahuan.

### Konteks

Restart atau perubahan identitas tidak boleh menjadi alasan menganggap percakapan lama sebagai sesi baru.

- [PRD](../PRD.md): C-6.
- Kualitas terkait: Q-3, Q-9.
- [ROADMAP](../ROADMAP.md): M2.2; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-13, AC-14. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** binding merujuk generasi yang telah dibatalkan, **When** request lanjutan diterima, **Then** gateway mengembalikan 409 affinity_lost tanpa relay pengganti.
- **AC2:** **Given** gateway telah restart, **When** client melanjutkan sesi eksplisit sesuai kontrak, **Then** sesi lama ditolak dan tidak dibuat ulang diam-diam.
- **AC3:** **Given** binding kedaluwarsa menurut kontrak, **When** client mengirim lanjutan sesi, **Then** hasil mengikuti penolakan sesi lama yang ditetapkan EP-04-T01.
- **AC4 (negatif/edge):** **Given** probe identitas tidak dapat membuktikan IP yang dilihat semua tujuan, **When** status atau kompatibilitas model dilaporkan, **Then** gateway menyatakan batas observasi tanpa menjamin identitas absolut.

### Batas dan dependensi

- Di luar scope: Plugin session ID otomatis, persistence percakapan, controller WARP otomatis, dan jaminan IP absolut yang tidak dapat dibuktikan.
- Diblokir oleh: [EP-04-S02](#ep-04-s02), [EP-03-S06](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-04-s04"></a>

## [STORY] EP-04-S04 Pelihara satu relay tanpa memutus pekerjaan pada relay lain

**Status:** draft · **Parent:** EP-04 · **Milestone:** M2.2 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** melakukan maintenance relay secara terkontrol, **agar** percakapan dan relay lain tetap mengikuti kontraknya.

### Konteks

Maintenance harus mempertimbangkan stream aktif dan binding yang masih memerlukan identitas lama.

- [PRD](../PRD.md): C-6, C-7.
- Kualitas terkait: Q-3, Q-6.
- [ROADMAP](../ROADMAP.md): M2.2; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-15. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** operator yang berwenang memulai drain, **When** request baru datang, **Then** relay tidak menerima assignment baru termasuk untuk sesi yang terikat.
- **AC2:** **Given** stream sudah berjalan sebelum drain, **When** request pada relay lain diproses, **Then** stream yang sudah di-admit dan relay lain tidak dihentikan hanya karena drain.
- **AC3:** **Given** inflight nol tetapi binding strict masih aktif, **When** maintenance identitas diminta, **Then** prosedur tetap menahan restart/rekey sampai binding selesai atau dibatalkan eksplisit.
- **AC4 (negatif/edge):** **Given** syarat maintenance terpenuhi dan operator mengganti identitas, **When** relay diaktifkan kembali, **Then** generasi serta health diperbarui sebelum assignment baru diperbolehkan.

### Batas dan dependensi

- Di luar scope: Plugin session ID otomatis, persistence percakapan, controller WARP otomatis, dan jaminan IP absolut yang tidak dapat dibuktikan.
- Diblokir oleh: [EP-04-S03](#ep-04-s03), [EP-03-S02](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-04-t05"></a>

## [TASK] EP-04-T05 Tetapkan kebijakan shutdown ketika drain belum selesai

**Status:** draft · **Parent:** EP-04 · **Milestone:** M2.2 · **Prioritas:** P0

### Konteks

O-5 memblokir prosedur shutdown yang konsisten dengan deadline dan termination container.

- [PRD](../PRD.md): C-7, C-10.
- Kualitas terkait: Q-6.
- [ROADMAP](../ROADMAP.md): M2.2; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- Keputusan yang ditangani task ini: O-5.

D-014 menetapkan drain 60 detik yang hanya menolak admission baru, batas keras 30 menit, dan pencatatan eksplisit sebelum keluar; task ini menetapkan nilai final `stop_grace_period`, format log shutdown, dan exit code.

### Acceptance criteria

- **AC1:** **Given** stream atau binding masih aktif saat shutdown, **When** kebijakan disepakati, **Then** dokumen menetapkan admission, readiness, deadline, status pekerjaan tersisa, dan tindakan operator.
- **AC2:** **Given** runtime container mempunyai grace period, **When** kebijakan termination ditinjau, **Then** konsekuensi sinyal/termination dan kemungkinan stream terputus dijelaskan.
- **AC3:** **Given** deadline drain tercapai, **When** kebijakan dinilai, **Then** deadline tidak diperlakukan sebagai persetujuan penghentian paksa diam-diam.
- **AC4 (negatif/edge):** **Given** kondisi tidak selesai dapat direproduksi, **When** skenario uji dirumuskan, **Then** kriteria membedakan shutdown selesai, drain tertahan, dan tindakan paksa eksplisit.

### Batas dan dependensi

- Di luar scope: Plugin session ID otomatis, persistence percakapan, controller WARP otomatis, dan jaminan IP absolut yang tidak dapat dibuktikan.
- Diblokir oleh: [EP-04-T01](#ep-04-t01), [EP-03-S06](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-04-s06"></a>

## [STORY] EP-04-S06 Hentikan gateway sesuai kebijakan drain yang terdokumentasi

**Status:** draft · **Parent:** EP-04 · **Milestone:** M2.2 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** menghentikan gateway dengan perilaku yang diketahui, **agar** pemeliharaan deployment tidak bergantung pada asumsi deadline.

### Konteks

Operator memerlukan shutdown nyata yang mengikuti kebijakan, bukan hanya dokumen.

- [PRD](../PRD.md): C-7, C-10.
- Kualitas terkait: Q-6, Q-10.
- [ROADMAP](../ROADMAP.md): M2.2; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** operator memulai shutdown terencana, **When** gateway menangani sinyal, **Then** admission baru dihentikan dan readiness menjadi tidak siap.
- **AC2:** **Given** request aktif selesai sebelum batas kebijakan, **When** drain diproses, **Then** resource dilepas dan gateway dapat berhenti sesuai EP-04-T05.
- **AC3:** **Given** request atau binding belum selesai pada deadline, **When** status shutdown diperiksa, **Then** kondisi tersisa dan tindakan berikutnya mengikuti kebijakan tanpa penanda selesai palsu.
- **AC4 (negatif/edge):** **Given** shutdown dilakukan pada artefak container, **When** skenario termination dijalankan, **Then** hasil cocok dengan prosedur dan tidak mengklaim semua stream selalu selesai.

### Batas dan dependensi

- Di luar scope: Plugin session ID otomatis, persistence percakapan, controller WARP otomatis, dan jaminan IP absolut yang tidak dapat dibuktikan.
- Diblokir oleh: [EP-04-T05](#ep-04-t05), [EP-04-S04](#ep-04-s04)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.
