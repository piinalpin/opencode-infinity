# M0 — Gerbang dan rencana pengukuran MVP

Tanggal peninjauan: 24 November 2026, Asia/Jakarta

Task: [EP-01-T06](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t06)

Status: **hasil dokumentasi parsial; gerbang M0 belum lulus**.

Keputusan: [D-019 — Profil resource dan metode pengukuran provisional](DECISIONS.md#d-019-profil-resource-dan-metode-pengukuran-provisional).

Pengguna menyetujui pengerjaan dokumentasi independen ketika T04 belum selesai, lalu menyetujui profil dan metode pada laporan ini. Persetujuan tersebut tidak menghapus dependensi T06 terhadap T04/T05, tidak mengesahkan seluruh draft arsitektur, dan tidak membuktikan hasil benchmark. Tidak ada kode runtime, konfigurasi deployment, request provider, registrasi WARP, atau pengukuran performa yang dijalankan dalam T06.

## 1. Penilaian gerbang M0

[PRD](PRD.md#9-tahapan-dan-gerbang-penerimaan) dan [ROADMAP](ROADMAP.md#3-m0-validasi-arsitektur-dan-kelayakan) mensyaratkan request nyata melalui WARP, fixture jalur tanpa direct fallback, kesesuaian client/model awal, serta penyelesaian hambatan kelayakan. [D-005](DECISIONS.md#d-005-baseline-dua-tahap-dan-pelaksanaan-pengukuran-g-1g-7) juga meminta baseline aktual sebelum M1. Status sekarang **belum lulus**, dengan alasan berikut:

| Input/hasil | Bukti yang tersedia | Kekurangan dan implikasi |
|---|---|---|
| T01 lingkungan/integrasi | [Matriks T01](M0-lingkungan-dan-integrasi.md): CachyOS/Linux x86_64, target image `linux/amd64`, Docker 29.9.3/Compose 5.7.1, Desktop dan CLI bawaan 2.3.0; OpenCode Free/model `deepseek-v4.1-flash-free` dipilih D-016 | Exact version runtime provider aktif, akses Free melalui gateway, JSON/non-stream ketika `forceStream` aktif, listener/tailnet final dan baseline aktual masih terbuka; versi client bukan versi SDK |
| T02 stack/provenance | [Laporan T02](M0-stack-dan-versi.md) dan [pin/lock/checksum](m0/ep-01-t02/): Go 1.27.4, Node 26.13.1/npm 12.2.2, Svelte 5.59.1, Vite 8.3.8, TypeScript 6.0.3; commit/lisensi MicroWARP dan digest base/builder terpin | Hasil kompatibilitas stack independen tidak menutup T01 atau membuktikan transport; digest base bukan identitas image produk final |
| T03 fixture transport | [Laporan T03](M0-fixture-transport.md): SOCKS5 domain/relay/peer, TLS, fragmentasi/flushing SSE, fase dispatch, cancellation, kontrol direct yang ditolak oracle; race 50 pengulangan pada hasil historis | Bukti client kontrol sintetis, bukan gateway/SDK/client nyata atau WARP; helper dibatasi 16 koneksi, body/frame 64 KiB, dan deadline socket 5 detik sehingga belum siap untuk benchmark 20 stream × 30 menit |
| T04 WARP/target nyata | [Laporan T04](M0-warp-request.md), [checks](m0/ep-01-t04/checks.json), dan [provenance](m0/ep-01-t04/provenance.json): image relay fixture serta kontrol lokal tersedia | Native preflight ditolak `tools_disabled`; nol registrasi WARP dan nol generasi provider. AC T04 dan AC1 T06 belum penuh; langkah native berikutnya masih menunggu revisi yang disetujui pada pekerjaan T04 |
| T05 packaging | [Laporan T05](M0-ui-packaging.md) dan [provenance](m0/ep-01-t05/provenance.json): UI Svelte tertanam dalam binary Go, browser Chromium 159.0.8154.6/Playwright 1.65.1 dan image Distroless amd64 diuji; runtime satu proses tanpa toolchain/browser | Packaging independen tidak membuktikan snapshot pool, sesi admin produksi atau dashboard AC-20–AC-22; batas fixture bukan konfigurasi/performa produk |

Laporan sumber mencatat tanggal dan lingkup pemeriksaannya masing-masing. T06 meninjau bukti tersebut, bukan memeriksa ulang versi atau menjalankan ulang spike. Hasil T02 sebelum T05 yang menyebut browser belum diuji tetap benar dalam konteks tanggalnya.

Penutupan M0 memerlukan bukti T04 aktual yang memenuhi AC-nya, pemenuhan input T01 dan baseline D-005, lalu peninjauan ulang seluruh hasil spike beserta keputusan kelayakan pada T06. Jika model/protokol/akses ternyata tidak sesuai, pengguna menentukan tindak lanjut di DECISIONS; tidak ada penggantian model, pelonggaran semantik JSON/SSE, atau pemindahan P1 ke MVP secara otomatis. Laporan yang lengkap secara struktur tidak mengubah gerbang menjadi lulus.

## 2. Target dan invariant yang tetap berlaku

| Acuan | Target/kontrak |
|---|---|
| Q-4, D-008 | p95 tambahan ≤50 ms dibanding jalur SOCKS5 yang sama tanpa gateway, memakai upstream lokal dan SOCKS5 fixture |
| Q-5, D-008 | 20 stream bersamaan selama 30 menit; RSS proses gateway ≤256 MiB; kelompok server dibatasi 2 vCPU/2 GiB; kapasitas fixture 5 relay × 5 slot |
| Q-6, D-014 | Cancellation upstream ≤2 detik setelah disconnect terdeteksi; timeout header/idle/total terpisah; drain 60 detik hanya menolak admission baru; stream aktif mengikuti batas keras 30 menit, catatan pemutusan dan exit eksplisit; grace container >30 menit |
| Q-7 | Body individual maksimum 10 MiB termasuk setelah decoding; token gateway/admin/upstream terpisah; target dari konfigurasi, TLS hostname aktif, redirect tidak diikuti |
| D-009 | Polling snapshot 5 detik, stale setelah >15 detik tanpa sukses; kegagalan dan waktu sukses terakhir terlihat; refresh/multitab tidak memicu probe atau generasi |
| D-013 | Fixed routing default; strict lifecycle dibuktikan dengan fixture sesi eksplisit; tidak ada klaim strict affinity per percakapan untuk client nyata pada MVP |
| Q-3, Q-8, Q-10 | Tidak mengulang POST ambigu/stream yang sudah diteruskan; fallback non-strict hanya dengan bukti belum terkirim, maksimal dua attempt total; bukti/log teredaksi dan data belum tersedia tidak menjadi nol |

Sumber target: [PRD bagian kualitas](PRD.md#8-standar-kualitas) dan [DECISIONS](DECISIONS.md). Target provisional D-008 tidak diturunkan oleh profil baru; hasil aktual dan pengesahan kelayakannya tetap memerlukan pengukuran.

## 3. Profil resource provisional

Profil ini dikonfirmasi melalui D-019 sebagai anggaran awal implementasi dan pengujian. Ini **bukan default produk final atau bukti bahwa RSS tercapai**. Schema konfigurasi, accounting alokasi, dan penolakan pada batas harus dibuktikan pada task pemilik. Batas byte dan jumlah berlaku bersamaan.

| Bucket | Batas profil | Yang dihitung | Pemilik alokasi dan pelepasan |
|---|---:|---|---|
| Body request agregat | 64 MiB | Seluruh representasi body aktif, salinan decode/JSON/remap, dan scratch selama masih hidup; bukan hanya body asli | Validasi/admission sebelum alokasi, coordinator hingga dispatch/selesai/error/cancel; EP-02-T01, EP-02-S04/S05, EP-03-S01/S06 |
| Buffer stream agregat | 8 MiB | Buffer copy/framing yang diperlukan; tidak menampung seluruh respons atau tool-call output | Transport/coordinator menerapkan backpressure dan melepas buffer tepat sekali; EP-02-S04, EP-03-S06 |
| State operasional | 16 MiB | Binding, sesi admin, cooldown, indeks/map/string dan metadata, termasuk overhead yang harus diukur | Pool/access melalui operasi pemilik state; EP-03-S01, EP-04-T01, EP-05-T01 |
| Monitoring | 8 MiB | Salinan snapshot/serialisasi, probe, telemetry serta buffer log yang dibatasi; tanpa riwayat konten | Health/status/telemetry, terpisah dari jalur dispatch; EP-03-S02, EP-05-T01/S05 |
| Subtotal terkelola | 96 MiB | 64 + 8 + 16 + 8; salinan hanya dibebankan pada satu bucket tetapi semua salinan serentak harus masuk accounting | Pembuktian agregat dan cleanup lintas modul pada EP-06-T03 |
| Ruang terhadap target RSS | 160 MiB | 256 − 96 untuk runtime Go, slack heap/GC, stack/goroutine, TLS/HTTP/transport/koneksi dan alokasi di luar bucket di atas | Ukur RSS nyata; ruang ini tidak merupakan batas alokasi yang otomatis ditegakkan atau jaminan RSS |

| Batas jumlah/ukuran | Profil dan alasan | Task verifikasi |
|---|---|---|
| Request generasi aktif | 20 global, tetap aktif sampai stream selesai; tidak ada antrean generasi tambahan | EP-03-S01/S06 |
| Kapasitas relay | 5 relay × 5 slot = 25; lima alias fixed, setiap alias primary pada relay berbeda dan membawa 4 stream = 20; headroom relay 5 slot tidak menaikkan admission global | EP-03-S01, EP-06-T03 |
| Koneksi masuk | 64 total, termasuk admin, pembacaan header/body, aktif dan idle; bukan 64 request generasi | EP-02-T01/S05, EP-03-S01 |
| Koneksi keluar | 32 total, termasuk aktif, idle dan probe; limit probe tidak boleh menambah koneksi di luar cap ini | EP-03-S02/S06 |
| Binding | 1.024, tetap dibatasi bucket state; binding aktif tidak digusur untuk menerima binding baru | EP-04-T01/S02 |
| Sesi admin | 32, tetap dibatasi bucket state; bukan penetapan TTL atau kontrak autentikasi produksi | EP-05-T01/S02 |
| Probe | 2 paralel; timeout/backoff ditetapkan terpisah; tidak ada probe generasi berbayar otomatis | EP-03-S02 |
| Pembacaan status | 4 paralel; proses dan buffer setiap pembaca dibatasi agar snapshot lambat tidak menahan dispatch | EP-05-T01/T06 |
| Snapshot teredaksi | Maksimal 256 KiB; 4 salinan serentak = 1 MiB di dalam bucket monitoring, bukan tambahan di luar 8 MiB | EP-05-T01/T06; bentuk snapshot/config yang dapat dilayani divalidasi sebelum dinyatakan siap; jangan memangkas relay diam-diam |

Batas individual body tetap 10 MiB. Secara aritmetika, 20 × 10 MiB = 200 MiB dan 25 × 10 MiB = 250 MiB sebelum salinan decode, runtime, TLS, state atau monitoring. Profil tidak menjanjikan 20 body maksimum dapat dibuffer serentak. Ketika cap agregat habis, admission membatasi alokasi dan menolak request sebelum dispatch, meskipun slot lain tersedia. Jangan menurunkan batas individual diam-diam atau mengantre body tanpa batas. Semua reservasi byte/slot/koneksi dilepas tepat sekali pada sukses, error, timeout dan disconnect; state aktif mengikuti kontrak binding/sesi yang ditetapkan task pemilik.

Satu alias fixed dengan primary berkapasitas lima tidak dapat memakai kapasitas 25 secara otomatis. Distribusi lima alias di atas adalah konfigurasi fixture deterministik; tidak mengganti model upstream nyata atau menyatakan model D-016 stateless. Benchmark least-inflight, jika dijalankan, memakai model fixture yang tervalidasi dan hasilnya dicatat terpisah. Dependensi EP-06-T03 terhadap EP-03-S03 tetap berlaku.

## 4. Metode pengukuran Q-4–Q-6

Metode berikut adalah rencana D-019 untuk [EP-06-T03](tickets/2026-11-22-ep-06-integrasi-dan-rilis-mvp.md#ep-06-t03), bukan command atau benchmark yang sudah tersedia/dijalankan. Fixture benchmark harus mendukung batas koneksi, body dan durasi yang diperlukan tanpa melemahkan guard helper T03; pengembangannya merupakan pekerjaan task benchmark tersebut.

### Q-4: overhead berpasangan

Kontrol mengikuti client → SOCKS5 → upstream fixture, sedangkan perlakuan mengikuti client → gateway → SOCKS5 yang sama → upstream fixture yang sama. Ukur TTFB dari awal request hingga byte respons pertama dengan clock monotonic pada observer client yang sama. Body sintetis profil utama berukuran 64 KiB. Payload/respons sintetis, fragmentasi, protokol, TLS, kebijakan koneksi/keepalive, beban, dan jadwal probe harus sebanding serta dicatat; jangan mencampur profil cold/warm tanpa label.

Jalankan tiga run independen. Setiap run memiliki 100 pasangan warm-up, lalu 1.000 pasangan yang diukur. Dalam setiap pasangan kedua jalur dijalankan tanpa tumpang tindih; urutan AB dan BA bergantian agar urutan tidak selalu menguntungkan satu jalur. Warm-up dipisahkan dari statistik, tetapi jumlah dan hasilnya tetap dicatat.

Untuk pasangan valid ke-i, hitung `d_i = TTFB_gateway_i - TTFB_control_i`. Urutkan d_i dan ambil elemen berperingkat `ceil(0,95 × N)`, dengan peringkat dimulai dari satu. Masing-masing run harus memenuhi p95(d_i) ≤50 ms. Jangan menggantinya dengan selisih dua p95, memotong nilai negatif menjadi nol, membuang outlier, atau menyisipkan latency buatan. Pasangan error/timeout dicatat sebagai kegagalan dan membuat run tidak lengkap/gagal; jangan membuangnya lalu mengklaim 1.000 pasangan sukses.

Simpan durasi tiap jalur/pasangan, p50/p95/p99, jumlah sukses/gagal, estimator, seed/urutan dan identitas fixture. Angka gabungan tidak boleh menyembunyikan satu run yang melampaui target. Stress body 10 MiB dilakukan terpisah dari profil utama.

### Q-5: 20 stream kontinu dan monitoring aktif

Satu soak wajib mempertahankan 20 stream secara bersamaan selama 1.800 detik, dihitung setelah barrier menyatakan seluruh 20 stream aktif. Gunakan lima alias fixed × empat stream. Ledger start/end dan inflight membuktikan seluruh stream tetap aktif sepanjang jendela; bukan 20 request kumulatif atau penggantian stream yang putus. Payload request utama 64 KiB; stream mengirim potongan sintetis sampai observer menutup jendela. Fixture dan client harus dapat menyelesaikan respons/teardown secara deterministik.

Gunakan total timeout 2.100 detik **khusus profil soak**, agar setup dan jendela 30 menit tidak bertabrakan dengan usulan historis total timeout 1.800 detik. Ini tidak menetapkan default total request produk dan tidak mengubah batas keras shutdown D-014. Connect/header/idle serta parameter probe lengkap ditetapkan task pemilik sebelum benchmark, lalu dicatat; timer idle tidak boleh secara tidak sengaja mematikan profil yang sedang diuji.

Aktifkan dashboard terintegrasi pada tiga tab dengan polling 5 detik dan satu request pembaruan aktif per tab. Bandingkan jumlah probe dengan jadwal server; refresh/tab tambahan tidak boleh membuat probe atau generasi baru. Jalankan kasus kegagalan pembaruan dan pemulihan terpisah, termasuk stale setelah >15 detik, agar keberhasilan polling tidak menyamarkan probe lama.

Gateway, lima SOCKS5 relay fixture dan upstream fixture memakai batas server agregat 2 vCPU/2 GiB yang diverifikasi. Batas tersebut bukan jatah yang dikalikan per container. Generator/browser dapat berada di luar kelompok server dengan penempatan dan konsumsi dicatat. Build toolchain tidak berjalan saat jendela ukur. Laporkan penggunaan WARP, fixture, generator/browser, gateway, serta total deployment secara terpisah; benchmark SOCKS5 sintetis tidak mengukur konsumsi lima WARP aktual.

Catat RSS **proses gateway**, inflight, koneksi, byte terkelola, binding/sesi, probe dan pembaca status setiap 1 detik. Simpan timestamp/urutan sampel dan nilai maksimum; gunakan peak/high-water RSS OS bila tersedia. RSS bukan heap Go atau memory total container. Nilai maksimum yang teramati harus ≤256 MiB. Sampling tidak membuktikan ketiadaan spike di antara sampel; laporkan keterbatasan tersebut. Sampel hilang, stream terputus, OOM, kapasitas kurang, atau RSS melewati target menghalangi kelulusan; jangan menutup celah dengan nilai interpolasi atau mengurangi beban.

### Q-6: cancellation dan lifecycle

Jalankan 30 percobaan untuk masing-masing skenario disconnect saat menunggu header dan saat stream aktif. Upstream harus sudah mengamati request dan masih aktif ketika cancellation diukur. `t_detect` adalah saat gateway mendeteksi disconnect; `t_cancel` saat upstream mengamati cancellation aktif; `t_release` saat reservasi selesai dilepas. Gunakan satu observer clock atau timebase yang terbukti sebanding, bukan pengurangan sembarang timestamp monotonic antarproses. Latency sejak client disconnect hingga deteksi dicatat terpisah.

Setiap percobaan wajib memenuhi `t_cancel - t_detect ≤2 detik`, bukan sekadar rata-rata atau p95. Catat latency pelepasan resource, pastikan slot/byte/koneksi dilepas tepat sekali, dan bedakan EOF sukses dari cancellation. Uji 20 disconnect serentak sebagai kasus resource terpisah; tidak menambah attempt generasi pada fault ambigu.

Drain diuji terpisah: tolak admission baru, termasuk sesi terikat; stream/binding aktif tetap diperiksa. Pada 60 detik, stream aktif tidak dihentikan paksa. Pada batas keras 30 menit, pemutusan dicatat dan proses keluar dengan exit code eksplisit; `stop_grace_period` produk harus lebih besar. Grace 5 detik pada fixture packaging T05 bukan bukti kontrak shutdown produk.

### Stress dan syarat hasil gagal

Selain profil primer, jadwalkan body oversized/setelah decoding, 20 body maksimum versus cap agregat, encoding tidak didukung, slow upload/consumer, snapshot lambat/multitab, binding/sesi penuh, probe macet, pool penuh/down, serta cleanup setelah error/cancel. Catat penolakan, jumlah dispatch, puncak buffer dan resource yang tersisa. Stress tidak menggantikan soak primer dengan beban lebih ringan. Perubahan profil/target setelah melihat hasil memerlukan keputusan yang dicatat; hasil sebelumnya tetap disimpan beserta alasan kegagalannya.

## 5. Baseline G-1–G-7

Sumber manual dan fixture mengikuti [D-005](DECISIONS.md#d-005-baseline-dua-tahap-dan-pelaksanaan-pengukuran-g-1g-7) serta [matriks awal T01](M0-lingkungan-dan-integrasi.md#4-sumber-dan-pencatatan-baseline-g-1g-7). D-019 menetapkan protokol awal: sedikitnya tiga episode penggunaan/prosedur manual terpisah yang memang tersedia, serta sedikitnya tiga pengulangan per skenario fixture wajib yang relevan. Sampel manual dan sintetis dilaporkan terpisah, bukan dirata-ratakan sebagai populasi yang sama.

| G | Definisi/unit | Sumber manual dan fixture yang direncanakan | Baseline aktual | Pemilik/task penutup |
|---|---|---|---|---|
| G-1 | Client yang menuntaskan percakapan dan satu siklus tool call; pembilang client lulus/daftar client wajib | Rekaman client yang benar-benar dipakai; jalur kontrol M0 dicatat terpisah; CLI dan Desktop produk tetap diterima masing-masing pada M3 | TBD | Pengguna/operator + agent; EP-01-T01/T04 dan rekap T06; penerimaan EP-06-S01/S02 |
| G-2 | Skenario JSON/SSE/tool-call deltas yang lulus/total skenario wajib | Catatan format/kegagalan penggunaan saat ini dan assertion fixture dengan seed/fragmentasi; bukan salinan konten percakapan | TBD | Pengguna + agent; EP-01-T03/T04, EP-02-S03/S04 |
| G-3 | Field metadata yang tersedia/field wajib per skenario | Catatan diagnosis manual, identitas relay/jalur/fase dispatch dan metadata teredaksi fixture | TBD | Pengguna + agent; EP-01-T03/T04, EP-05-S05 |
| G-4 | Jumlah replay generasi setelah dispatch ambigu/stream dimulai, dengan denominator fault yang diuji | Riwayat kegagalan yang dapat dibuktikan dan ledger attempt/dispatch fixture sebelum/ambigu/setelah stream | TBD | Pengguna + agent; EP-01-T03, EP-03-S04 |
| G-5 | Jumlah perubahan konfigurasi client per prosedur pembaruan credential server | Rekaman prosedur yang diotorisasi, lalu fixture credential sintetis; tidak melakukan rotasi nyata atau mengasumsikan hot reload | TBD | Pengguna/operator + agent; EP-01-T01/T06; pembuktian integrasi EP-06-S01/S02/T04 |
| G-6 | Relay dengan catatan identitas/status observasi yang lengkap atau ketidaktersediaan eksplisit/seluruh relay terkonfigurasi pada lingkungan yang dinilai; hasil IPv4/IPv6 aktual dihitung terpisah | Inventaris observasi yang sudah tersedia dan WARP T04 bila diotorisasi; relay belum diamati tetap masuk denominator; IP duplikat/IPv6 tidak tersedia diberi status, bukan janji IP unik | TBD | Pengguna/operator + agent; EP-01-T04/T06, EP-05-S03 |
| G-7 | Skenario monitoring/status/perubahan/error/auth yang lulus/total skenario | Inventaris pemantauan yang benar-benar tersedia; fixture sumber status dan browser produk pada EP-05; contoh UI T05 bukan dashboard pembanding | TBD | Pengguna/operator + agent; EP-01-T01/T06, EP-05-T01/S03/S04/T06 |

Format tiap catatan memuat G, tahap manual/fixture, waktu dan zona, client/runtime/jalur, skenario/seed, jumlah sampel aktual dan rencana, pembilang/penyebut atau nilai+satuan, gagal/tidak tersedia, pemilik, dan referensi bukti teredaksi. G-6 memakai seluruh relay terkonfigurasi pada lingkungan yang dinilai sebagai denominator, termasuk relay belum diamati. Bedakan kelengkapan catatan/status dari jumlah observasi IPv4/IPv6 aktual; status tidak tersedia bukan hasil probe terbaru. Jangan menyamakan lima relay sintetis dengan lima identitas WARP yang terbukti.

Jumlah sampel bukan otorisasi tiga generasi/probe/rotasi baru. Bukti nyata mengikuti otorisasi dan budget sesi yang berlaku, termasuk D-018; jangan menambah request untuk mengejar minimum. Jika data atau client belum tersedia, catat kekurangan dan owner, pertahankan TBD, serta bawa keputusan kekurangan data sebelum M1. Tidak ada waiver D-005 yang diberikan T06. Nilai TBD tidak menjadi nol, dan kelulusan suite historis tidak otomatis mengisi baseline penggunaan nyata. Penerimaan G pada M2/M3 tetap diuji pada produk.

## 6. Input dan default yang belum final

| Input/default | Bukti/usulan historis | Status dan pemilik | Task/batas keputusan |
|---|---|---|---|
| Connect/header/idle/total timeout | Contoh 10/120/120/1.800 detik pada catatan teknis | TBD final; pengguna + pelaksana backend; profil soak 2.100 detik tidak menggantikannya | EP-02-T01/S05 sebelum konfigurasi gateway dasar diterima; verifikasi EP-06-T03 sebelum M3 |
| Interval/jitter/backoff/timeout/threshold probe | Usulan 30 detik dengan jitter, timeout 5 detik, unhealthy setelah 3 gagal dan pulih setelah 2 sukses; jitter/backoff rinci belum ada | TBD final; pengguna + pelaksana health; tidak ada generasi berbayar otomatis | EP-03-S02 sebelum M2.1 diterima; parameter benchmark lengkap sebelum EP-06-T03 |
| Header sesi, TTL dan kapasitas binding | Fixture EP-04-T01 memakai header eksplisit, idle TTL 60 menit, kapasitas 1.024 binding (D-029) | Kontrak fixture ditetapkan; default produk/runtime dan kemampuan client tetap TBD; D-019 tetap profil, bukan bukti enforcement | D-029; runtime dan integrasi tetap perlu dibuktikan sebelum penerimaan M2.2/M3 |
| Sesi admin/TTL/expiry/logout/origin dan kapasitas final | Tidak ditetapkan oleh Basic auth fixture T05 atau jumlah sesi profil | TBD; pengguna + backend/UI | EP-05-T01 sebelum AC-21/AC-22 dinyatakan lulus |
| Grace/exit/log shutdown | Grace harus >30 menit; nilai dan exit code final belum ada | TBD; pengguna/operator + backend | EP-04-T05 sebelum uji EP-04-S06; verifikasi EP-06-T03 sebelum M3 |
| SDK aktif, akses Free, JSON/non-stream, listener/tailnet | Provider/model/client dipilih; detail tersebut belum terbukti | TBD; pengguna/operator + agent | EP-01-T01/T04 sebelum menutup M0 |
| Batas produk dan schema resource | Profil D-019 tersedia; byte accounting dan kapasitas aktual belum diimplementasikan | TBD final; pelaksana area pada bagian 3 bersama pengguna | Task area sebelum penerimaannya; EP-06-T03 memverifikasi keseluruhan sebelum M3 |
| Ukuran aset/image produk, konsumsi WARP/total deployment | T05 hanya memberi identitas artefak contoh | TBD; pengguna/operator + agent | EP-06-T03 sebelum M3; laporkan terpisah dari RSS gateway |

Nilai final yang belum tersedia tetap terbuka dengan task dan pemilik. Tidak mengubah catatan historis menjadi default yang otomatis disetujui. Input protokol dan akses yang memengaruhi kelayakan harus diselesaikan sebelum bagian implementasi yang bergantung padanya diterima.

## 7. Matriks penerimaan AC-01–AC-22

Sumber kontrak adalah [catatan teknis](PRD-technical-notes.md#15-acceptance-criteria-dan-pengujian), [tambahan dashboard](PRD-technical-notes.md#tambahan-kebutuhan-versi-012), dan keputusan terbaru. Milestone pertama mengikuti [ROADMAP](ROADMAP.md#7-matriks-penerimaan), owner child mengikuti [indeks AC](README.md#pemetaan-acceptance-sumber). **Semua baris berikut adalah rencana produk**, bukan hasil tes yang dijalankan T06. Bukti M0/control dipisahkan dari bukti produk; seluruh AC diulang pada artefak rilis M3 yang sama melalui EP-06-T05.

Nama berkas pada kolom bukti adalah rencana dalam bundle pada bagian 8, belum merupakan berkas yang tersedia. Bukti konten hanya memakai data sintetis; request nyata hanya menyimpan assertion/metadata teredaksi.

| AC | Metode dan assertion utama | Lingkungan | Milestone pertama/ulang | Bukti yang disimpan | Pemilik task | Status |
|---|---|---|---|---|---|---|
| AC-01 | Pilih alias, selesaikan percakapan dan satu siklus tool call CLI; catat retry client aktif dan kegagalan kompatibilitas | CLI versi tercatat → gateway → WARP → provider yang diotorisasi | M3 | `ac-01.json`: versi/config teredaksi, hasil alur dan metadata jalur | EP-06-S01 | Rencana |
| AC-02 | Alur setara AC-01 dari Desktop; kelulusan CLI tidak meluluskan Desktop | Desktop versi tercatat dan jalur nyata yang sama | M3 | `ac-02.json`: versi, langkah/assertion Desktop dan jalur | EP-06-S02 | Rencana |
| AC-03 | Fragmentasi byte deterministik/acak, UTF-8, content/tool deltas/usage/field tambahan utuh; event pertama sebelum terminal | Gateway + HTTP/TLS/SOCKS5 fixture nyata | Kontrol M0; produk M1; ulang M3 | `ac-03.json`, seed/pola fragmentasi dan ledger flush/dispatch | EP-01-T03, EP-02-S04 | Rencana |
| AC-04 | JSON/non-stream mempertahankan status, content type, struktur/payload dan usage, termasuk error | Gateway + upstream JSON sintetis | M1; ulang M3 | `ac-04.json`: assertion semantik JSON sintetis dan status | EP-02-S03 | Rencana |
| AC-05 | Relay terpilih cocok peer/CONNECT; proxy mati tidak bypass; kontrol direct bisa mencapai target dan ditolak oracle | HTTP/SOCKS5 fixture; smoke WARP nyata terpisah | M0; produk M1; ulang M3 | `ac-05.json`, counter dial/CONNECT/dispatch/direct dan peer | EP-01-T03/T04, EP-02-S05 | Rencana |
| AC-06 | CONNECT membawa hostname upstream; tidak ada resolusi lokal tujuan upstream; TLS hostname tetap diverifikasi | SOCKS5/TLS fixture dan jalur WARP target | M0; produk M1; ulang M3 | `ac-06.json`: domain CONNECT dan observasi resolver/dial | EP-01-T03/T04, EP-02-S05 | Rencana |
| AC-07 | Fault sebelum dispatch yang terbukti memberi fallback non-strict maksimal dua attempt total; bukti tidak ada berarti satu | Fault fixture + coordinator gateway | Kontrol M0; produk M2.1; ulang M3 | `ac-07.json`: fase fault, attempt dan ledger dispatch | EP-01-T03, EP-03-S04 | Rencana |
| AC-08 | POST ambigu, timeout header, atau SSE putus tidak menghasilkan generasi kedua; koneksi baru/reusable serta retry SDK/client dicatat | HTTP/SOCKS5 fault fixture + transport gateway | Kontrol M0; produk M2.1; ulang M3 | `ac-08.json`: dispatch tepat satu per fault ambigu, timeline/error stream | EP-01-T03, EP-03-S04 | Rencana |
| AC-09 | Retry-After valid diteruskan, invalid ditangani eksplisit; scope tidak diketahui konservatif upstream/credential-wide; tanpa restart/rekey/key swap | Upstream 429 sintetis + pool/cooldown | M2.1; ulang M3 | `ac-09.json`: scope, deadline, header dan counter tindakan | EP-03-S05 | Rencana |
| AC-10 | Token hilang/salah/cross-use ditolak; credential upstream menggantikan token client tanpa bocor ke client/log | Endpoint model dan admin terpisah + fixture auth | M1; ulang M3 | `ac-10.json`: status/dispatch nol untuk penolakan dan assertion redaksi | EP-02-S02 | Rencana |
| AC-11 | URL/header target bebas dan redirect tidak memilih tujuan lain; sertifikat hostname salah ditolak; credential tidak dikirim ke origin lain | Gateway + TLS/redirect/SOCKS5 fixture | M1; ulang M3 | `ac-11.json`: allowlist/dial/redirect ledger dan assertion TLS | EP-02-S05 | Rencana |
| AC-12 | Disconnect membatalkan upstream ≤2 detik dan melepas slot/byte/koneksi tepat sekali; EOF sukses bukan cancel | Fixture stream nyata + gateway, termasuk konkurensi | M2.1; ulang M3 | `ac-12.json`: t_detect/t_cancel/t_release, hasil tiap trial dan race/resource counters | EP-03-S06 | Rencana |
| AC-13 | Sesi eksplisit mempertahankan token/upstream/model/relay/generasi; kehilangan binding mendapat 409; strict+least-inflight ditolak | Fixture sesi eksplisit, bukan klaim strict client MVP | M2.2; ulang M3 | `ac-13.json`: assertion binding/generasi teredaksi, kapasitas dan invalidasi | EP-04-T01/S02/S03 | Rencana |
| AC-14 | Setelah restart, kelanjutan sesi strict lama ditolak; tidak rebinding atau membentuk sesi baru diam-diam | Fixture sesi/generasi dengan restart terkontrol | M2.2; ulang M3 | `ac-14.json`: identitas proses/generasi dan penolakan kelanjutan | EP-04-T01/S03 | Rencana |
| AC-15 | Drain menolak assignment baru termasuk sesi terikat; stream relay lain berjalan; binding aktif dan batas shutdown D-014 diperiksa | Gateway + multirelay fixture/lifecycle | M2.1; lengkap M2.2; ulang M3 | `ac-15.json`: timeline admission/inflight/binding, log shutdown/exit/grace | EP-04-S04/T05/S06 | Rencana |
| AC-16 | Semua relay down menghasilkan 503 terstruktur; tidak ada direct dial/fallback internet | Gateway + multirelay down fixture | M2.1; ulang M3 | `ac-16.json`: error, counter direct/dispatch nol | EP-03-S02 | Rencana |
| AC-17 | IPv4 duplikat/IPv6 tidak tersedia tampil dengan tujuan/waktu observasi dan batas klaim, tanpa janji IP unik | Snapshot/UI fixture; observasi WARP bila diotorisasi | M2.3; ulang M3 | `ac-17.json`: snapshot referensi teredaksi dan assertion status/UI | EP-05-S03 | Rencana |
| AC-18 | Oversized/setelah decode, encoding unsupported, byte cap, overload dan consumer lambat dibatasi; resource kembali setelah error/cancel | Gateway + fixture jaringan dengan batas aktual | M2.1; ulang M3 | `ac-18.json`: penolakan/dispatch, puncak byte/koneksi/slot/state dan RSS | EP-03-S01/S06 | Rencana |
| AC-19 | Jalankan Q-4 paired p95, Q-5 kontinu dan metode Q-6; fixture/beban/limit tidak sesuai atau target terlampaui berarti gagal | Server 2 vCPU/2 GiB, 5×5 slot, upstream lokal, dashboard aktif | M3 | `ac-19.json`, `latency.csv`, `resources.csv`, parameter/command/digest dan hasil tiap run | EP-06-T03 | Rencana |
| AC-20 | Seluruh relay dan ringkasan/state/inflight/kapasitas/IP/cooldown/alasan cocok dengan snapshot server | Browser + artefak gateway terintegrasi + fixture state | M2.3; ulang M3 | `ac-20.json`: snapshot pembanding dan assertion browser; screenshot aman bila perlu | EP-05-T01/S03/T06 | Rencana |
| AC-21 | Poll 5 detik; stale >15 detik, error dan last-success terlihat; waktu probe terpisah; refresh/multitab tidak menambah probe | Browser clock/network terkontrol + counter probe gateway | M2.3; ulang M3 | `ac-21.json`: timeline server/probe/browser, recovery dan ledger probe | EP-05-T01/S04/T06 | Rencana |
| AC-22 | UI dan sumber data menolak tanpa admin, token gateway, sesi expired/logout; URL/storage/bundle/snapshot/log tidak bocor | Browser + auth/admin API terintegrasi dengan kontrak sesi EP-05-T01 | M2.3; ulang M3 | `ac-22.json`: hasil negatif auth/TTL/logout dan assertion redaksi | EP-05-T01/S02/S05/T06 | Rencana |

## 8. Bundle bukti dan cara reproduksi yang harus disediakan

Task pemilik menyimpan bundle per run, misalnya lokasi **rencana** `docs/verification/<run-id>/`, berisi manifest lingkungan, konfigurasi teredaksi, command yang benar-benar dijalankan beserta exit code, hasil per AC/G/Q, metrik nonkonten dan checksum. Lokasi tersebut belum dibuat dan tidak menyiratkan runner acceptance produk sudah tersedia.

Manifest mencatat tanggal/waktu ISO 8601 dan offset zona (Asia/Jakarta, +07:00), HEAD serta fingerprint working tree jika belum commit, identitas binary/image yang diuji, digest base/builder, exact version client/runtime provider, arsitektur, CPU/memory aktual dan cakupan cgroup, protokol/TLS/koneksi, pemetaan alias/relay/kapasitas, seed/payload sintetis, probe/timeout, tab/polling, sampel yang direncanakan/aktual, kegagalan dan batas pembuktian. Bedakan image ID lokal dari digest registry serta ukuran image terkompresi/hasil ekstrak bila diukur.

Jangan simpan secret, nilai Authorization, config akun mentah, prompt/respons/tool body nyata, profil WARP, URL bertoken atau raw session ID. Assertion konten memakai fixture sintetis; artefak nyata menyimpan hasil assertion dan metadata saja. Request/session ID tidak dijadikan label metrik. Usage yang tidak tersedia tidak dilaporkan nol atau perkiraan billing.

Task pemilik harus menambahkan command eksekusi beserta setup/teardown dan batas otorisasi yang dapat direproduksi ketika runner tersedia. T06 tidak menciptakan atau mengklaim command benchmark baru. Perubahan image/config/client/SDK yang memengaruhi hasil memerlukan verifikasi ulang pada artefak terkait; M3 tidak merakit klaim lulus dari image berbeda tanpa penjelasan. Semua hasil gagal disimpan. Cleanup hanya resource fixture yang diotorisasi, tanpa menyentuh deployment/profil aktif.

## 9. Status acceptance T06 dan tindak lanjut

| AC T06 | Hasil dokumentasi | Batas |
|---|---|---|
| AC1 | Parsial: rekap host/client/model, versi, bukti dan implikasi tersedia | Seluruh hasil spike belum tersedia; T04 dan input T01 masih menahan penerimaan penuh |
| ✅ AC2 | Target tetap; profil resource/metode D-019 dikonfirmasi; input/default final punya pemilik/task | Tidak membuktikan accounting runtime, RSS atau default produk final |
| ✅ AC3 | 22 AC masing-masing memiliki metode, lingkungan, milestone, artefak dan owner | Semua baris adalah rencana; belum kelulusan acceptance produk |
| ✅ AC4 | Gerbang M0 dinyatakan belum lulus dan aturan bukti gagal/hilang dipertahankan | Tidak ada penghapusan dependensi atau pengesahan milestone |

Baseline aktual, T04 dan input T01 dituntaskan pada task pemilik sebelum penutupan M0/M1 sesuai D-005. Setelah bukti tersedia, T06 ditinjau ulang, terutama AC1. Status persetujuan tiket tetap `approved`. Rencana ini tidak memberi izin generasi/probe baru, perubahan lifecycle deployment, commit, push, publish atau rilis.
