---
title: "Client dikonfigurasi melalui export dan secret store OS"
date: 2026-11-22
version: 2
status: approved
approved_by: pengguna
approved_at: 2026-11-23
planner: none
project: ""
hierarchy: epic
epic_id: EP-11
milestone: "P1"
priority: P1
capabilities: ["C-3", "C-8", "C-16"]
---

# EP-11 Client dikonfigurasi melalui export dan secret store OS

**Milestone:** P1 · **Prioritas:** P1 sesuai pembagian docs · **Status:** approved · **Pemilik:** pengguna · **Tanggal:** TBD

## Ringkasan

Operator mengekspor konfigurasi client yang valid dan client memakai token gateway melalui integrasi secret store OS yang diuji.

## Masalah

Konfigurasi client dan penyimpanan token perlu disederhanakan tanpa menyebarkan credential upstream dari server.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-3, C-8, C-16.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone P1.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), kriteria P1 dikembangkan pada tiket di bawah; bukan perluasan AC MVP yang sudah ada.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- Konfigurasi hasil export dapat dipakai client yang dipilih tanpa membawa credential upstream.
- Token gateway dapat dipakai melalui secret store target dan kegagalannya tidak membocorkan secret.
- Baseline yang belum terukur tetap TBD; metode dan pelaksana pengukuran mengikuti D-005 (angka aktual menyusul M0).

## Scope

- Kontrak client/OS dan jenis secret
- Export konfigurasi provider
- Integrasi secret store token gateway

## Di luar scope

Credential upstream pada agent, browser automation autentikasi, dan dukungan seluruh OS/client tanpa pengujian.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-11-T01](#ep-11-t01) | Tetapkan format export dan integrasi secret store target | Task | 4 | [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01) |
| [EP-11-S02](#ep-11-s02) | Export konfigurasi provider yang valid tanpa secret upstream | Story | 4 | [EP-11-T01](#ep-11-t01) |
| [EP-11-S03](#ep-11-s03) | Pakai token gateway dari secret store OS yang didukung | Story | 4 | [EP-11-S02](#ep-11-s02) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-07](2026-11-22-ep-07-responses-api.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Schema konfigurasi client berubah antarversi; validasi export pada versi yang dipilih.
- Secret store dapat terkunci atau hilang; kegagalan harus eksplisit tanpa fallback plaintext.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC berasal dari usulan agent dan telah disetujui pengguna pada 23 November 2026 ([D-015](../DECISIONS.md#d-015-persetujuan-seluruh-tiket-mvp-dan-p1)).
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan mencakup epic dan seluruh child story/task pada file ini. Status `approved` menyatakan persetujuan tiket; implementasi, pemenuhan dependensi, dan kelulusan AC tetap memerlukan bukti. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-11-t01"></a>

## [TASK] EP-11-T01 Tetapkan format export dan integrasi secret store target

**Status:** approved · **Parent:** EP-11 · **Milestone:** P1 · **Prioritas:** P1

### Konteks

C-16 perlu menetapkan client/OS dan kontrak penyimpanan sebelum menjanjikan integrasi lintas platform.

- [PRD](../PRD.md): C-8, C-16.
- Kualitas terkait: Q-7, Q-8, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** C-16 diprioritaskan, **When** target integrasi dipilih, **Then** versi client, OS, format konfigurasi, serta secret store yang diuji tercatat.
- **AC2:** **Given** jenis secret dan alur pemasangan ditentukan, **When** kontrak dibakukan, **Then** export hanya membawa data provider yang diperlukan dan credential upstream tetap di server.
- **AC3:** **Given** token gateway perlu disimpan, diganti, atau dihapus, **When** perilaku integrasi ditetapkan, **Then** otorisasi akses serta penanganan store terkunci/hilang mempunyai hasil yang dapat diuji.
- **AC4 (negatif/edge):** **Given** platform tidak menyediakan integrasi yang diperlukan, **When** dukungan dinilai, **Then** platform ditandai belum didukung tanpa memasukkan token plaintext diam-diam.

### Batas dan dependensi

- Di luar scope: Credential upstream pada agent, browser automation autentikasi, dan dukungan seluruh OS/client tanpa pengujian.
- Diblokir oleh: [EP-07-T01](2026-11-22-ep-07-responses-api.md#ep-07-t01)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-11-s02"></a>

## [STORY] EP-11-S02 Export konfigurasi provider yang valid tanpa secret upstream

**Status:** approved · **Parent:** EP-11 · **Milestone:** P1 · **Prioritas:** P1

**Sebagai** operator, **saya ingin** mengekspor konfigurasi client, **agar** pemasangan provider lebih sederhana dan secret upstream tetap terpusat.

### Konteks

Operator ingin membagikan konfigurasi endpoint/model tanpa mengubah batas kepemilikan credential.

- [PRD](../PRD.md): C-3, C-8, C-16.
- Kualitas terkait: Q-7, Q-8, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** client target dan mapping gateway valid, **When** operator meminta export, **Then** konfigurasi memuat endpoint, provider, alias, dan referensi token sesuai EP-11-T01.
- **AC2:** **Given** konfigurasi hasil export diterapkan pada client yang diuji, **When** client meminta katalog/generasi, **Then** konfigurasi dapat digunakan melalui gateway.
- **AC3:** **Given** export diperiksa sebagai artefak, **When** kontennya ditinjau, **Then** credential upstream dan secret admin tidak disertakan.
- **AC4 (negatif/edge):** **Given** client/version atau model yang diminta tidak valid, **When** export diminta, **Then** kegagalan eksplisit diberikan tanpa menghasilkan konfigurasi yang diklaim valid.

### Batas dan dependensi

- Di luar scope: Credential upstream pada agent, browser automation autentikasi, dan dukungan seluruh OS/client tanpa pengujian.
- Diblokir oleh: [EP-11-T01](#ep-11-t01)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-11-s03"></a>

## [STORY] EP-11-S03 Pakai token gateway dari secret store OS yang didukung

**Status:** approved · **Parent:** EP-11 · **Milestone:** P1 · **Prioritas:** P1

**Sebagai** developer, **saya ingin** memakai token gateway dari secret store OS, **agar** credential client dapat dikelola tanpa penyimpanan plaintext yang tidak disetujui.

### Konteks

Token client perlu diakses dengan kontrak OS yang jelas tanpa fallback penyimpanan yang tidak disetujui.

- [PRD](../PRD.md): C-8, C-16.
- Kualitas terkait: Q-7, Q-8, Q-10.
- [ROADMAP](../ROADMAP.md): P1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** store tersedia dan pengguna memberi otorisasi yang diperlukan, **When** token gateway disimpan serta digunakan, **Then** client dapat mengakses gateway menurut EP-11-T01.
- **AC2:** **Given** token diperbarui atau dihapus dari store, **When** client memakai integrasi kembali, **Then** perilaku mengikuti perubahan tanpa menggunakan token lama secara tersembunyi.
- **AC3:** **Given** store terkunci, token tidak ada, atau otorisasi ditolak, **When** client membutuhkan token, **Then** error eksplisit ditampilkan tanpa menyalin secret ke file plaintext.
- **AC4 (negatif/edge):** **Given** integrasi dan export ditinjau, **When** log serta artefak diperiksa, **Then** token tidak bocor ke log/export dan credential upstream tetap tidak diberikan kepada client.

### Batas dan dependensi

- Di luar scope: Credential upstream pada agent, browser automation autentikasi, dan dukungan seluruh OS/client tanpa pengujian.
- Diblokir oleh: [EP-11-S02](#ep-11-s02)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.
