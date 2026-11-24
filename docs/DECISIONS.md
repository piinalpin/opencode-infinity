# Keputusan opencode-infinity

Versi: 0.1.18
Tanggal: 24 November 2026

Catatan ini memisahkan keputusan eksplisit pengguna dari usulan dan pertanyaan yang belum selesai. Keputusan terkonfirmasi diterapkan ke PRD/arsitektur terkait; status keseluruhan draft tidak otomatis menjadi approved.

## Keputusan terkonfirmasi

### D-001 Deployment siap container di lokal atau homelab

- **Status:** ditetapkan pengguna.
- **Keputusan:** gateway dan pool WARP dapat dijalankan pada lingkungan lokal atau server homelab. Kebutuhan utamanya adalah siap dijalankan sebagai container.
- **Tanggung jawab:** pengguna menyiapkan deployment homelab bila diperlukan.
- **Implikasi:** paket aplikasi tidak mengharuskan penyediaan homelab otomatis. Pilihan ini tidak meminta gateway dan pool tersebar pada beberapa host sekaligus.
- **Belum diputuskan:** arsitektur CPU image, OS/runtime host aktual, dan detail pengoperasian container. Kandidat pada TECHSTACK tetap perlu validasi M0.
- **Acuan:** O-2, C-10, dan EP-01-T01.

### D-002 Akses homelab melalui Tailscale

- **Status:** ditetapkan pengguna.
- **Keputusan:** deployment homelab diakses melalui Tailscale.
- **Tanggung jawab:** pengguna menyiapkan akses homelab tersebut.
- **Implikasi:** jalur akses homelab bersifat privat; detail listener dan akses dashboard mengikuti kontrak admin/gateway.
- **Belum diputuskan:** alamat listener aktual. Penempatan Tailscale serta terminasi HTTPS ditetapkan kemudian pada D-006.
- **Acuan:** O-2, C-8, C-10, dan C-14.

### D-003 Pengguna awal adalah pemilik sendiri

- **Status:** ditetapkan pengguna.
- **Keputusan:** pengguna yang mengakses deployment awal adalah pengguna sendiri, yang menjalankan peran developer dan operator.
- **Implikasi:** kebutuhan awal berfokus pada penggunaan pribadi; tidak ada kebutuhan onboarding tim atau isolasi tenant penuh yang ditambahkan.
- **Belum diputuskan:** upstream/model, versi client uji, baseline, target rilis, dan anggaran. Penggunaan pribadi tidak menghapus kebutuhan pemisahan token atau otorisasi admin.
- **Acuan:** O-1, bagian pengguna awal pada O-8, serta C-1/C-8/C-10/C-14.

### D-004 Pemilik keputusan produk dan target keberhasilan

- **Status:** ditetapkan pengguna.
- **Keputusan:** pengguna sendiri menjadi pemilik keputusan produk dan menetapkan target keberhasilan proyek.
- **Dasar:** pengguna memilih opsi 1 pada pertanyaan pemilik keputusan produk dan target keberhasilan.
- **Implikasi:** keputusan produk serta pengesahan target kembali kepada pengguna; pilihan ini tidak otomatis menyetujui angka usulan pada PRD atau seluruh draft.
- **Belum diputuskan:** baseline aktual, bukti masalah, dan pelaksana/cara pengukuran teknis.
- **Acuan:** bagian kepemilikan produk pada O-1 dan EP-01-T01.

### D-005 Baseline dua tahap dan pelaksanaan pengukuran G-1–G-7

- **Status:** ditetapkan pengguna.
- **Keputusan:** baseline diisi dua tahap. Tahap pertama mencatat bukti manual dari penggunaan nyata saat ini (waktu penyiapan relay/client, jumlah kegagalan, langkah manual yang berulang) sebagai bukti awal. Tahap kedua mengukur ulang secara terinstrumentasi pada fixture M0 (EP-01-T03/EP-01-T04) sehingga G-1–G-7 memperoleh angka baseline sebelum M1.
- **Pelaksana:** pengguna sebagai operator dan developer; agent membantu instrumentasi fixture serta perhitungan hasil. Penanggung jawab bernama tidak ditambahkan.
- **Implikasi:** O-1 tidak lagi menahan pembuatan fixture dan spike, tetapi klaim dampak maupun pengesahan target numerik PRD belum boleh dibuat sampai baseline M0 terkumpul. O-6 (target benchmark 20 stream) tetap terbuka; target pada PRD tetap provisional.
- **Belum diputuskan:** angka baseline aktual, format pencatatan manual, dan jumlah sampel minimum per metrik.
- **Acuan:** O-1, G-1–G-7, EP-01-T01, dan EP-01-T06.

### D-006 Host uji amd64 dan Tailscale di host

- **Status:** ditetapkan pengguna.
- **Keputusan:** host uji adalah mesin Linux x86_64 yang dipakai untuk pengembangan (Docker server `amd64`, Docker Compose 5.7.1). Image dibangun sebagai `linux/amd64`. Tailscale dipasang di host, bukan sebagai sidecar container; gateway dan relay hanya memublikasikan listener ke loopback host, dan akses dari tailnet melewati `tailscale serve` yang menyediakan terminasi HTTPS.
- **Implikasi:** tidak ada container Tailscale, state autentikasi Tailscale, atau urutan startup tambahan di Compose. Container gateway tetap tanpa Docker socket dan tanpa NET_ADMIN; izin jaringan khusus hanya diberikan pada service relay WARP. Image yang sama dapat dipakai pada host amd64 lain, tetapi konfigurasi host (Tailscale dan `tailscale serve`) harus diulang pada tiap deployment.
- **Belum diputuskan:** alamat/port listener final, hostname tailnet, serta detail sertifikat dan kebijakan akses Tailscale. Dukungan arm64/multi-arch berada di luar cakupan MVP.
- **Acuan:** O-2, D-001/D-002, ARCHITECTURE bagian deployment, dan EP-01-T01.

### D-007 Client utama Desktop dan upstream Chat Completions

- **Status:** ditetapkan pengguna.
- **Keputusan:** client utama M0 adalah OpenCode Desktop yang sedang dipakai (versi persis dicatat saat M0), dengan upstream pertama berupa endpoint Chat Completions yang kompatibel OpenAI dan satu model coding besar yang sudah tersedia bagi pengguna. Fixed routing menjadi default. OpenCode CLI ditambahkan sebelum gerbang M3 agar G-1 terbukti; strict affinity diuji setelah sinyal sesi tersedia.
- **Implikasi:** Responses API (C-11) dan adapter protokol native Anthropic tetap P1. Model awal harus mendukung streaming SSE dan tool calling melalui Chat Completions; tidak ada penggantian credential atau model otomatis saat gagal. Angka baseline G-1–G-7 tetap diukur menurut D-005.
- **Tindak lanjut M0:** provider/model dipilih kemudian melalui D-016; versi Desktop dan CLI bawaan `2.3.0` tercatat pada [matriks lingkungan](M0-lingkungan-dan-integrasi.md). Kesesuaian akses gateway, kebutuhan sesi, dan versi SDK terpisah tetap perlu dibuktikan. Ketidakcocokan tidak boleh membuat M0 ditandai lulus diam-diam.
- **Acuan:** O-3, G-1, C-1/C-2, EP-01-T01, EP-01-T02, dan EP-02.

### D-008 Target performa provisional dan fixture benchmark 5×5

- **Status:** ditetapkan pengguna.
- **Keputusan:** target Q-4 (overhead p95 ≤50 ms), Q-5 (20 stream bersamaan selama 30 menit, RSS gateway ≤256 MiB), dan host benchmark yang dibatasi 2 vCPU/2 GiB disetujui sebagai target penerimaan provisional. Fixture benchmark memakai 5 relay dengan 5 slot masing-masing (25 kapasitas), bukan contoh deployment 2×4 pada dokumentasi. Limit CPU/memori diterapkan melalui cgroup/Compose pada host uji, dan konfigurasi benchmark dicatat menurut Q-10.
- **Implikasi:** target boleh direvisi hanya melalui keputusan baru bila M0/M1 membuktikan angka tidak realistis; penurunan target tanpa keputusan tidak diperbolehkan. Contoh Compose minimal dua relay tetap hanya contoh deployment.
- **Belum diputuskan:** ukuran aset/image dan pemakaian WARP/total deployment diukur terpisah; default timeout, probe, dan buffer final mengikuti usulan catatan teknis sampai diuji pada M3.
- **Acuan:** O-6, Q-4, Q-5, Q-10, ROADMAP M3, dan EP-01-T06.

### D-009 Dashboard polling pada MVP dan push SSE di P1

- **Status:** ditetapkan pengguna.
- **Keputusan:** dashboard MVP memakai polling snapshot admin tiap 5 detik. Data ditandai stale setelah 15 detik tanpa pembaruan sukses (tiga interval terlewat), dan setiap kegagalan pengambilan data ditampilkan bersama waktu pembaruan terakhir. Tombol refresh hanya membaca snapshot/probe yang sudah ada dan tidak memicu probe WARP baru. Push SSE/WebSocket ditunda ke P1 dengan kontrak tersendiri.
- **Implikasi:** C-14 tetap bagian MVP dengan polling; tidak ada endpoint push, buffer per klien, atau koneksi persisten di scope MVP. Banyak tab tidak menambah probe karena semua membaca state yang sama. Tiket P1 untuk push belum dibuat dan menunggu prioritas O-9.
- **Belum diputuskan:** target kesegaran P1 push (interval/ambang stale), mekanisme autentikasi koneksi push, dan batas jumlah klien.
- **Acuan:** O-10, Q-11, AC-20–AC-22, ARCHITECTURE bagian dashboard, dan EP-05-T01.

### D-010 Rilis MVP event-driven tanpa anggaran baru

- **Status:** ditetapkan pengguna.
- **Keputusan:** penanggung jawab rilis adalah pengguna sendiri. Tidak ada tanggal kalender atau anggaran baru; deployment memakai host serta langganan upstream yang sudah tersedia. MVP dirilis ketika seluruh gerbang M3 lulus dan diberi tag `v0.1.0-mvp`. Tidak ada rencana peluncuran publik.
- **Implikasi:** jadwal bersifat event-driven, bukan komitmen tanggal. Keterlambatan M0 tidak otomatis memotong scope Must; perubahan scope tetap memerlukan keputusan produk. Biaya pihak ketiga (upstream/model, host) tetap tanggung jawab pengguna dan tidak diklaim sebagai anggaran proyek.
- **Belum diputuskan:** isi catatan rilis, mekanisme distribusi image, dan apakah tag bersifat anotasi; ditetapkan pada EP-06-T05.
- **Acuan:** O-8, D-003/D-004, ROADMAP M3, dan EP-06-T05.

### D-011 Tanpa SLA formal, pemulihan best-effort

- **Status:** ditetapkan pengguna.
- **Keputusan:** MVP personal tidak memiliki SLA ketersediaan formal maupun RTO/RPO yang dijanjikan. Target pemulihan bersifat best-effort: operator mengenali gangguan melalui dashboard/status, menjalankan prosedur restart gateway/relay, dan mencatat waktu pemulihan aktual sebagai observasi. Tidak ada sistem alerting wajib yang ditambahkan ke scope MVP.
- **Implikasi:** dokumentasi dan UI tidak boleh mengklaim jaminan uptime. Angka uptime atau waktu pemulihan hanya dilaporkan bila benar-benar tercatat, bukan estimasi. Kegagalan upstream pihak ketiga tetap di luar kendali proyek.
- **Belum diputuskan:** format pencatatan insiden (jika diperlukan) dan mekanisme review berkala; tidak memblokir MVP.
- **Acuan:** O-7, Q-9, ROADMAP M3, dan EP-06-T05.

### D-012 Urutan prioritas P1

- **Status:** ditetapkan pengguna.
- **Keputusan:** urutan P1 adalah C-11 Responses API, lalu C-15 plugin session ID, C-16 export konfigurasi dan secret store, C-13 controller maintenance WARP, dan C-12 integrasi relay HTTP lama. Hot reload tidak diprioritaskan dan tetap tanpa tiket.
- **Alasan:** kompatibilitas client/model baru paling mungkin dibutuhkan lebih dulu; maintenance WARP manual sudah tercakup prosedur drain MVP; relay HTTP lama hanya relevan bila stack tersebut benar-benar dipertahankan.
- **Implikasi:** epic P1 tetap berdiri sendiri dan tidak wajib menunggu Responses API selesai, kecuali dependensi kontrak yang memang tertulis. Urutan dapat direvisi melalui keputusan baru bila kebutuhan berubah.
- **Belum diputuskan:** penanggung jawab dan dependensi kontrak tiap epic P1 dicatat saat EP-07-T01 ditutup setelah penerimaan MVP.
- **Acuan:** O-9, PRD §6.2, README indeks P1, dan EP-07-T01.

### D-013 Fixed routing pada MVP, lifecycle strict dibuktikan lewat fixture

- **Status:** ditetapkan pengguna.
- **Keputusan:** MVP memakai fixed routing sebagai default dan tidak mengklaim strict affinity per percakapan pada client nyata, karena OpenCode Desktop belum memasok sinyal sesi stabil. Lifecycle strict (sesi baru, sesi lanjutan, TTL/kedaluwarsa, dan penolakan sesi lama setelah restart) dibuktikan pada fixture dengan header sesi eksplisit. Dukungan strict untuk client nyata menunggu C-15 plugin session ID pada P1.
- **Implikasi:** C-6 tetap dipenuhi sebatas binding yang didukung: binding menjaga relay dan generasi identitas, kehilangan binding ditolak eksplisit, dan keterbatasan client tanpa sinyal sesi didokumentasikan. Tidak ada heuristik sesi dari token, IP, atau isi prompt. Drain tetap memeriksa binding aktif.
- **Belum diputuskan:** skema header sesi yang dipakai fixture dan ambang TTL; dicatat pada EP-04-T01.
- **Acuan:** O-4, C-6/C-7, Q-3, EP-04-T01, dan EP-04-T05.

### D-014 Drain 60 detik dengan batas keras yang tercatat

- **Status:** ditetapkan pengguna.
- **Keputusan:** deadline drain 60 detik hanya menghentikan admission/assignment baru; stream dan binding yang sudah di-admit dilanjutkan sampai selesai dengan batas keras 30 menit. Status shutdown, daftar stream/binding yang masih aktif, serta keputusan akhir terlihat pada log metadata dan status gateway. Bila batas keras tercapai, gateway mencatat stream/binding yang diputus lalu keluar dengan exit code eksplisit. Compose menetapkan `stop_grace_period` pada service gateway lebih besar dari batas keras tersebut.
- **Implikasi:** tidak ada SIGKILL tanpa catatan; penghentian paksa hanya tindakan operator atau sistem setelah batas keras. Drain tetap menolak assignment baru untuk sesi terikat. Readiness berubah tidak siap sejak drain dimulai. Binding strict yang hilang tetap menghasilkan error eksplisit.
- **Belum diputuskan:** nilai final `stop_grace_period`, format log ringkasan shutdown, dan exit code yang dipakai; dicatat pada EP-04-T05 dan EP-06-T03.
- **Acuan:** O-5, C-7, Q-6, ARCHITECTURE §10, EP-04-T05, dan EP-06-T03.

### D-015 Persetujuan seluruh tiket MVP dan P1

- **Status:** ditetapkan pengguna pada 23 November 2026.
- **Dasar:** instruksi eksplisit pengguna: “approve semua tiket”.
- **Keputusan:** seluruh 11 epic EP-01–EP-11 beserta 27 story dan 24 task disetujui dengan status `approved`, termasuk 208 AC child yang tercantum pada tiket saat persetujuan diberikan.
- **Implikasi:** persetujuan tiket tidak menyatakan implementasi selesai, dependensi terpenuhi, atau AC lulus. Gerbang M0–M3, urutan P1 D-012, input TBD, dan keputusan D-001–D-014 tetap berlaku. Persetujuan ini tidak otomatis mengubah status keseluruhan PRD, ARCHITECTURE, ROADMAP, TECHSTACK, atau catatan teknis menjadi approved.
- **Batas tindakan:** pencatatan dilakukan pada tiket lokal dan indeks dokumentasi; tidak mencakup pembuatan tiket di planner eksternal, commit, push, atau rilis.
- **Acuan:** [indeks epic MVP](README.md#indeks-epic-mvp), [indeks epic P1](README.md#indeks-epic-p1), dan bagian Keputusan dan kesiapan pada setiap epic.

### D-016 Provider awal OpenCode Free dan DeepSeek V4.1 Flash Free

- **Status:** ditetapkan pengguna pada 23 November 2026.
- **Keputusan:** provider awal adalah `opencode free`, model `deepseek-v4.1-flash-free`, dengan origin transport `https://opencode.ai`, header `x-opencode-client: desktop`, dan `forceStream: true` sesuai input pengguna. Ini melengkapi pilihan provider/model pada D-007.
- **Implikasi:** origin transport bukan path endpoint generasi. [Katalog resmi](https://opencode.ai/docs/console/models/) mencantumkan model tersebut pada Chat Completions di `/inference/openai/v1/chat/completions`. [Dokumentasi akses](https://opencode.ai/docs/console/api/inference/) membatasi free-tier chat untuk penggunaan dari dalam OpenCode; pemakaian melalui gateway belum terbukti diterima. Pilihan provider bukan bukti kompatibilitas maupun instruksi mengubah kontrak JSON/SSE gateway.
- **Belum dibuktikan:** kelayakan jalur OpenCode → gateway → WARP → upstream, semantik JSON/non-stream ketika input adapter memaksa stream, SDK/runtime persis, tool calling aktual, dan kebutuhan affinity. Header client saja tidak dijadikan bukti akses diterima. Tidak ada model pengganti otomatis atau pelonggaran gerbang M0.
- **Acuan:** O-3, C-1/C-2/C-5, Q-1/Q-3/Q-10, EP-01-T01/T03/T04, serta [hasil pemeriksaan M0](M0-lingkungan-dan-integrasi.md).

### D-017 Validasi stack T02 dapat berjalan independen dari hambatan T01

- **Status:** ditetapkan pengguna pada 23 November 2026.
- **Dasar:** pengguna menyetujui rencana EP-01-T02 untuk menjalankan validasi stack yang independen, dengan blocker T01 tetap tercatat.
- **Keputusan:** evaluasi versi/provenance dan eksperimen kompatibilitas terisolasi dapat dilanjutkan sambil mempertahankan dependensi EP-01-T01 yang belum selesai. Tidak ada penghapusan dependency atau kelulusan M0 yang diasumsikan.
- **Hasil teknis:** [laporan T02](M0-stack-dan-versi.md) mencatat pemilihan Go/Svelte untuk spike, exact version/lock/checksum, digest base image, serta hasil eksperimen. Angka pin adalah hasil evaluasi agent dalam rencana yang diotorisasi, bukan klaim pengguna menyetujui seluruh draft arsitektur atau hasil integrasi upstream.
- **Batas:** fixture berada di direktori sementara; tidak ada generasi upstream, perubahan deployment aktif, atau implementasi gateway/dashboard produk. Batas OpenCode Free dan input transport D-016 tetap memerlukan pembuktian terkait.
- **Acuan:** EP-01-T01/T02, A-1–A-8, TECHSTACK, dan [laporan stack](M0-stack-dan-versi.md).

### D-018 Spike WARP T04 terisolasi dan checklist berdasarkan bukti

- **Status:** ditetapkan pengguna pada 23 November 2026 melalui persetujuan kebutuhan dan rencana T04.
- **Keputusan:** gunakan satu relay fixture baru dengan profil privat tersendiri; jangan memakai atau mengubah container/volume deployment aktif. Native OpenCode diuji melawan fixture lokal sebelum registrasi atau generasi nyata. Model dan transport tetap D-016.
- **Batas tindakan:** maksimal satu POST registrasi Cloudflare dan satu PATCH enable tanpa retry, empat GET non-generasi (trace normal, katalog model, trace setelah tunnel turun, trace setelah relay berhenti), serta satu POST generasi total dengan maksimum 512 token output. Budget persisten direservasi sebelum dispatch; error tidak meresetnya. Startup relay tidak meregistrasi ulang atau menjalankan probe otomatis. Cleanup hanya resource baru milik fixture.
- **Batas klaim:** kegagalan preflight menghalangi request provider; 403/429, stream rusak, atau single batch tidak otomatis meluluskan integrasi/streaming. Tidak ada model/key pengganti, tambahan header untuk mengakali akses, pelonggaran AC, atau kelulusan M0 dari fixture.
- **Checklist:** tambahkan ✅ hanya pada AC T01–T04 yang didukung bukti; tulis lingkup dokumentasi/stack independen/fixture/aktual. Status `approved` dan wording AC dipertahankan.
- **Acuan:** [hasil T04](M0-warp-request.md), [EP-01-T04](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t04), D-006/D-013/D-016/D-017. Persetujuan ini tidak mencakup commit, push, atau rilis.

### D-019 Profil resource dan metode pengukuran provisional

- **Status:** ditetapkan pengguna pada 24 November 2026 melalui persetujuan rencana EP-01-T06, termasuk angka profil dan metode pengukurannya. Pengguna mengizinkan dokumentasi independen ketika T04 belum selesai; T06 dan gerbang M0 tetap belum diterima penuh.
- **Lingkup:** anggaran awal implementasi dan profil uji, bukan default produk final, hasil benchmark, atau persetujuan seluruh draft. Target D-008, polling D-009, fixed/strict D-013, dan drain D-014 tetap berlaku. Laporan rinci serta task pemilik berada pada [gerbang dan pengukuran M0](M0-gerbang-dan-pengukuran.md).
- **Anggaran:** body agregat 64 MiB mencakup seluruh representasi/salinan decode/JSON/remap; buffer stream 8 MiB; state operasional 16 MiB; monitoring 8 MiB. Subtotal 96 MiB menyisakan ruang rancangan 160 MiB terhadap target RSS gateway 256 MiB untuk runtime Go/GC/stack/TLS/HTTP/transport/koneksi dan alokasi lainnya. Ini bukan jaminan RSS atau bukti accounting sudah diterapkan. Batas body individual tetap 10 MiB; admission menghitung byte sebelum alokasi, menolak ketika agregat habis dan melepas reservasi tepat sekali.
- **Kapasitas profil:** 20 request generasi aktif; 64 koneksi masuk dan 32 koneksi keluar total, termasuk idle/probe; 1.024 binding; 32 sesi admin; 2 probe dan 4 pembacaan status paralel. Batas byte dan jumlah berlaku bersama. Snapshot teredaksi maksimal 256 KiB; empat salinan = 1 MiB di dalam bucket monitoring. Tidak menambah antrean generasi atau menggusur binding aktif.
- **Distribusi benchmark:** 5 relay × 5 slot tetap 25 kapasitas. Lima alias fixed dengan primary berbeda masing-masing membawa 4 stream, sehingga admission global 20 dapat dibuktikan tanpa fallback/least-inflight tersembunyi atau klaim model nyata stateless.
- **Q-4:** tiga run, masing-masing 100 pasangan warm-up lalu 1.000 pasangan terukur, urutan AB/BA bergantian. TTFB diamati client yang sama; kontrol memakai SOCKS5/upstream yang sama tanpa gateway, dengan body sintetis utama 64 KiB. Hitung p95 nearest-rank dari selisih latency setiap pasangan pada peringkat `ceil(0,95 × N)`; tiap run harus ≤50 ms. Nilai negatif/outlier tidak dihapus; pair error/timeout tidak dibuang untuk mengklaim run lengkap.
- **Q-5:** satu soak wajib 20 stream kontinu selama 1.800 detik setelah seluruh stream aktif, dengan dashboard tiga tab/polling 5 detik. Total timeout 2.100 detik khusus profil soak tidak menjadi default produk atau mengubah hard shutdown 30 menit. RSS proses gateway disampel tiap 1 detik dan peak OS dicatat bila tersedia; laporkan maksimum serta celah sampling. Kelompok server gateway/relay SOCKS5/upstream fixture dibatasi agregat 2 vCPU/2 GiB; generator/browser di luar kelompok dicatat terpisah. Gap sampel, stream drop, OOM atau target terlampaui menghalangi kelulusan.
- **Q-6:** 30 percobaan per skenario disconnect saat menunggu header dan stream aktif, setiap hasil cancellation ≤2 detik dari deteksi gateway hingga cancellation upstream saat aktif. Timebase harus sebanding; uji 20 disconnect serentak dan release tepat sekali terpisah. Drain 60 detik/hard 30 menit/exit eksplisit serta grace >30 menit tetap D-014.
- **Baseline:** melengkapi D-005 dengan protokol awal sedikitnya tiga episode manual terpisah yang tersedia dan tiga pengulangan per skenario fixture wajib yang relevan. Catat tahap, waktu/zona, versi/jalur, skenario, sampel rencana/aktual, pembilang/penyebut atau nilai+satuan, gagal/tidak tersedia, pemilik dan bukti teredaksi. Ini bukan izin generasi/probe/rotasi baru; batas D-018 dan otorisasi sesi tetap berlaku. Data kurang tetap TBD dan dibawa sebelum M1, tanpa waiver atas kewajiban baseline aktual.
- **Belum diputuskan/dibuktikan:** default produk final untuk timeout/probe/binding/sesi/schema resource, accounting byte, ukuran image/aset produk, serta hasil RSS/performa. Daftar pemilik/task dan tenggat milestone tersedia pada laporan. Baseline aktual, SDK/akses/listener T01, bukti WARP T04 dan penerimaan penuh AC1 T06 masih terbuka. Tidak ada perubahan deployment, request provider, commit, push atau rilis yang diotorisasi keputusan ini.
- **Acuan:** EP-01-T06, EP-06-T03, O-1/O-6, G-1–G-7, Q-4–Q-6/Q-10, dan D-005/D-008/D-009/D-013/D-014/D-018.

### D-020 Konfigurasi dan registry offline EP-02-T01

- **Status:** ditetapkan pengguna pada 24 November 2026 melalui persetujuan kebutuhan dan rencana EP-02-T01.
- **Keputusan:** kerjakan modul konfigurasi/registry dan pengujian offline secara independen sementara EP-01-T06/M0 masih parsial. Penerimaan startup/listener/restart gateway terintegrasi tetap tertunda; dependensi dan gerbang tidak dihapus.
- **Kontrak terbatas:** YAML `version: 1` strict dan bounded; alias unik per upstream; default upstream eksplisit jika lebih dari satu; tahap ini fixed/none dan menolak mode belum didukung. Secret environment/file eksklusif, role terpisah, nilai opaque serta error teredaksi. Rincian schema, caps parser/secret, URL, angka eksplisit, default nonnumerik dan contoh tersedia pada [konfigurasi](konfigurasi.md).
- **Batas:** angka contoh/profil D-019 tidak menjadi default numerik produk final atau bukti enforcement/performa. Snapshot/load baru membuktikan konsistensi offline; startup/restart proses, transport, autentikasi endpoint dan kelulusan M0 belum dibuktikan. Tidak ada perubahan deployment, provider request, commit atau push.
- **Acuan:** EP-02-T01, D-006/D-013/D-019, C-3/C-8/C-10, Q-7/Q-10.

### D-021 Kontrol token dan credential independen EP-02-S02

- **Status:** ditetapkan pengguna pada 24 November 2026 melalui persetujuan kebutuhan dan rencana S02.
- **Keputusan:** lanjutkan kontrol token/role, guard HTTP dan persiapan credential server dengan fixture lokal secara independen. Penerimaan endpoint/restart gateway tetap terbuka; T01/T06/M0 tidak diluluskan. Tidak membuat server, JSON/SSE S03/S04, drain atau browser session EP-05 untuk menutup AC secara semu.
- **Kontrak terbatas:** token opaque printable mengikuti T01, satu field/value Authorization Bearer tanpa duplikat; model dan admin hanya menerima token role masing-masing. Penolakan 401 memakai challenge tetap dan envelope/request ID server. Persiapan outbound mengikat target/key pada satu snapshot, hanya meneruskan Content-Type/Accept terbatas, tanpa client auth/target/trailer/replay metadata atau jaringan default. Detail caps dan API terdapat pada [akses dan credential](akses-dan-credential.md).
- **Privasi:** AC5 membuktikan output/error/log milik gateway tidak menyisipkan secret dan credential client tidak menjadi key upstream. Body upstream tidak di-rewrite; jaminan menghapus refleksi secret yang sengaja dibuat upstream memerlukan keputusan kontrak terpisah. Raw HTTP request outbound tetap objek istimewa yang tidak boleh di-dump.
- **Batas:** pergantian snapshot bukan restart proses; transport/pool/performa, client/provider/WARP, seluruh endpoint dan admin browser belum dibuktikan. Tidak menambah dependency, mengubah schema T01, atau mengotorisasi deployment, provider request, commit/push.
- **Acuan:** EP-02-S02, C-8/Q-7/Q-8, AC-10, D-006/D-013/D-020, EP-05-T01/S02.

## Status pertanyaan terbuka

Seluruh pertanyaan keputusan yang dijadwalkan (O-1–O-10) sudah memiliki jawaban pengguna atau batas terbuka yang tercatat. Provider/model sudah dipilih D-016 dan versi Desktop/CLI bawaan sudah dicatat pada matriks lingkungan. D-019 menetapkan profil resource/metode serta protokol baseline awal; angka baseline aktual, versi SDK/runtime provider terpisah, listener/tailnet, default produk final dan bukti kelayakan integrasi M0 tetap terbuka. Profil bukan hasil pengukuran atau kelulusan M0.

O-1 tetap terbuka untuk angka baseline dan bukti aktual meskipun metode pelaksanaannya sudah ditetapkan D-005/D-019; O-2 terbuka hanya untuk alamat/listener serta detail Tailscale, sedangkan host, CPU image, dan topologi akses sudah ditetapkan D-006; O-3 menetapkan client dan kelas upstream melalui D-007, lalu OpenCode Free/model `deepseek-v4.1-flash-free` melalui D-016; bukti akses gateway dan runtime provider masih terbuka; O-6 menetapkan target performa provisional dan fixture 5×5 (D-008), dilengkapi profil/metode D-019 dengan default produk final belum diputuskan; O-10 menetapkan polling 5 detik dengan stale >15 detik untuk MVP dan menunda push SSE ke P1 (D-009); O-8 menetapkan rilis event-driven tanpa anggaran baru dengan tag `v0.1.0-mvp` (D-010); O-7 menetapkan tidak ada SLA formal dan pemulihan best-effort (D-011); O-9 menetapkan urutan P1 dengan Responses API lebih dulu (D-012); O-4 menetapkan fixed routing pada MVP dengan lifecycle strict diuji lewat fixture (D-013); O-5 menetapkan drain 60 detik dengan batas keras 30 menit yang tercatat (D-014). Task EP-01-T01 belum dinyatakan selesai hanya dari D-001–D-014.
