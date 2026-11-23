---
title: "Request tetap terkendali ketika kondisi relay berubah"
date: 2026-11-22
version: 1
status: draft
planner: none
project: ""
hierarchy: epic
epic_id: EP-03
milestone: "M2.1"
priority: P0
capabilities: ["C-2", "C-4", "C-5", "C-9"]
---

# EP-03 Request tetap terkendali ketika kondisi relay berubah

**Milestone:** M2.1 · **Prioritas:** P0 sesuai pembagian docs · **Status:** draft · **Pemilik/tanggal:** TBD

## Ringkasan

Gateway memilih relay eligible, membatasi kapasitas, dan menangani kegagalan tanpa direct fallback atau retry ambigu.

## Masalah

Pool tanpa reservasi konsisten, pemisahan health, dan batas retry dapat menggandakan generasi atau menghabiskan resource.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-2, C-4, C-5, C-9.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone M2.1.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), AC-07, AC-08, AC-09, AC-12, AC-16, AC-18.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- AC-07–AC-09, AC-12, AC-15–AC-16, dan AC-18 lulus sesuai scope M2.1.
- Tidak ada request generasi kedua otomatis setelah pengiriman ambigu atau stream dimulai.
- Baseline yang belum terukur tetap TBD; metode dan pelaksana pengukuran mengikuti D-005 (angka aktual menyusul M0).

## Scope

- Reservasi kapasitas dan fixed routing
- Health/circuit dan least-inflight stateless
- Fallback, cooldown, cancellation, dan timeout
- Batas total buffer dan beban client lambat

## Di luar scope

Rebinding strict otomatis, pergantian upstream/key, rotasi WARP otomatis, serta controller P1.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-03-S01](#ep-03-s01) | Kirim request hanya ketika relay dan kapasitas tersedia | Story | 4 | [EP-02-S05](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s05), [EP-01-T06](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06) |
| [EP-03-S02](#ep-03-s02) | Tentukan eligibility dari health dan mode operasi yang terpisah | Story | 4 | [EP-03-S01](#ep-03-s01), [EP-02-T06](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-t06) |
| [EP-03-S03](#ep-03-s03) | Sebarkan request stateless ke relay dengan inflight terendah | Story | 4 | [EP-03-S02](#ep-03-s02) |
| [EP-03-S04](#ep-03-s04) | Hindari duplikasi generasi saat failover | Story | 4 | [EP-03-S02](#ep-03-s02), [EP-01-T03](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03) |
| [EP-03-S05](#ep-03-s05) | Kenali batas upstream tanpa merotasi relay secara sembarang | Story | 4 | [EP-03-S04](#ep-03-s04) |
| [EP-03-S06](#ep-03-s06) | Batalkan pekerjaan upstream dan lepaskan resource tepat waktu | Story | 4 | [EP-03-S01](#ep-03-s01), [EP-03-S04](#ep-03-s04) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-01](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](2026-11-22-ep-02-gateway-api-dasar.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Reservasi serentak dapat melebihi batas; uji konkurensi dan pelepasan slot tepat sekali.
- 429 dapat berscope credential; klasifikasi konservatif tanpa mematikan relay global.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC adalah **(agent's call)** berdasarkan docs dan tetap draft.
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan draft tidak diasumsikan. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-03-s01"></a>

## [STORY] EP-03-S01 Kirim request hanya ketika relay dan kapasitas tersedia

**Status:** draft · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** membatasi request pada pool, **agar** kapasitas relay dan gateway tidak terlampaui.

### Konteks

Admission harus mengikat pemilihan relay dengan reservasi agar request serentak tidak memakai slot yang sama.

- [PRD](../PRD.md): C-4.
- Kualitas terkait: Q-3, Q-5.
- [ROADMAP](../ROADMAP.md): M2.1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-18. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** mapping fixed memiliki relay eligible, **When** request di-admit, **Then** relay utama dipilih dan reservasi gateway/relay tercatat sebagai satu keputusan atomik.
- **AC2:** **Given** request serentak mencapai batas kapasitas yang dikonfigurasi, **When** request tambahan datang, **Then** request ditolak dengan gateway_capacity_exceeded sebelum dispatch.
- **AC3:** **Given** anggaran total buffer telah habis meskipun body individual valid, **When** request tambahan datang, **Then** admission menolak beban yang tidak dapat ditampung tanpa pertumbuhan buffer tak terbatas.
- **AC4 (negatif/edge):** **Given** request selesai atau gagal, **When** lifecycle ditutup, **Then** reservasi dilepas tepat sekali dan kapasitas dapat digunakan kembali.

### Batas dan dependensi

- Di luar scope: Rebinding strict otomatis, pergantian upstream/key, rotasi WARP otomatis, serta controller P1.
- Diblokir oleh: [EP-02-S05](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s05), [EP-01-T06](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-03-s02"></a>

## [STORY] EP-03-S02 Tentukan eligibility dari health dan mode operasi yang terpisah

**Status:** draft · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** membedakan health dan mode relay, **agar** routing mengikuti kondisi yang benar.

### Konteks

Relay hidup, relay sehat, dan relay diaktifkan operator adalah kondisi yang berbeda.

- [PRD](../PRD.md): C-4, C-9.
- Kualitas terkait: Q-2, Q-9.
- [ROADMAP](../ROADMAP.md): M2.1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-16. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** relay baru tersedia, **When** probe terjadwal berlangsung, **Then** status memisahkan keterjangkauan SOCKS5, jalur HTTPS, dan ketersediaan upstream/model.
- **AC2:** **Given** ambang gagal atau sukses yang dikonfigurasi tercapai, **When** hasil probe diperbarui, **Then** health/circuit berubah menurut kebijakan tanpa mengabaikan mode operasi.
- **AC3:** **Given** relay draining atau disabled kembali lolos probe, **When** request baru datang, **Then** relay tetap tidak menerima assignment baru.
- **AC4 (negatif/edge):** **Given** semua kandidat tidak eligible, **When** client meminta generasi, **Then** gateway mengembalikan 503 no_eligible_relay tanpa koneksi langsung.

### Batas dan dependensi

- Di luar scope: Rebinding strict otomatis, pergantian upstream/key, rotasi WARP otomatis, serta controller P1.
- Diblokir oleh: [EP-03-S01](#ep-03-s01), [EP-02-T06](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-t06)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-03-s03"></a>

## [STORY] EP-03-S03 Sebarkan request stateless ke relay dengan inflight terendah

**Status:** draft · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** menyeimbangkan request model stateless, **agar** relay yang tersedia dapat dimanfaatkan sesuai kapasitas.

### Konteks

Least-inflight hanya aman untuk model yang dinyatakan stateless dan tidak memakai strict affinity.

- [PRD](../PRD.md): C-4.
- Kualitas terkait: Q-3, Q-5.
- [ROADMAP](../ROADMAP.md): M2.1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** model stateless tervalidasi memakai least-inflight, **When** request di-admit, **Then** relay eligible dengan inflight paling sedikit dipilih bersama reservasi.
- **AC2:** **Given** beberapa relay memiliki inflight yang sama, **When** request berulang dijalankan, **Then** tie-break mengikuti urutan stabil yang didokumentasikan.
- **AC3:** **Given** model memakai strict atau belum dinyatakan stateless, **When** konfigurasi mencoba least-inflight, **Then** validasi menolak kombinasi tersebut.
- **AC4 (negatif/edge):** **Given** relay termurah sedang cooldown atau penuh, **When** request di-admit, **Then** relay itu tidak dipilih hanya karena counter inflight lebih rendah.

### Batas dan dependensi

- Di luar scope: Rebinding strict otomatis, pergantian upstream/key, rotasi WARP otomatis, serta controller P1.
- Diblokir oleh: [EP-03-S02](#ep-03-s02)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-03-s04"></a>

## [STORY] EP-03-S04 Hindari duplikasi generasi saat failover

**Status:** draft · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** menghindari pengulangan generasi yang ambigu, **agar** kegagalan transport tidak menggandakan pekerjaan dan biaya.

### Konteks

POST yang mungkin sudah diproses tidak boleh dikirim ulang hanya karena koneksi gagal.

- [PRD](../PRD.md): C-4, C-5.
- Kualitas terkait: Q-3.
- [ROADMAP](../ROADMAP.md): M2.1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-07, AC-08. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** mapping non-strict gagal sebelum request aplikasi terkirim dan bukti tersedia, **When** fallback eligible ada, **Then** gateway melakukan paling banyak satu fallback dengan total maksimal dua attempt.
- **AC2:** **Given** dispatch POST ambigu atau timeout menunggu header, **When** request gagal, **Then** tidak ada generasi otomatis kedua di relay mana pun.
- **AC3:** **Given** JSON atau SSE sudah mulai diteruskan, **When** jalur gagal, **Then** gateway tidak memindahkan stream ke relay lain.
- **AC4 (negatif/edge):** **Given** transport tidak menyediakan bukti belum terkirim atau client memiliki retry sendiri, **When** attempt dievaluasi, **Then** gateway menggunakan satu attempt pada kondisi ambigu dan jumlah attempt gateway dapat ditelusuri.

### Batas dan dependensi

- Di luar scope: Rebinding strict otomatis, pergantian upstream/key, rotasi WARP otomatis, serta controller P1.
- Diblokir oleh: [EP-03-S02](#ep-03-s02), [EP-01-T03](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-03-s05"></a>

## [STORY] EP-03-S05 Kenali batas upstream tanpa merotasi relay secara sembarang

**Status:** draft · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** mengetahui penyebab batas provider, **agar** tindakan pemulihan sesuai masalah sebenarnya.

### Konteks

Auth, akses, quota, timeout, dan gangguan jaringan memerlukan tindakan yang berbeda.

- [PRD](../PRD.md): C-5, C-9.
- Kualitas terkait: Q-3, Q-9.
- [ROADMAP](../ROADMAP.md): M2.1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-09. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** upstream memberi 429 dengan Retry-After valid, **When** gateway menerima respons, **Then** status/body serta Retry-After diteruskan dan cooldown diterapkan pada scope yang diketahui.
- **AC2:** **Given** scope 429 tidak diketahui, **When** gateway menentukan cooldown, **Then** pembatasan diperlakukan konservatif sebagai upstream/credential-wide.
- **AC3:** **Given** upstream memberi 400/401/403/404/422 atau 5xx, **When** request selesai, **Then** kategori tercatat tanpa rotasi relay, key, atau generasi ulang otomatis.
- **AC4 (negatif/edge):** **Given** satu upstream/model sedang cooldown, **When** request untuk scope lain diproses, **Then** relay tidak dianggap rusak global hanya karena cooldown tersebut.

### Batas dan dependensi

- Di luar scope: Rebinding strict otomatis, pergantian upstream/key, rotasi WARP otomatis, serta controller P1.
- Diblokir oleh: [EP-03-S04](#ep-03-s04)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.

<a id="ep-03-s06"></a>

## [STORY] EP-03-S06 Batalkan pekerjaan upstream dan lepaskan resource tepat waktu

**Status:** draft · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** membatalkan stream yang tidak lagi diperlukan, **agar** pekerjaan upstream dan kapasitas gateway segera dilepas.

### Konteks

Stream panjang dan client lambat tidak boleh menahan slot atau menumbuhkan buffer tanpa batas.

- [PRD](../PRD.md): C-2, C-4, C-5.
- Kualitas terkait: Q-1, Q-5, Q-6.
- [ROADMAP](../ROADMAP.md): M2.1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-12, AC-18. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** disconnect client telah terdeteksi, **When** gateway menangani pembatalan, **Then** pekerjaan upstream dibatalkan dalam ≤2 detik dan slotnya dilepas.
- **AC2:** **Given** header, idle stream, atau total request mencapai timeout tersendiri, **When** deadline terkait terlewati, **Then** request dihentikan sesuai kategori timeout tanpa menunggu deadline lain.
- **AC3:** **Given** client membaca lebih lambat dari upstream, **When** stream berlangsung, **Then** backpressure dan buffer tetap berada dalam anggaran yang ditetapkan EP-01-T06.
- **AC4 (negatif/edge):** **Given** EOF, error, timeout, dan disconnect bersaing, **When** finalisasi berlangsung, **Then** slot dan koneksi tidak bocor atau dilepas dua kali.

### Batas dan dependensi

- Di luar scope: Rebinding strict otomatis, pergantian upstream/key, rotasi WARP otomatis, serta controller P1.
- Diblokir oleh: [EP-03-S01](#ep-03-s01), [EP-03-S04](#ep-03-s04)
- Acuan keputusan dan penjabaran: **(agent's call)** mengikuti bagian Keputusan dan kesiapan pada epic ini.
