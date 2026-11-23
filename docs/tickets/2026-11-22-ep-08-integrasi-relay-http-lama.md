---
title: "Operator mempertahankan stack relay HTTP yang sudah ada"
date: 2026-11-22
version: 1
status: draft
planner: none
project: ""
hierarchy: epic
epic_id: EP-08
milestone: "P1"
priority: P1
capabilities: ["C-12"]
---

# EP-08 Operator mempertahankan stack relay HTTP yang sudah ada

**Milestone:** P1 · **Prioritas:** P1 sesuai pembagian docs · **Status:** draft · **Pemilik/tanggal:** TBD

## Ringkasan

Gateway dapat memakai stack relay HTTP referensi yang dipin dengan kontrak auth, target, dan streaming yang teruji.

## Masalah

Operator yang mempertahankan warp-relay-pool lama memerlukan adapter protokol header tanpa membuka target bebas.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-12.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone P1.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), kriteria P1 dikembangkan pada tiket di bawah; bukan perluasan AC MVP yang sudah ada.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- Adapter bekerja pada versi referensi yang dicatat dan seluruh skenario kontraknya lulus.
- Client tetap menggunakan token/endpoint gateway tanpa menentukan target bebas.
- Baseline yang belum terukur tetap TBD; pemilik pengukuran ditetapkan melalui O-1.

## Scope

- Kontrak dan versi relay referensi
- Adapter relay HTTP melalui konfigurasi server
- Integrasi dan panduan stack lama

## Di luar scope

Public forward proxy, CONNECT, URL target dari client, dan kewajiban memakai adapter lama pada deployment MVP.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-08-T01](#ep-08-t01) | Tetapkan kontrak stack relay HTTP yang akan diintegrasikan | Task | 4 | [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01) |
| [EP-08-S02](#ep-08-s02) | Gunakan stack relay lama melalui endpoint provider gateway | Story | 4 | [EP-08-T01](#ep-08-t01) |
| [EP-08-T03](#ep-08-t03) | Sediakan bukti kompatibilitas dan panduan adapter lama | Task | 4 | [EP-08-S02](#ep-08-s02) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-07](2026-11-22-ep-07-responses-api.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Protokol/sumber referensi dapat berubah; pin commit dan verifikasi perilaku.
- Header relay dapat membocorkan akses bila diteruskan ke client/upstream; uji redaksi.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC adalah **(agent's call)** berdasarkan docs dan tetap draft.
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan draft tidak diasumsikan. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-08-t01"></a>

## [TASK] EP-08-T01 Tetapkan kontrak stack relay HTTP yang akan diintegrasikan

**Status:** draft · **Parent:** EP-08 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

Integrasi harus didasarkan pada perilaku source/version referensi yang benar-benar akan digunakan.

- [PRD](../PRD.md): C-12.
- Kualitas terkait: Q-1, Q-2, Q-3, Q-7, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** C-12 diprioritaskan, **When** source/image relay referensi dipilih, **Then** commit/digest, lisensi, protokol header, auth, serta path mapping dicatat.
- **AC2:** **Given** stack lama mempunyai health dan timeout sendiri, **When** kontrak ditetapkan, **Then** perbedaan streaming, cancellation, retry, dan health dibanding jalur SOCKS5 dijelaskan.
- **AC3:** **Given** relay menerima header penentu target, **When** batas kepercayaan ditetapkan, **Then** target hanya berasal dari registry server gateway.
- **AC4 (negatif/edge):** **Given** versi referensi tidak dapat menjaga aturan transport gateway, **When** kelayakan ditinjau, **Then** kombinasi dinyatakan belum didukung tanpa melonggarkan larangan replay atau target bebas.

### Batas dan dependensi

- Di luar scope: Public forward proxy, CONNECT, URL target dari client, dan kewajiban memakai adapter lama pada deployment MVP.
- Diblokir oleh: [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-08-s02"></a>

## [STORY] EP-08-S02 Gunakan stack relay lama melalui endpoint provider gateway

**Status:** draft · **Parent:** EP-08 · **Milestone:** P1 · **Prioritas:** P1

**Sebagai** operator, **saya ingin** memakai pool HTTP relay lama melalui gateway, **agar** client tetap menggunakan satu kontrak provider.

### Konteks

Pengguna stack lama tetap memerlukan kontrak provider biasa dan pemisahan credential.

- [PRD](../PRD.md): C-12.
- Kualitas terkait: Q-1, Q-2, Q-3, Q-7, Q-8.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** adapter dan mapping stack lama dikonfigurasi, **When** client mengirim request gateway, **Then** server menyusun target/path/header relay menurut EP-08-T01.
- **AC2:** **Given** stack lama mengirim JSON atau SSE, **When** client menerima respons, **Then** semantik dan urutan data yang disepakati dipertahankan.
- **AC3:** **Given** client mencoba menyuplai header target atau origin bebas, **When** gateway menerima request, **Then** nilai tersebut ditolak dan tidak mengganti registry server.
- **AC4 (negatif/edge):** **Given** stack lama gagal setelah dispatch ambigu atau memberi redirect, **When** gateway menangani kegagalan, **Then** request tidak diulang otomatis dan credential tidak dikirim ke origin lain.

### Batas dan dependensi

- Di luar scope: Public forward proxy, CONNECT, URL target dari client, dan kewajiban memakai adapter lama pada deployment MVP.
- Diblokir oleh: [EP-08-T01](#ep-08-t01)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-08-t03"></a>

## [TASK] EP-08-T03 Sediakan bukti kompatibilitas dan panduan adapter lama

**Status:** draft · **Parent:** EP-08 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

Operator perlu mengetahui stack/version yang didukung serta perbedaan konfigurasinya.

- [PRD](../PRD.md): C-12.
- Kualitas terkait: Q-2, Q-3, Q-7, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** adapter tersedia, **When** fixture dan integrasi versi referensi dijalankan, **Then** auth, mapping target, JSON/SSE, disconnect, serta batas attempt lulus.
- **AC2:** **Given** deployment lama dipertahankan, **When** panduan diikuti, **Then** gateway dapat dihubungkan tanpa membagikan credential upstream atau port relay kepada client.
- **AC3:** **Given** adapter lama tidak dikonfigurasi, **When** deployment SOCKS5 MVP dijalankan, **Then** jalur langsung SOCKS5 tetap berfungsi tanpa service relay HTTP tambahan.
- **AC4 (negatif/edge):** **Given** source/version lain belum diuji, **When** dukungan didokumentasikan, **Then** batas versi dan perbedaan perilaku disebutkan tanpa klaim universal.

### Batas dan dependensi

- Di luar scope: Public forward proxy, CONNECT, URL target dari client, dan kewajiban memakai adapter lama pada deployment MVP.
- Diblokir oleh: [EP-08-S02](#ep-08-s02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.
