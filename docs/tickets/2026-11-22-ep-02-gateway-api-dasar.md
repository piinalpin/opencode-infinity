---
title: "Agent memakai kontrak provider melalui satu gateway"
date: 2026-11-22
version: 5
status: approved
approved_by: pengguna
approved_at: 2026-11-23
planner: none
project: ""
hierarchy: epic
epic_id: EP-02
milestone: "M1"
priority: P0
capabilities: ["C-1", "C-2", "C-3", "C-5", "C-8", "C-9", "C-10"]
---

# EP-02 Agent memakai kontrak provider melalui satu gateway

**Milestone:** M1 · **Prioritas:** P0 sesuai pembagian docs · **Status:** approved · **Pemilik:** pengguna · **Tanggal:** TBD

## Ringkasan

Gateway satu relay melayani katalog model, JSON, serta streaming sesuai kontrak dan batas akses.

## Masalah

Client memerlukan endpoint provider biasa dengan credential terpisah dan transport yang transparan.

## Konteks sumber

- [PRD](../PRD.md), kapabilitas C-1, C-2, C-3, C-5, C-8, C-9, C-10.
- [ARCHITECTURE](../ARCHITECTURE.md), batas modul, state, transport, dan lifecycle.
- [ROADMAP](../ROADMAP.md), milestone M1.
- [TECHSTACK](../TECHSTACK.md), kandidat implementasi; pilihan dan versi ditetapkan pada M0.
- [Catatan teknis](../PRD-technical-notes.md), AC-03, AC-04, AC-05, AC-06, AC-10, AC-11.

## Pengguna dan ukuran keberhasilan

- Developer memakai API model; operator memelihara deployment dan memantau pool.
- AC-03–AC-06, AC-10, dan AC-11 lulus pada gateway.
- Kontrak default upstream dan provider spesifik memberikan perilaku yang konsisten.
- Baseline yang belum terukur tetap TBD; metode dan pelaksana pengukuran mengikuti D-005 (angka aktual menyusul M0).

## Scope

- Registry dan validasi konfigurasi
- Auth serta pemetaan model/upstream
- JSON/SSE dan tool calling
- Target, error, dan health dasar

## Di luar scope

Routing pool dinamis, lifecycle binding per percakapan, dashboard operasional lengkap, dan Responses API.

## Kondisi yang menggagalkan penerimaan

- Skenario wajib dihilangkan untuk menandai pekerjaan selesai.
- Secret/konten percakapan bocor, generasi diulang pada dispatch ambigu, atau target yang belum terpenuhi tetap dinyatakan lulus.

## Story dan task

| ID lokal | Hasil | Jenis | Jumlah AC | Diblokir oleh |
|---|---|---|---|---|
| [EP-02-T01](#ep-02-t01) | Tolak konfigurasi yang tidak dapat dilayani | Task | 4 | [EP-01-T06](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06) |
| [EP-02-S02](#ep-02-s02) | Akses model dengan token gateway tanpa membawa secret upstream | Story | 5 | [EP-02-T01](#ep-02-t01) |
| [EP-02-S03](#ep-02-s03) | Pilih alias model dan terima respons JSON dari upstream yang benar | Story | 4 | [EP-02-S02](#ep-02-s02) |
| [EP-02-S04](#ep-02-s04) | Terima streaming content dan tool calls tanpa menunggu respons lengkap | Story | 4 | [EP-02-S03](#ep-02-s03), [EP-01-T03](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03) |
| [EP-02-S05](#ep-02-s05) | Kenali error dan batas request sebelum dispatch | Story | 4 | [EP-02-S03](#ep-02-s03) |
| [EP-02-T06](#ep-02-t06) | Bedakan proses hidup dari kesiapan melayani request | Task | 4 | [EP-02-S05](#ep-02-s05) |

## Constraints dan dependensi

- Cakupan dan angka target mengikuti PRD; angka usulan belum merupakan hasil benchmark.
- Tidak ada assignee, sprint, estimasi, atau tanggal yang ditebak.
- Dependensi eksternal terkait: [EP-01](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md). Ikuti ID child yang tepat pada tabel; tidak semua dependensi memerlukan seluruh epic lain selesai.

## Risiko

- Validasi yang menghapus field tambahan merusak kompatibilitas; gunakan fixture protokol.
- Header atau redirect dapat mengubah tujuan; tolak perubahan origin dan verifikasi credential tidak bocor.

## Keputusan dan kesiapan

- Satu file untuk satu epic dengan banyak story/task adalah arahan pengguna.
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC berasal dari usulan agent dan telah disetujui pengguna pada 23 November 2026 ([D-015](../DECISIONS.md#d-015-persetujuan-seluruh-tiket-mvp-dan-p1)).
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan mencakup epic dan seluruh child story/task pada file ini. Status `approved` menyatakan persetujuan tiket; implementasi, pemenuhan dependensi, dan kelulusan AC tetap memerlukan bukti. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-02-t01"></a>

## [TASK] EP-02-T01 Tolak konfigurasi yang tidak dapat dilayani

**Status:** approved · **Parent:** EP-02 · **Milestone:** M1 · **Prioritas:** P0

### Konteks

Registry yang tidak valid harus gagal sebelum menerima trafik atau memilih target yang salah.

- [PRD](../PRD.md): C-3, C-8, C-10.
- Kualitas terkait: Q-7, Q-10.
- [ROADMAP](../ROADMAP.md): M1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** konfigurasi valid dan secret tersedia, **When** gateway dimulai, **Then** registry memuat alias, upstream, relay, default upstream, serta batas yang dipilih.
- **AC2:** **Given** alias ganda, referensi relay hilang, default upstream ambigu, atau URL upstream tidak valid, **When** startup dijalankan, **Then** gateway menolak konfigurasi sebelum melayani generasi.
- **AC3:** **Given** secret kosong atau masih placeholder, **When** startup dijalankan, **Then** validasi menolak konfigurasi tanpa menampilkan nilai secret.
- **AC4 (negatif/edge):** **Given** operator mengganti konfigurasi, **When** restart terkontrol dilakukan, **Then** konfigurasi baru diterapkan sebagai satu versi tanpa hot reload parsial.

### Batas dan dependensi

- Di luar scope: Routing pool dinamis, lifecycle binding per percakapan, dashboard operasional lengkap, dan Responses API.
- Diblokir oleh: [EP-01-T06](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 24 November 2026

Persetujuan [D-020](../DECISIONS.md#d-020-konfigurasi-dan-registry-offline-ep-02-t01) mengizinkan konfigurasi/registry offline independen. [Kontrak dan bukti](../konfigurasi.md) serta [contoh YAML](../../examples/infinity.yaml) tersedia. Dependensi EP-01-T06 dan M0 tetap belum terpenuhi; status persetujuan tiket tidak berarti implementasi terintegrasi selesai.

- **AC1 parsial:** pemuatan registry/default/batas/secret dan lookup alias terbukti pada modul; startup gateway belum dibuktikan.
- **AC2 parsial:** schema, duplikasi/referensi/default/URL invalid ditolak tanpa hasil parsial; wiring sebelum listener/generasi menunggu aplikasi server.
- **AC3 ✅ (validasi modul):** secret kosong/placeholder/tidak tersedia/oversize dan role overlap ditolak tanpa nilai pada error atau fmt/JSON/YAML.
- **AC4 parsial:** snapshot/load baru immutable dan kegagalan tidak mengubah versi lama; restart/drain proses terintegrasi belum dibuktikan.
- Tidak ada endpoint generasi/admin, probe/generasi provider, perubahan WARP/deployment, atau klaim M0/performa/enforcement runtime lulus.

<a id="ep-02-s02"></a>

## [STORY] EP-02-S02 Akses model dengan token gateway tanpa membawa secret upstream

**Status:** approved · **Parent:** EP-02 · **Milestone:** M1 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** mengakses model memakai token gateway, **agar** credential provider cukup dikelola operator.

### Konteks

Token client, admin, dan upstream harus mempunyai peran terpisah.

- [PRD](../PRD.md): C-8.
- Kualitas terkait: Q-7, Q-8.
- [ROADMAP](../ROADMAP.md): M1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-10. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** token gateway valid, **When** client meminta katalog atau generasi, **Then** request diizinkan dengan credential upstream dari server bila diperlukan.
- **AC2:** **Given** token gateway tidak valid atau tidak ada, **When** client meminta endpoint model, **Then** gateway mengembalikan 401 tanpa menghubungi upstream.
- **AC3:** **Given** client hanya mempunyai token generasi, **When** client mencoba endpoint admin, **Then** akses operasional admin ditolak.
- **AC4:** **Given** client non-strict sudah memakai endpoint dan token gateway, **When** operator mengganti credential upstream lalu menerapkan perubahan lewat restart terkontrol, **Then** request berikutnya memakai credential baru tanpa perubahan konfigurasi provider atau token gateway pada client; kelanjutan sesi strict mengikuti kontrak restart pada EP-04.
- **AC5 (negatif/edge):** **Given** credential server digunakan untuk dispatch, **When** respons dan log diperiksa, **Then** token client tidak diteruskan sebagai credential upstream dan secret tidak bocor.

### Batas dan dependensi

- Di luar scope: Routing pool dinamis, lifecycle binding per percakapan, dashboard operasional lengkap, dan Responses API.
- Diblokir oleh: [EP-02-T01](#ep-02-t01)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 24 November 2026

[D-021](../DECISIONS.md#d-021-kontrol-token-dan-credential-independen-ep-02-s02) mengizinkan kontrol akses/credential dan fixture independen. [Kontrak dan bukti](../akses-dan-credential.md) tersedia. Seluruh AC berikut masih memerlukan penerimaan gateway terintegrasi; blocker T01/T06/M0 tetap terbuka.

- **AC1 parsial:** guard membuka empat path model pada harness; request outbound fixture memakai key server sesuai default/provider. Katalog/generasi produk belum tersedia.
- **AC2 parsial:** token tidak valid/tidak ada, role salah, atau header ambigu menghasilkan 401; nol handler/dispatch/CONNECT/upstream terbukti pada komponen/fixture; wiring seluruh endpoint produk belum dibuktikan.
- **AC3 parsial:** gateway token tidak membuka fixture status/metrics/operasi admin; listener/admin browser produk belum dibuktikan.
- **AC4 parsial:** snapshot/instance baru memakai key baru dengan token dan request client tetap; bukan restart/drain proses.
- **AC5 parsial:** credential outbound tepat, token client tidak diteruskan sebagai key, dan output/error/log milik modul teredaksi. Body upstream tidak di-rewrite; alur respons/error generasi produk belum tersedia.
- Tidak ada server/transport runtime, model mapping JSON/SSE, perubahan deployment, provider/WARP nyata atau klaim M0/performa lulus.

<a id="ep-02-s03"></a>

## [STORY] EP-02-S03 Pilih alias model dan terima respons JSON dari upstream yang benar

**Status:** approved · **Parent:** EP-02 · **Milestone:** M1 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** memilih model alias dan menerima hasil JSON, **agar** konfigurasi provider tetap terpusat.

### Konteks

Client harus memakai katalog lokal dan alias tanpa mengetahui detail relay/provider.

- [PRD](../PRD.md): C-1, C-2, C-3.
- Kualitas terkait: Q-1, Q-7.
- [ROADMAP](../ROADMAP.md): M1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-04. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** alias terdaftar pada mapping, **When** client meminta katalog dengan token valid, **Then** katalog memuat model alias sesuai scope default atau provider spesifik.
- **AC2:** **Given** alias menunjuk satu upstream dan model nyata, **When** client mengirim generasi non-stream, **Then** upstream yang dikonfigurasi menerima model nyata dan client menerima payload JSON beserta usage yang tersedia.
- **AC3:** **Given** request memuat field tambahan yang didukung upstream, **When** generasi diteruskan, **Then** field tersebut mempertahankan semantik setelah mapping model.
- **AC4 (negatif/edge):** **Given** alias tidak ditemukan, **When** client mengirim generasi, **Then** gateway mengembalikan 404 model_not_found tanpa dispatch.

### Batas dan dependensi

- Di luar scope: Routing pool dinamis, lifecycle binding per percakapan, dashboard operasional lengkap, dan Responses API.
- Diblokir oleh: [EP-02-S02](#ep-02-s02)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 25 November 2026

[D-022](../DECISIONS.md#d-022-katalog-alias-dan-json-independen-ep-02-s03) mengizinkan komponen/handler dan fixture independen. [Kontrak dan bukti](../model-alias-dan-json.md) tersedia. Penerimaan gateway terintegrasi dan blocker S02/T01/T06/M0 tetap terbuka; hasil berikut adalah bukti komponen/fixture.

- **AC1 parsial produk:** katalog default/provider menampilkan alias dalam scope, tanpa lookup provider/jaringan atau pengungkapan model nyata/key/relay. Empat route memasang guard S02; penolakan token terbukti sebelum body read dan CONNECT.
- **AC2 parsial produk:** dua upstream HTTPS sintetis menerima model nyata, prefix, key dan relay yang benar. Status/content type/body/usage dipertahankan; usage absent tetap absent. Tidak ada request provider/WARP nyata.
- **AC3 parsial produk:** hanya nilai top-level model dipetakan; nested field/tools, Unicode, null, angka besar/eksponen dan duplicate extension keys mempertahankan byte. Kontrol model/stream ambigu ditolak.
- **AC4 parsial produk:** alias unknown menghasilkan `404 model_not_found` dan nol CONNECT/dispatch. Provider unknown menghasilkan `404 upstream_not_found` tanpa default fallback.
- Admission statis membatasi request, buffer, slot relay dan dispatch outbound; EOF/error/cancel menutup jaringan, sementara lease tetap hidup sampai forwarding/caller selesai dan body ditutup. Ini bukan health eligibility, cooldown, binding atau routing dinamis.
- Efek jaringan memakai dispatcher tepercaya yang diinjeksi. TLS/DNS proxy/no redirect/no direct fallback/no replay ambigu/cancellation terbukti pada wiring fixture, bukan transport runtime. SSE S04, kontrak error S05 lengkap, listener/deployment, restart, performa dan M0 belum diluluskan.

<a id="ep-02-s04"></a>

## [STORY] EP-02-S04 Terima streaming content dan tool calls tanpa menunggu respons lengkap

**Status:** approved · **Parent:** EP-02 · **Milestone:** M1 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** menerima output dan tool calls secara streaming, **agar** agent dapat melanjutkan pekerjaan tanpa buffering seluruh jawaban.

### Konteks

Agent membutuhkan token dan tool-call deltas bertahap, termasuk ketika jaringan memecah event menjadi fragmen.

- [PRD](../PRD.md): C-1, C-2.
- Kualitas terkait: Q-1, Q-6.
- [ROADMAP](../ROADMAP.md): M1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-03. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** upstream mengirim event SSE pertama, **When** gateway meneruskannya, **Then** client dapat menerima event tanpa menunggu event selanjutnya atau stream selesai.
- **AC2:** **Given** fixture memecah event pada batas byte berbeda, **When** client membaca content dan tool-call deltas, **Then** urutan dan makna deltas tetap sama.
- **AC3:** **Given** upstream mengirim usage atau field tambahan pada stream, **When** gateway meneruskan event, **Then** informasi tersebut tetap tersedia tanpa perubahan semantik.
- **AC4 (negatif/edge):** **Given** upstream putus setelah stream mulai, **When** client membaca respons, **Then** gateway tidak menyisipkan generasi lain atau penanda selesai sukses palsu.

### Batas dan dependensi

- Di luar scope: Routing pool dinamis, lifecycle binding per percakapan, dashboard operasional lengkap, dan Responses API.
- Diblokir oleh: [EP-02-S03](#ep-02-s03), [EP-01-T03](2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t03)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-02-s05"></a>

## [STORY] EP-02-S05 Kenali error dan batas request sebelum dispatch

**Status:** approved · **Parent:** EP-02 · **Milestone:** M1 · **Prioritas:** P0

**Sebagai** developer, **saya ingin** mendapat error yang dapat ditelusuri, **agar** penyebab request ditolak tidak tertukar dengan gangguan provider.

### Konteks

Gateway perlu menjaga target, ukuran request, dan perbedaan error tanpa meneruskan secret ke origin lain.

- [PRD](../PRD.md): C-5, C-8.
- Kualitas terkait: Q-2, Q-7, Q-9.
- [ROADMAP](../ROADMAP.md): M1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

- [AC sumber](../PRD-technical-notes.md): AC-05, AC-06, AC-11. Referensi ini menunjukkan area pembuktian, bukan klaim sudah lulus.

### Acceptance criteria

- **AC1:** **Given** request berukuran valid ditujukan ke mapping server, **When** request dikirim, **Then** fixture menunjukkan SOCKS5 terpilih dan hostname tujuan melalui proxy.
- **AC2:** **Given** client mencoba target bebas atau upstream memberi redirect, **When** request diproses, **Then** gateway menolak pengalihan tujuan dan tidak mengirim credential ke origin lain.
- **AC3:** **Given** JSON invalid, encoding tidak didukung, atau body melewati batas termasuk setelah decoding, **When** request diperiksa, **Then** gateway memberi error terstruktur sebelum dispatch.
- **AC4 (negatif/edge):** **Given** gateway gagal sebelum stream atau upstream mengembalikan error, **When** respons dibaca, **Then** error gateway memiliki request ID dan error source; error upstream mempertahankan status/body sesuai kontrak.

### Batas dan dependensi

- Di luar scope: Routing pool dinamis, lifecycle binding per percakapan, dashboard operasional lengkap, dan Responses API.
- Diblokir oleh: [EP-02-S03](#ep-02-s03)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

<a id="ep-02-t06"></a>

## [TASK] EP-02-T06 Bedakan proses hidup dari kesiapan melayani request

**Status:** approved · **Parent:** EP-02 · **Milestone:** M1 · **Prioritas:** P0

### Konteks

Operator membutuhkan health dasar yang tidak menghabiskan kuota atau menyamakan proses hidup dengan relay siap.

- [PRD](../PRD.md): C-9, C-10.
- Kualitas terkait: Q-9, Q-10.
- [ROADMAP](../ROADMAP.md): M1; [ARCHITECTURE](../ARCHITECTURE.md): tanggung jawab dan batas sistem tetap berlaku.

### Acceptance criteria

- **AC1:** **Given** proses gateway berfungsi, **When** liveness diperiksa, **Then** hasil tidak memicu generasi berbayar ke upstream.
- **AC2:** **Given** konfigurasi valid dan relay untuk konfigurasi default eligible, **When** readiness diperiksa, **Then** kesiapan dilaporkan sesuai scope yang terdokumentasi.
- **AC3:** **Given** relay tidak tersedia sementara proses tetap berjalan, **When** health diperiksa, **Then** readiness tidak menyatakan siap dan liveness tidak otomatis gagal.
- **AC4 (negatif/edge):** **Given** kondisi proses berubah, **When** health dan status diperiksa, **Then** metadata versi serta waktu kondisi dapat ditelusuri tanpa secret.

### Batas dan dependensi

- Di luar scope: Routing pool dinamis, lifecycle binding per percakapan, dashboard operasional lengkap, dan Responses API.
- Diblokir oleh: [EP-02-S05](#ep-02-s05)
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.
