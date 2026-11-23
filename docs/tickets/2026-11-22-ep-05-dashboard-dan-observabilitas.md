---
title: "Operator memantau relay pool dari dashboard privat"
date: 2026-11-22
version: 2
status: approved
approved_by: pengguna
approved_at: 2026-11-23
planner: none
project: ""
hierarchy: epic
epic_id: EP-05
milestone: "M2.3"
priority: P0
capabilities: ["C-5", "C-8", "C-9", "C-10", "C-14"]
---

# EP-05 Operator memantau relay pool dari dashboard privat

**Milestone:** M2.3 · **Prioritas:** P0 sesuai pembagian docs · **Status:** approved · **Pemilik:** pengguna · **Tanggal:** TBD

## Ringkasan

Dashboard terlindungi menampilkan kondisi seluruh pool, memperbarui data, dan menandai kegagalan pembaruan tanpa membocorkan konten.

## Masalah

Status relay, kapasitas, IP teramati, dan penyebab kegagalan perlu dapat ditinjau dari browser dengan sumber state yang sama seperti router.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-5, C-8, C-9, C-10, C-14.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone M2.3.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), AC-17, AC-20, AC-21, AC-22.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- AC-17 dan AC-20–AC-22 lulus pada browser dengan fixture state yang berubah.
- G-3/G-6/G-7 dapat dinilai dari state dan metadata yang konsisten.
- Baseline yang belum terukur tetap TBD; metode dan pelaksana pengukuran mengikuti D-005 (angka aktual menyusul M0).

## Scope

- Kontrak snapshot dan akses browser
- Ringkasan pool serta detail relay
- Pembaruan otomatis dan penanda stale
- Log/metrik metadata dan packaging UI

## Di luar scope

Kontrol container dari UI, editor konfigurasi, billing, retensi riwayat percakapan, dan platform observabilitas wajib terpisah.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-05-T01](#ep-05-t01) | Tetapkan kontrak monitoring dan akses dashboard | Task | 4 | [EP-03-S02](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02), [EP-01-T02](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t02) |
| [EP-05-S02](#ep-05-s02) | Buka dashboard hanya dengan otorisasi admin | Story | 4 | [EP-05-T01](#ep-05-t01), [EP-02-S02](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s02) |
| [EP-05-S03](#ep-05-s03) | Lihat ringkasan pool dan kondisi setiap relay | Story | 4 | [EP-05-S02](#ep-05-s02), [EP-03-S05](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s05) |
| [EP-05-S04](#ep-05-s04) | Kenali perubahan state dan data dashboard yang sudah lama | Story | 4 | [EP-05-S03](#ep-05-s03) |
| [EP-05-S05](#ep-05-s05) | Telusuri kegagalan lewat metadata dan metrik yang teredaksi | Story | 4 | [EP-05-T01](#ep-05-t01), [EP-03-S05](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s05) |
| [EP-05-T06](#ep-05-t06) | Sajikan dashboard terintegrasi pada artefak gateway | Task | 4 | [EP-05-S04](#ep-05-s04), [EP-05-S05](#ep-05-s05), [EP-04-S04](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s04), [EP-01-T05](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t05) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-01](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md), [EP-02](2026-11-22-ep-02-gateway-api-dasar.md), [EP-03](2026-11-22-ep-03-pool-routing-dan-kegagalan.md), [EP-04](2026-11-22-ep-04-affinity-dan-maintenance.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Polling sukses dapat menyamarkan probe lama; pisahkan waktu snapshot, probe, dan pembaruan UI.
- Snapshot atau label metrik dapat membocorkan identitas; terapkan redaksi di server.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC berasal dari usulan agent dan telah disetujui pengguna pada 23 November 2026 ([D-015](../DECISIONS.md#d-015-persetujuan-seluruh-tiket-mvp-dan-p1)).
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan mencakup epic dan seluruh child story/task pada file ini. Status `approved` menyatakan persetujuan tiket; implementasi, pemenuhan dependensi, dan kelulusan AC tetap memerlukan bukti. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-05-t01"></a>

## [TASK] EP-05-T01 Tetapkan kontrak monitoring dan akses dashboard

**Status:** approved · **Parent:** EP-05 · **Milestone:** M2.3 · **Prioritas:** P0

### Konteks

UI memerlukan kontrak data, auth, dan target kesegaran yang pasti sebelum implementasi perilaku browser.

- [PRD](../PRD.md): C-8, C-9, C-14.
- Kualitas terkait: Q-7, Q-8, Q-9, Q-11.
- [ROADMAP](../ROADMAP.md): M2.3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-20, AC-21, AC-22. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

- Keputusan yang ditangani task ini: O-10, A-2, A-5, A-7.

D-009 menetapkan polling 5 detik dengan stale >15 detik untuk MVP; push SSE/WebSocket menjadi kandidat P1. Task ini merinci cara ukur dan kontrak snapshot, bukan memilih target baru.

### Acceptance criteria

- **AC1:** **Given** state pool telah didefinisikan, **When** kontrak snapshot ditetapkan, **Then** field, satuan, nilai belum tersedia, scope cooldown, serta waktu snapshot/probe/pembaruan dijelaskan.
- **AC2:** **Given** target pembaruan sudah ditetapkan D-009 (polling 5 detik, stale >15 detik), **When** kontrak snapshot ditetapkan, **Then** interval, ambang stale, penandaan kegagalan, dan cara ukurnya dijelaskan secara numerik.
- **AC3:** **Given** browser mengakses listener admin, **When** kontrak auth dibakukan, **Then** otorisasi, sesi/TTL/expiry, logout, perlindungan request browser, dan akses data operasional ditetapkan.
- **AC4 (negatif/edge):** **Given** target kesegaran sudah ditetapkan D-009, **When** kontrak ditutup, **Then** implementasi tidak menambah endpoint push atau mengubah interval tanpa keputusan baru, dan kandidat P1 tidak dianggap disetujui.

### Batas dan dependensi

- Di luar scope: Kontrol container dari UI, editor konfigurasi, billing, retensi riwayat percakapan, dan platform observabilitas wajib terpisah.
- Diblokir oleh: [EP-03-S02](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s02), [EP-01-T02](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t02)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-05-s02"></a>

## [STORY] EP-05-S02 Buka dashboard hanya dengan otorisasi admin

**Status:** approved · **Parent:** EP-05 · **Milestone:** M2.3 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** mengakses dashboard dengan hak admin, **agar** informasi operasional tidak terbuka untuk client generasi.

### Konteks

Akses jaringan privat tidak menggantikan otorisasi dan token generasi tidak memberi hak operasional.

- [PRD](../PRD.md): C-8, C-14.
- Kualitas terkait: Q-7, Q-8, Q-11.
- [ROADMAP](../ROADMAP.md): M2.3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-22. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** operator mempunyai otorisasi admin yang valid, **When** dashboard dibuka, **Then** operator dapat membaca data sesuai kontrak EP-05-T01.
- **AC2:** **Given** otorisasi hilang, salah, kedaluwarsa, atau sudah logout, **When** UI atau endpoint data diakses, **Then** data operasional tidak diberikan.
- **AC3:** **Given** client hanya membawa token gateway, **When** data dashboard diminta, **Then** akses tetap ditolak.
- **AC4 (negatif/edge):** **Given** dashboard digunakan, **When** URL, storage, payload pemantauan, dan tampilan diperiksa, **Then** token admin, credential upstream, prompt/response, body tool, dan raw session ID tidak muncul dalam URL, storage client, snapshot, atau tampilan pemantauan.

### Batas dan dependensi

- Di luar scope: Kontrol container dari UI, editor konfigurasi, billing, retensi riwayat percakapan, dan platform observabilitas wajib terpisah.
- Diblokir oleh: [EP-05-T01](#ep-05-t01), [EP-02-S02](2026-11-22-ep-02-gateway-api-dasar.md#ep-02-s02)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-05-s03"></a>

## [STORY] EP-05-S03 Lihat ringkasan pool dan kondisi setiap relay

**Status:** approved · **Parent:** EP-05 · **Milestone:** M2.3 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** melihat seluruh pool dan detail relay, **agar** jalur yang bermasalah dapat dikenali dari browser.

### Konteks

Operator perlu memahami kondisi nyata pool dan keterbatasan IP dari satu tampilan yang konsisten dengan router.

- [PRD](../PRD.md): C-9, C-14.
- Kualitas terkait: Q-9, Q-11.
- [ROADMAP](../ROADMAP.md): M2.3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-17, AC-20. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** pool memuat relay pada state berbeda, **When** operator membuka dashboard, **Then** seluruh relay terkonfigurasi dan ringkasan jumlah menurut state ditampilkan sesuai snapshot.
- **AC2:** **Given** relay memiliki request aktif, kapasitas, cooldown, dan alasan tidak eligible, **When** detail relay ditinjau, **Then** nilai serta scope yang ditampilkan sama dengan sumber status gateway.
- **AC3:** **Given** probe menghasilkan IP sama antar-relay atau IPv6 tidak tersedia, **When** dashboard diperbarui, **Then** duplikasi atau ketidaktersediaan ditampilkan tanpa janji IP unik atau kerusakan otomatis.
- **AC4 (negatif/edge):** **Given** probe belum memberikan hasil, **When** operator membaca status, **Then** nilai belum tersedia dibedakan dari nol, sehat, atau hasil observasi terbaru.

### Batas dan dependensi

- Di luar scope: Kontrol container dari UI, editor konfigurasi, billing, retensi riwayat percakapan, dan platform observabilitas wajib terpisah.
- Diblokir oleh: [EP-05-S02](#ep-05-s02), [EP-03-S05](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s05)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-05-s04"></a>

## [STORY] EP-05-S04 Kenali perubahan state dan data dashboard yang sudah lama

**Status:** approved · **Parent:** EP-05 · **Milestone:** M2.3 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** mengenali kesegaran informasi pool, **agar** keputusan tidak dibuat dari data lama yang terlihat baru.

### Konteks

Data lama tidak boleh terlihat sebagai pemeriksaan terbaru saat koneksi monitoring gagal.

- [PRD](../PRD.md): C-9, C-14.
- Kualitas terkait: Q-11.
- [ROADMAP](../ROADMAP.md): M2.3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-21. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** state gateway berubah setelah tampilan awal, **When** siklus pembaruan berjalan, **Then** perubahan tercermin otomatis dalam target numerik EP-05-T01.
- **AC2:** **Given** pengambilan snapshot gagal, **When** dashboard mencoba memperbarui, **Then** UI menandai kegagalan dan mempertahankan waktu pembaruan terakhir tanpa menyatakan data lama masih terbaru.
- **AC3:** **Given** snapshot baru memuat probe lama, **When** UI menerima snapshot, **Then** waktu snapshot tidak menggantikan waktu probe relay.
- **AC4 (negatif/edge):** **Given** operator membuka beberapa tab atau refresh berulang, **When** permintaan monitoring bertambah, **Then** tidak ada generasi atau probe upstream tambahan per refresh dan pekerjaan pembaruan tetap dibatasi.

### Batas dan dependensi

- Di luar scope: Kontrol container dari UI, editor konfigurasi, billing, retensi riwayat percakapan, dan platform observabilitas wajib terpisah.
- Diblokir oleh: [EP-05-S03](#ep-05-s03)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-05-s05"></a>

## [STORY] EP-05-S05 Telusuri kegagalan lewat metadata dan metrik yang teredaksi

**Status:** approved · **Parent:** EP-05 · **Milestone:** M2.3 · **Prioritas:** P0

**Sebagai** operator, **saya ingin** menelusuri kegagalan melalui metadata, **agar** penyebab gangguan dapat dipahami tanpa merekam percakapan.

### Konteks

Operator membutuhkan diagnosis tanpa menyimpan konten percakapan atau membuat label metrik tak terbatas.

- [PRD](../PRD.md): C-5, C-9.
- Kualitas terkait: Q-8, Q-9, Q-10.
- [ROADMAP](../ROADMAP.md): M2.3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-22. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** request selesai atau gagal, **When** operator meninjau log, **Then** request ID, upstream, relay, attempt, kategori, dan durasi dapat ditelusuri tanpa body, secret, authorization, atau raw session ID.
- **AC2:** **Given** trafik dan cooldown berubah, **When** metrik dibaca dengan otorisasi admin, **Then** counter, inflight, ketersediaan, dan latensi mengikuti sumber state yang relevan.
- **AC3:** **Given** banyak request atau session ID berbeda dipakai, **When** label metrik diperiksa, **Then** request/session ID tidak membentuk seri label baru.
- **AC4 (negatif/edge):** **Given** upstream tidak menyediakan usage atau latency belum tersedia, **When** status dan metrik ditampilkan, **Then** ketidaktersediaan tidak diubah menjadi angka nol atau estimasi billing.

### Batas dan dependensi

- Di luar scope: Kontrol container dari UI, editor konfigurasi, billing, retensi riwayat percakapan, dan platform observabilitas wajib terpisah.
- Diblokir oleh: [EP-05-T01](#ep-05-t01), [EP-03-S05](2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s05)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-05-t06"></a>

## [TASK] EP-05-T06 Sajikan dashboard terintegrasi pada artefak gateway

**Status:** approved · **Parent:** EP-05 · **Milestone:** M2.3 · **Prioritas:** P0

### Konteks

Dashboard MVP harus dapat dipasang bersama aplikasi dan diuji terhadap state nyata gateway.

- [PRD](../PRD.md): C-10, C-14.
- Kualitas terkait: Q-5, Q-7, Q-10, Q-11.
- [ROADMAP](../ROADMAP.md): M2.3; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-20, AC-21, AC-22. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** UI dan API admin selesai, **When** artefak aplikasi dibangun, **Then** dashboard serta asetnya dapat dilayani tanpa service frontend produksi terpisah.
- **AC2:** **Given** fixture menjalankan relay sehat/gagal/drain/cooldown, **When** suite browser berjalan, **Then** tampilan dan pembaruan cocok dengan data sumber.
- **AC3:** **Given** monitoring lambat atau beberapa browser membaca status, **When** generasi berjalan bersamaan, **Then** snapshot/telemetry tidak menahan dispatch dan beban tetap dalam anggaran yang diputuskan.
- **AC4 (negatif/edge):** **Given** artefak dipasang pada konfigurasi default, **When** listener dan payload diperiksa, **Then** admin tetap privat dan proteksi AC-22 berlaku juga pada endpoint data langsung.

### Batas dan dependensi

- Di luar scope: Kontrol container dari UI, editor konfigurasi, billing, retensi riwayat percakapan, dan platform observabilitas wajib terpisah.
- Diblokir oleh: [EP-05-S04](#ep-05-s04), [EP-05-S05](#ep-05-s05), [EP-04-S04](2026-11-22-ep-04-affinity-dan-maintenance.md#ep-04-s04), [EP-01-T05](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t05)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.
