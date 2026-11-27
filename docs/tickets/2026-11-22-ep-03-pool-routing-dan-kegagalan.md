---
title: "Request tetap terkendali ketika kondisi relay berubah"
date: 2026-11-22
version: 3
status: approved
approved_by: pengguna
approved_at: 2026-11-23
planner: none
project: ""
hierarchy: epic
epic_id: EP-03
milestone: "M2.1"
priority: P0
capabilities: ["C-2", "C-4", "C-5", "C-9"]
---

# EP-03 Request tetap terkendali ketika kondisi relay berubah

**Milestone:** M2.1 · **Prioritas:** P0 sesuai pembagian docs · **Status:** approved · **Pemilik:** pengguna · **Tanggal:** TBD

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
- Pembagian epic, ID lokal, urutan/dependensi, serta penjabaran AC berasal dari usulan agent dan telah disetujui pengguna pada 23 November 2026 ([D-015](../DECISIONS.md#d-015-persetujuan-seluruh-tiket-mvp-dan-p1)).
- Nilai/kontrak yang belum ditetapkan menjadi hasil task pendahulu. Story yang bergantung padanya belum siap diimplementasikan sampai hasil task tersedia.
- Kebutuhan P1 menunggu penerimaan MVP dan keputusan prioritas O-9; task bersama EP-07-T01 tidak mewajibkan Responses API dikerjakan lebih dahulu.
- Persetujuan mencakup epic dan seluruh child story/task pada file ini. Status `approved` menyatakan persetujuan tiket; implementasi, pemenuhan dependensi, dan kelulusan AC tetap memerlukan bukti. Tidak ada tiket yang dibuat pada planner eksternal.

<a id="ep-03-s01"></a>

## [STORY] EP-03-S01 Kirim request hanya ketika relay dan kapasitas tersedia

**Status:** approved · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

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
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 26 November 2026

[D-026](../DECISIONS.md#d-026-admission-fixed-independen-ep-03-s01) menyetujui komponen dan fixture lokal independen. [Kontrak dan bukti admission](../admission-fixed.md) memisahkan persiapan bounded dari commit kapasitas atomik. Status `approved` tetap persetujuan tiket; penerimaan produk serta dependensi S05/T06/M0 tetap terbuka.

- **AC1 parsial produk:** primary fixed fixture yang tersedia di-admit bersama slot gateway/relay/outbound dan stream deklaratif dalam satu transaksi. Tidak ada klaim configured relay sudah sehat; eligibility health/mode/cooldown terpadu belum dibuktikan.
- **AC2 parsial produk:** batas global, relay dan outbound menghasilkan `429 gateway_capacity_exceeded`; fixture dua HTTPS/SOCKS5 membuktikan nol tambahan koneksi/CONNECT/POST pada request ditolak. Relay lain tetap dapat dipakai jika batas global tersedia, tanpa fallback tersembunyi.
- **AC3 parsial produk:** pending dan byte direservasi sebelum alokasi/read; budget body/stream agregat, exact cap, known/unknown body dan pemulihan kapasitas dibuktikan. `pending + active` dibatasi bersama; tidak ada antrean generasi baru atau klaim RSS/performa.
- **AC4 parsial produk:** release pending/admitted idempotent; JSON/SSE EOF/error dan Close/cancel konkuren tidak membebaskan tail sebelum forwarding/caller selesai. Kapasitas pulih dan jaringan ditutup tepat sekali; target cancellation/performa S06 tetap terpisah.
- Tidak menambah dependency/schema/default numerik, owner health, transport/listener/deployment, provider/WARP nyata, least-inflight/fallback/cooldown, binding atau dashboard. SSE tak terduga tetap dapat ditolak sesudah satu dispatch menurut D-023.

<a id="ep-03-s02"></a>

## [STORY] EP-03-S02 Tentukan eligibility dari health dan mode operasi yang terpisah

**Status:** approved · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

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
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 27 November 2026

[D-027](../DECISIONS.md#d-027-eligibility-health-dan-mode-independen-ep-03-s02) menyetujui komponen dan fixture lokal independen. [Kontrak dan bukti eligibility](../eligibility-relay.md) memakai satu owner untuk health/circuit/mode/admission, scheduler bounded, dan handler model/admin dari snapshot yang sama. Status `approved` tetap persetujuan tiket; penerimaan produk S01/T06/M0 tetap parsial.

- **AC1 parsial produk:** scheduler fixture mengumpulkan SOCKS5 dan HTTPS keluar secara terpisah melalui jaringan lokal. Availability upstream/model dicatat per scope dan tetap unknown tanpa sumber aman; tidak ada probe generasi berbayar atau klaim provider tersedia.
- **AC2 parsial produk:** threshold 3 gagal/2 sukses, replay/order, expiry/recovery, jitter/backoff/timeout dan hasil lama diuji dengan waktu virtual serta fixture. Mode tidak diubah oleh hasil probe; angka fixture bukan default produksi.
- **AC3 parsial produk:** draining/disabled tetap menolak assignment sesudah probe pulih. SSE admitted berlanjut sampai DONE dan resource dipulihkan, tanpa menjalankan maintenance atau shutdown penuh.
- **AC4 parsial produk:** primary default/nondefault unknown atau disabled menghasilkan `503 no_eligible_relay`, dengan nol tambahan dial/CONNECT/POST/direct. Cap probe/generasi bersama tetap menghasilkan 429; failure model primary tidak merusak scope secondary atau readiness transport.
- Snapshot/redaksi/budget maksimum relay+model, linearization, release tepat sekali dan cancellation worker dibuktikan secara lokal. Hasil command dan batas pengujian ada pada dokumen kontrak; integrasi provider/WARP, RSS/performa, cooldown/binding serta gerbang produk tetap terbuka.

<a id="ep-03-s03"></a>

## [STORY] EP-03-S03 Sebarkan request stateless ke relay dengan inflight terendah

**Status:** approved · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

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
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 27 November 2026

[D-028](../DECISIONS.md#d-028-least-inflight-stateless-dan-cooldown-minimum-ep-03-s03) menyetujui komponen dan fixture lokal independen, termasuk cooldown minimum agar seluruh AC4 terbukti. [Kontrak least-inflight dan hasil tes](../least-inflight.md) tersedia. Status approved tetap persetujuan tiket; penerimaan produk S02/S01/T06/M0 tetap parsial.

- **AC1 parsial produk:** kandidat allowlist upstream dipilih menurut total generasi inflight per relay bersama reservasi atomik. ID pilihan dipakai pada dispatch; dua SOCKS5 menuju authority TLS yang sama membuktikan stream aktif menahan slot dan request berikutnya memakai relay lain.
- **AC2 parsial produk:** tie memilih anggota pertama dalam urutan allowlist, termasuk urutan warp-2 sebelum warp-1 dan pengulangan serial setelah release. Tidak menjanjikan round-robin/fairness serial.
- **AC3 parsial produk:** least_inflight hanya stateless true/affinity none; missing/false/strict/boolean ambigu dan mode unknown ditolak. Fixed tetap default; metadata config/catalog disalin. Provider/model nyata tidak dinyatakan stateless dari fixture.
- **AC4 parsial produk:** kandidat cooldown maupun full-low dilewati, deadline exact expiry pulih, dan probe sukses tidak menghapus cooldown. Semua state-ineligible memberi 503; kandidat state-eligible penuh memberi 429, tanpa dispatch tambahan. Cooldown minimum relay-wide eksplisit tidak menafsirkan 429/Retry-After S05.
- Konkurensi/release, budget, metadata teredaksi, dan batas fixture dibuktikan pada command di dokumen kontrak. Tidak ada runtime/deployment/provider/WARP/native, benchmark/RSS, retry S04 atau scope 429 pada S05 baru. Contoh default tetap fixed.

<a id="ep-03-s04"></a>

## [STORY] EP-03-S04 Hindari duplikasi generasi saat failover

**Status:** approved · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

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
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 27 November 2026

Kontrak dan bukti lokal tercatat pada [failover.md](../failover.md). Coordinator hanya mengulang dispatch setelah error eksplisit `RequestNotSentError`; perpindahan lease ke fallback tervalidasi dan atomik, dengan satu fallback maksimum.

- **AC1 parsial produk:** tes unit membuktikan fallback non-strict hanya dengan bukti no-send dan batas dua dispatch. Fixture HTTP/SOCKS5 membuktikan refusal primary sebelum tunnel, satu dispatch melalui fallback, dan nol direct dial.
- **AC2 parsial produk:** error generik, bukti no-send false, dan timeout tetap satu dispatch. Fixture gateway mengirim POST sampai body terbaca upstream, memutus koneksi sebelum header, dan membuktikan relay fallback tidak menerima dispatch.
- **AC3 parsial produk:** jalur coordinator tidak mengulang setelah menyerahkan response; tes gateway JSON/SSE membuktikan stream terputus tidak diganti.
- **AC4 parsial produk:** tes/ledger membedakan dispatch upstream dari CONNECT serta membuktikan satu attempt tanpa bukti no-send. Tidak ada log atau metrik produk baru.
- Adapter jaringan produksi belum tersedia. Penerbitan bukti no-send oleh adapter runtime, retry provider/client nyata, WARP, performa, dan penerimaan produk tetap belum dibuktikan.

<a id="ep-03-s05"></a>

## [STORY] EP-03-S05 Kenali batas upstream tanpa merotasi relay secara sembarang

**Status:** approved · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

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
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 27 November 2026

Komponen lokal meneruskan error upstream, menyimpan kategori/status teredaksi, dan menerapkan cooldown upstream-wide untuk scope tidak diketahui atau model-scoped hanya dari sinyal server tepercaya. Cooldown scoped tidak mengubah health/readiness relay; error upstream tidak memicu retry generasi. `Retry-After` malformed/overflow dihilangkan; tanggal valid kedaluwarsa diteruskan tanpa cooldown aktif.

- **AC1–AC2 parsial produk:** tes pool/gateway membuktikan cooldown model/upstream, expiry, scope isolasi, dan satu dispatch; tes integrasi membuktikan status/body/header upstream 429 dipertahankan. Sumber bukti scope provider nyata belum tersedia; gateway memakai scope konservatif upstream-wide.
- **AC3 parsial produk:** metadata kategori/status teredaksi diuji untuk 400/401/403/404/422/5xx; integrasi membuktikan respons dipertahankan tanpa retry. Runtime provider/WARP belum diuji.
- **AC4 parsial produk:** tes pool membuktikan cooldown satu model tidak menghalangi alias lain pada relay yang sama dan tidak mengubah readiness/health relay.
- `go test ./...`, race test paket terkait, `go vet ./...`, dan pemeriksaan `gofmt` lulus. Penerimaan runtime/provider dan gerbang M0 tetap terbuka.

<a id="ep-03-s06"></a>

## [STORY] EP-03-S06 Batalkan pekerjaan upstream dan lepaskan resource tepat waktu

**Status:** approved · **Parent:** EP-03 · **Milestone:** M2.1 · **Prioritas:** P0

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
- Acuan keputusan dan penjabaran: mengikuti bagian Keputusan dan kesiapan pada epic ini; persetujuan tercatat pada D-015.

### Hasil pengerjaan 27 November 2026

- **AC1 parsial fixture:** `TestGatewayStreamCancellationProfile` menjalankan 30 pembatalan saat menunggu header dan 30 saat SSE aktif. Tes mengukur deteksi disconnect → pembatalan context upstream (≤2 detik) terpisah dari finalisasi handler; setelah finalisasi, slot berkapasitas satu dapat dipakai ulang.
- **AC2 parsial fixture:** tes timeout header/idle/total yang sudah ada tetap menguji batas secara terpisah. Disconnect setelah stream dikomit tetap menghasilkan truncation tanpa envelope/event sintetis sesuai [kontrak streaming](../streaming-sse.md).
- **AC3 parsial fixture:** tes backpressure dan budget stream 8 MiB tercatat pada [kontrak streaming](../streaming-sse.md); hasil ini tidak mengukur RSS atau batas kernel/adapter runtime.
- **AC4 parsial fixture:** 20 stream pada dua relay dibatalkan serentak; fixture mengamati cancellation upstream dan handler selesai, lalu 20 admission berurutan berhasil. Tes kapasitas/Close/cancel dan pelepasan idempotent S01 menjadi bukti komplementer; drain controller dan D-014 tidak diubah.
- Verifikasi: `go test -count=1 -timeout=180s ./...`, race test integrasi terfokus untuk dua tes cancellation, `go vet ./...`, dan `git diff --check` lulus. Pengujian hanya memakai fixture loopback; transport runtime, provider/WARP, RSS/performa, dan gerbang produk/M0 belum dibuktikan.
