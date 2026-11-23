# Dokumentasi opencode-infinity

Dokumentasi produk, arsitektur, kandidat stack, roadmap, dan backlog gateway coding agent melalui relay pool WARP. Web UI pemantauan pool termasuk MVP.

## Urutan baca

Aturan kontribusi dan guardrail agent tersedia di [AGENTS.md](../AGENTS.md).

| Dokumen | Isi | Status acuan |
|---|---|---|
| [PRD](PRD.md) | Masalah, pengguna, cakupan, kapabilitas, target kualitas, dan keputusan terbuka | Draft 0.1.3 |
| [DECISIONS](DECISIONS.md) | Keputusan eksplisit pengguna dan bagian pertanyaan yang masih terbuka | Catatan 0.1.1; D-001–D-004 terkonfirmasi |
| [ARCHITECTURE](ARCHITECTURE.md) | Komponen, alur, state, lifecycle, dan [struktur project](ARCHITECTURE.md#13-struktur-project-yang-disarankan), termasuk MicroWARP | Draft 0.1.3 |
| [TECHSTACK](TECHSTACK.md) | Kandidat bahasa/library/tooling dan rencana validasi versi | Kandidat 0.1.1; belum pin final |
| [ROADMAP](ROADMAP.md) | Milestone M0–M3, dependensi, serta gerbang MVP/P1 | Draft 0.1.0 |
| [Catatan teknis](PRD-technical-notes.md) | Kontrak awal serta AC-01–AC-22 | Pendamping 0.1.2; bagian historis dipertahankan |

Urutan desain adalah PRD → ARCHITECTURE → TECHSTACK → implementasi. ROADMAP mengatur urutan pengerjaan. Jika isi bertentangan, selesaikan kebutuhan di PRD dan batas desain di ARCHITECTURE sebelum mengubah kandidat stack atau tiket.

## Indeks epic MVP

Satu file memuat satu epic beserta seluruh child story/task. Pemecahan dan dependensi merupakan **(agent's call)**; seluruh tiket berstatus `draft`.

| Epic | Hasil | Milestone | Story | Task | AC child |
|---|---|---|---|---|---|
| [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md) | Gateway layak dibangun pada lingkungan target | M0 | 0 | 6 | 25 |
| [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md) | Agent memakai kontrak provider melalui satu gateway | M1 | 4 | 2 | 25 |
| [EP-03](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md) | Request tetap terkendali ketika kondisi relay berubah | M2.1 | 6 | 0 | 24 |
| [EP-04](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md) | Percakapan mempertahankan jalur saat relay dipelihara | M2.2 | 4 | 2 | 24 |
| [EP-05](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md) | Operator memantau relay pool dari dashboard privat | M2.3 | 4 | 2 | 24 |
| [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md) | MVP dapat dipakai dan dioperasikan pada deployment yang teruji | M3 | 2 | 3 | 22 |

## Indeks epic P1

P1 dimulai setelah penerimaan MVP melalui [EP-06-T05](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t05). Prioritas lintas P1 ditetapkan oleh [EP-07-T01](tickets/2026-11-22-ep-07-responses-api.md#ep-07-t01); nomor epic bukan urutan implementasi. Epic lain hanya bergantung pada task prioritas itu, bukan pada penyelesaian Responses API.

| Epic | Hasil | Milestone | Story | Task | AC child |
|---|---|---|---|---|---|
| [EP-07](tickets/2026-11-22-ep-07-responses-api.md) | Agent dapat memakai Responses API yang diuji | P1 | 1 | 3 | 16 |
| [EP-08](tickets/2026-11-22-ep-08-integrasi-relay-http-lama.md) | Operator mempertahankan stack relay HTTP yang sudah ada | P1 | 1 | 2 | 12 |
| [EP-09](tickets/2026-11-22-ep-09-controller-maintenance-warp.md) | Maintenance WARP berjalan otomatis tanpa melanggar binding | P1 | 2 | 1 | 12 |
| [EP-10](tickets/2026-11-22-ep-10-plugin-session-id.md) | Client memasok identitas sesi dinamis untuk affinity | P1 | 1 | 2 | 12 |
| [EP-11](tickets/2026-11-22-ep-11-export-config-dan-secret-store.md) | Client dikonfigurasi melalui export dan secret store OS | P1 | 2 | 1 | 12 |

## Cara memakai tiket

1. Buka epic menurut milestone dan pilih ID child pada tabel Story dan task.
2. Ikuti dependensi child yang tertulis. M2.2 dan M2.3 dapat berjalan bersamaan setelah kontrak state/snapshot tersedia.
3. Selesaikan task keputusan sebelum story yang menggunakannya, terutama versi, sesi strict, shutdown, dan kesegaran UI.
4. Tinjau draft beserta AC-nya sebelum implementasi; status `approved` hanya diberikan setelah persetujuan eksplisit atas draft tersebut.
5. Simpan bukti hasil AC pada pekerjaan implementasi dan verifikasi kembali keseluruhan kontrak saat M3.

ID `EP-xx` dan `EP-xx-Sxx/Txx` adalah ID lokal, bukan key planner. `AC1` di dalam child adalah kriteria tiket; `AC-01` adalah skenario penerimaan sumber. Masing-masing child mempunyai tautan ke PRD, kapabilitas, quality bars, scope, dan blocker. File dapat dipakai sebagai draft lokal atau diimpor ke planner dengan pemetaan hierarki epic/child.

Tanggal, assignee, sprint, dan estimasi belum ditetapkan. Versi teknologi di TECHSTACK masih kandidat; pemilihan serta pin dilakukan melalui [EP-01-T02](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t02), lalu ditinjau kembali menjelang rilis. Hot reload tetap TBD dan tidak mendapat tiket implementasi tanpa keputusan cakupan.

## Pemetaan kapabilitas PRD

Pemetaan berikut menampilkan pekerjaan khusus per kapabilitas. Task prioritas lintas P1 dan verifikasi rilis menyeluruh ditautkan pada bagian keputusan serta acceptance.

| Kapabilitas | Epic terkait |
|---|---|
| C-1 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md) |
| C-2 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md), [EP-03](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md) |
| C-3 | [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md), [EP-11](tickets/2026-11-22-ep-11-export-config-dan-secret-store.md) |
| C-4 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-03](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md) |
| C-5 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md), [EP-03](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md), [EP-05](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md) |
| C-6 | [EP-04](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md), [EP-10](tickets/2026-11-22-ep-10-plugin-session-id.md) |
| C-7 | [EP-04](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md), [EP-09](tickets/2026-11-22-ep-09-controller-maintenance-warp.md) |
| C-8 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md), [EP-05](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md), [EP-11](tickets/2026-11-22-ep-11-export-config-dan-secret-store.md) |
| C-9 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md), [EP-03](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md), [EP-05](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md), [EP-09](tickets/2026-11-22-ep-09-controller-maintenance-warp.md) |
| C-10 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](tickets/2026-11-22-ep-02-gateway-api-dasar.md), [EP-04](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md), [EP-05](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md) |
| C-11 | [EP-07](tickets/2026-11-22-ep-07-responses-api.md) |
| C-12 | [EP-08](tickets/2026-11-22-ep-08-integrasi-relay-http-lama.md) |
| C-13 | [EP-09](tickets/2026-11-22-ep-09-controller-maintenance-warp.md) |
| C-14 | [EP-01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-05](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md), [EP-06](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md) |
| C-15 | [EP-10](tickets/2026-11-22-ep-10-plugin-session-id.md) |
| C-16 | [EP-11](tickets/2026-11-22-ep-11-export-config-dan-secret-store.md) |

## Pemetaan acceptance sumber

Tautan berikut menunjukkan tiket pembuktian terkait. [EP-06-T05](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t05) memverifikasi ulang AC-01–AC-22 pada artefak rilis yang sama; kelulusan spike bukan pengganti penerimaan produk.

| AC sumber | Story/task terkait |
|---|---|
| AC-01 | [EP-06-S01](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-s01) |
| AC-02 | [EP-06-S02](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-s02) |
| AC-03 | [EP-01-T03](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03), [EP-02-S04](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s04) |
| AC-04 | [EP-02-S03](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s03) |
| AC-05 | [EP-01-T03](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03), [EP-01-T04](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t04), [EP-02-S05](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s05) |
| AC-06 | [EP-01-T03](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03), [EP-01-T04](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t04), [EP-02-S05](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s05) |
| AC-07 | [EP-01-T03](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03), [EP-03-S04](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s04) |
| AC-08 | [EP-01-T03](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03), [EP-03-S04](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s04) |
| AC-09 | [EP-03-S05](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s05) |
| AC-10 | [EP-02-S02](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s02) |
| AC-11 | [EP-02-S05](tickets/2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s05) |
| AC-12 | [EP-03-S06](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s06) |
| AC-13 | [EP-04-T01](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-t01), [EP-04-S02](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s02), [EP-04-S03](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s03) |
| AC-14 | [EP-04-T01](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-t01), [EP-04-S03](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s03) |
| AC-15 | [EP-04-S04](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s04) |
| AC-16 | [EP-03-S02](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02) |
| AC-17 | [EP-05-S03](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-s03) |
| AC-18 | [EP-03-S01](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s01), [EP-03-S06](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s06) |
| AC-19 | [EP-06-T03](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t03) |
| AC-20 | [EP-05-T01](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t01), [EP-05-S03](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-s03), [EP-05-T06](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06) |
| AC-21 | [EP-05-T01](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t01), [EP-05-S04](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-s04), [EP-05-T06](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06) |
| AC-22 | [EP-05-T01](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t01), [EP-05-S02](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-s02), [EP-05-S05](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-s05), [EP-05-T06](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t06) |

## Task keputusan terbuka

| Pertanyaan PRD | Penanggung jawab lokal |
|---|---|
| O-1 | [EP-01-T01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t01) |
| O-2 | [EP-01-T01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t01) |
| O-3 | [EP-01-T01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t01) |
| O-4 | [EP-04-T01](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-t01) |
| O-5 | [EP-04-T05](tickets/2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-t05) |
| O-6 | [EP-01-T06](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06), [EP-06-T03](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t03) |
| O-7 | [EP-06-T05](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t05) |
| O-8 | [EP-06-T05](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t05) |
| O-9 | [EP-07-T01](tickets/2026-11-22-ep-07-responses-api.md#ep-07-t01) |
| O-10 | [EP-05-T01](tickets/2026-11-22-ep-05-dashboard-dan-observabilitas.md#ep-05-t01) |

Pemilik keputusan produk dan target adalah pengguna sendiri sesuai D-004; pelaksana pengukuran teknis serta peran lain yang belum dipilih tetap TBD. O-1 tidak menahan pembuatan fixture, O-7 hanya memblokir klaim SLA bila diperlukan, dan O-9 tidak menahan penerimaan MVP.

## Ringkasan backlog

**11 epic, 27 story, 24 task, dan 208 AC child.** Enam epic mencakup MVP dan lima epic mencakup P1. Seluruh C-1–C-16 dan AC-01–AC-22 mempunyai tautan pekerjaan; tidak ada progres implementasi atau persetujuan draft yang diasumsikan.
