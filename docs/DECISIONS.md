# Keputusan opencode-infinity

Versi: 0.1.23
Tanggal: 27 November 2026

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

### D-022 Katalog alias dan JSON independen EP-02-S03

- **Status:** ditetapkan pengguna pada 25 November 2026 melalui persetujuan kebutuhan/opsi 1 dan rencana implementasi S03.
- **Keputusan:** lanjutkan komponen katalog, pemetaan alias dan generasi JSON non-stream dengan empat route model, admission statis dan fixture HTTPS/SOCKS5 lokal. Penerimaan gateway terintegrasi dan dependensi S02/T01/T06/M0 tetap terbuka.
- **Kontrak terbatas:** default/provider memakai snapshot server dan guard gateway yang sama. Hanya nilai top-level `model` diganti; byte field lain dipertahankan. Kontrol `model`/`stream` ambigu ditolak, `stream:true` belum didukung. Alias tidak dikenal menghasilkan `404 model_not_found`, provider tidak dikenal `404 upstream_not_found`, tanpa fallback default atau dispatch. Status/content type/body/usage upstream diteruskan tanpa remap respons atau usage sintetis.
- **Kapasitas:** reservasi statis request dan buffer dilakukan sebelum alokasi body; slot primary relay dan dispatch outbound diambil sebelum dispatch. Penolakan langsung saat kapasitas habis, tanpa antrean. Lease tetap dipegang sampai caller selesai meneruskan respons dan menutup body, termasuk setelah EOF/error/cancellation.
- **Efek jaringan:** dispatcher server tepercaya wajib diinjeksi, tanpa default HTTP client. SOCKS5/DNS proxy, TLS hostname verification, no redirect/direct fallback/replay ambigu dan cancellation dibuktikan pada wiring adapter fixture; bukan implementasi transport runtime atau health eligibility. Anggaran konservatif dapat menolak request sebelum batas individual bila anggaran agregat tidak cukup.
- **Batas:** tidak mengubah schema T01 atau dependency; tidak membuat listener/executable/deployment gateway, pool dinamis, SSE S04, kontrak error S05 lengkap, dashboard, Responses API atau lifecycle binding. Tidak mengotorisasi provider/WARP nyata, commit atau push. Rincian kontrak dan bukti tersedia pada [model alias dan JSON](model-alias-dan-json.md).
- **Acuan:** EP-02-S03, AC-04, C-1/C-2/C-3, Q-1/Q-7/Q-8, D-020/D-021.

### D-023 Streaming SSE independen EP-02-S04

- **Status:** ditetapkan pengguna pada 25 November 2026 melalui persetujuan kebutuhan/opsi 1 dan rencana implementasi S04.
- **Keputusan:** perluas komponen S03 untuk `stream:true` pada endpoint default/provider, dibuktikan dengan gateway HTTP dan adapter HTTPS/SOCKS5 fixture lokal. Penerimaan gateway terintegrasi, provider/WARP/OpenCode dan M0 tetap terbuka.
- **Transparansi:** teruskan byte SSE bertahap dengan Flush; content/tool-call deltas, usage, field tambahan, komentar, line ending dan Content-Encoding yang masih berlaku dipertahankan. Tidak membangun ulang event, menampung seluruh stream, menambah DONE/envelope/generasi pengganti, atau mengulang POST ambigu. Absent/false dan kontrak non-stream S03 tetap berlaku; pengecualian true pada tahap S03 digantikan dukungan S04 ini.
- **Kapasitas:** reservasi stream copy 32 KiB memakai `stream_buffer_bytes` pada pemilik lease yang sama sebelum alokasi; ukuran copy bukan batas event. True mengambil reservasi sebelum dispatch. SSE tak terduga pada absent/false juga mengambil reservasi setelah header dan sebelum forwarding; bila penuh, tutup respons dan kembalikan 429 tanpa retry. Satu dispatch dapat sudah terjadi pada kasus tersebut.
- **Lifecycle:** header timeout milik adapter mengikuti snapshot pada fixture; idle upstream berlaku hanya selama Read pending, total context tetap berlaku sepanjang request. Write/Flush downstream mempunyai deadline dari idle dan batas total/client yang lebih awal. Callback cancellation/deadline disinkronkan sebelum handler kembali. Lease dilepas tepat sekali sesudah forwarding/caller Close, bukan ketika jaringan baru dibatalkan.
- **Batas:** writer tanpa capability Flush/deadline ditolak eksplisit; kegagalan setelah commit memutus framing HTTP tanpa sukses palsu. Gzip upstream perlu flush compressor dan HTTP untuk membuktikan event pertama. Tidak membuat transport/runtime listener/deployment, health/pool dinamis, binding, dashboard, Responses API, atau kontrak S05 lengkap. Tidak menambah dependency/schema/config key, menjalankan provider/WARP nyata, commit atau push.
- **Acuan:** EP-02-S04, AC-03, C-1/C-2, Q-1/Q-6, D-022; [kontrak dan bukti SSE](streaming-sse.md).

### D-024 Error dan batas request independen EP-02-S05

- **Status:** ditetapkan pengguna pada 26 November 2026 melalui persetujuan kebutuhan dan rencana implementasi S05.
- **Keputusan:** lanjutkan komponen/handler dan fixture HTTPS/SOCKS5 lokal sementara penerimaan gateway terintegrasi serta S03/T01/T06/M0 tetap terbuka. Encoding request tetap absent/`identity`; encoding kompresi ditolak sebelum dispatch, tanpa decoder atau default numerik baru.
- **Boundary API:** guard gateway tetap paling luar. Path tak dikenal menghasilkan `404 endpoint_not_found`; method salah pada katalog/generasi menghasilkan `405 method_not_allowed` dengan `Allow`; path noncanonical ditolak `400 invalid_request` tanpa redirect; endpoint Responses default/provider menghasilkan `501 unsupported_api` tanpa implementasi Responses. Error memakai envelope, request ID acak server dan error source, tanpa refleksi path/query/body/error mentah. Escape per segmen dan HEAD tetap mengikuti kontrak HTTP.
- **Target dan upstream:** target/key/relay tetap dari snapshot server; header `X-Relay-Target`/`X-Relay-Path` ditolak dan query client tidak menjadi selector target. Semua respons generasi 300–399, termasuk 304 tanpa conditional request, ditolak sebagai `502 upstream_redirect_rejected`; body ditutup dan lease dilepas tanpa replay. Ini menggantikan passthrough 3xx historis S03. Status/body error 4xx/5xx tetap opaque, dengan ID/source milik gateway dan hanya satu `Retry-After` valid diteruskan; duplikat ambigu dibuang.
- **Batas:** adapter tepercaya tetap wajib menegakkan SOCKS5/DNS proxy/TLS/no-follow/no-direct/no-retry; fixture bukan transport runtime. Tidak membuat listener/deployment, pool dinamis/cooldown, health T06, binding atau dashboard; tidak mengotorisasi provider/WARP nyata, commit atau push. Error yang dibuat parser HTTP/TLS/server sebelum handler berada di luar pembuktian komponen.
- **Acuan:** EP-02-S05, AC-05/AC-06/AC-11, C-5/C-8, Q-2/Q-7/Q-9, D-021–D-023; [kontrak dan bukti error/batas](error-dan-batas-request.md).

### D-025 Liveness dan readiness dasar independen EP-02-T06

- **Status:** ditetapkan pengguna pada 26 November 2026 melalui persetujuan kebutuhan dan rencana implementasi T06.
- **Keputusan:** lanjutkan owner observasi, snapshot dan handler privat dengan fixture lokal; tanpa scheduler/probe produksi, listener atau klaim gateway terintegrasi. Kedua endpoint `GET /health/live` dan `GET /health/ready` memerlukan token admin; token gateway/upstream tidak memberi akses. Model API tetap terpisah.
- **Scope readiness:** `default_primary_transport` memerlukan konfigurasi valid, kondisi proses accepting dan sedikitnya satu primary mapping model default dengan observasi SOCKS5 serta HTTPS keluar yang independen, sehat dan belum kedaluwarsa. Fallback atau relay upstream lain tidak menggantikannya. Provider/model tetap unknown; saturasi kapasitas dan eligibility/dispatch terpadu tetap EP-03-S01/S02, bukan janji admission berikutnya berhasil.
- **Observasi dan waktu:** sumber server tepercaya memperbarui state hanya melalui owner di package pool. `observed_at`/`valid_until` eksplisit, tanpa default freshness produksi baru; unknown/expired tidak ready. Update invalid/future/lebih tua/konflik ditolak; duplikat identik yang masih valid tidak menaikkan revisi. Revisi dan `state_changed_at` mengikuti update tersimpan, sedangkan `snapshot_at` mengikuti pembacaan. Expiry dihitung tanpa probe atau mutasi saat read. Metadata versi build/revisi berasal dari server dan bounded; nilai absent tetap unknown, waktu wire UTC.
- **Batas kerja dan HTTP:** state/copy/serialisasi dibatasi budget yang sudah ada. Constructor menolak budget komponen yang tidak cukup; `parallel_status` membatasi pembacaan sampai write selesai, dengan 429 tanpa antrean saat penuh. State disalin sebelum encoding/write agar client lambat tidak menahan owner. Liveness 200 saat handler berfungsi; readiness 200 atau `503 gateway_not_ready` dengan envelope serta snapshot. Error path/method/canonical memakai kontrak S05; HEAD tanpa body.
- **Batas pembuktian:** observasi fixture dikumpulkan eksplisit dan tidak dipicu endpoint. T06 tidak mengubah admission generasi, menjalankan drain, membuat dashboard/status lengkap, deployment atau transport/probe runtime. Guard admin tidak membuktikan binding listener privat. Integrasi OpenCode/provider/WARP, RSS/performa, dependensi S05/T01/EP-01-T06 dan gerbang M0 tetap terbuka. Tidak mengotorisasi commit, push atau request provider nyata.
- **Acuan:** EP-02-T06, C-9/C-10, Q-9/Q-10, D-021/D-024; [kontrak dan bukti health dasar](health-dasar.md).

### D-026 Admission fixed independen EP-03-S01

- **Status:** ditetapkan pengguna pada 26 November 2026 melalui pilihan cakupan lokal independen dan persetujuan rencana implementasi S01.
- **Keputusan:** lanjutkan komponen admission dan fixture HTTPS/SOCKS5 lokal sementara EP-02-S05, EP-01-T06/M0 serta penerimaan produk terintegrasi tetap terbuka. Primary fixture yang tersedia menjadi prasyarat bukti lokal; relay terkonfigurasi tidak dinyatakan sehat hanya karena tercantum dalam registry.
- **Reservasi:** persiapan body mengambil pending lease dan byte konservatif sebelum alokasi/read. `pending + active` dibatasi bersama oleh `max_active_requests`, tanpa queue atau default baru. Setelah mapping server diketahui, satu operasi pool memeriksa dan mengambil gateway/primary/outbound serta buffer stream deklaratif secara atomik. Promosi gagal tidak mengambil sebagian slot admitted. Katalog tetap lokal, bounded dan berbagi batas global tanpa slot relay/outbound.
- **Kapasitas dan error:** saturasi global, body, relay, outbound dan stream deklaratif menghasilkan `429 gateway_capacity_exceeded` sebelum dispatch; kapasitas relay penuh tidak lagi disatukan dengan `503 no_eligible_relay`. SSE tak terduga pada absent/false tetap mengambil stream budget setelah header dan dapat menolak setelah satu dispatch, sesuai D-023. Error upstream tetap opaque; tidak ada fallback, retry atau cooldown baru.
- **Finalisasi:** pending maupun admitted dilepas tepat sekali. Lease respons tetap hidup sampai forwarding selesai/caller Close, termasuk setelah EOF/error/cancellation; cleanup jaringan tidak membebaskan buffer yang masih dipakai caller.
- **Batas:** tidak menghubungkan observasi readiness sebagai precheck eligibility, membuat health/mode/circuit S02, least-inflight, fallback, cooldown, strict binding atau runtime/deployment baru. Accounting byte konservatif bukan bukti RSS; slot outbound bukan seluruh socket idle/probe. Default final, provider/WARP, performa dan gerbang M0 tetap terbuka. Tidak mengotorisasi commit atau push.
- **Acuan:** EP-03-S01, AC-18, C-4/Q-3/Q-5, D-019/D-022/D-023/D-024/D-025; [kontrak dan bukti admission](admission-fixed.md).

### D-027 Eligibility health dan mode independen EP-03-S02

- **Status:** ditetapkan pengguna pada 27 November 2026 melalui persetujuan cakupan lokal independen dan rencana implementasi S02.
- **Keputusan:** lanjutkan komponen serta fixture lokal S02 sementara penerimaan produk S01/T06/M0 tetap parsial. Satu owner Pool Manager menggabungkan health/circuit, mode dan reservasi atomik; handler model/admin dirakit dari snapshot serta owner yang sama, tanpa jalur capacity-only opsional.
- **Policy:** transport SOCKS5/HTTPS unknown/stale atau circuit belum closed menolak generasi baru. Probe tidak mengubah serving/draining/disabled; mode non-serving dan accepting=false menolak assignment baru tanpa membatalkan stream admitted. Availability upstream ID + alias yang unknown tidak dianggap sehat dan tidak memblokir transport sehat; circuit availability open yang fresh hanya memblokir scope terkait, sedangkan expiry kembali unknown. Tidak ada probe generasi berbayar otomatis.
- **Profil bukti lokal:** threshold 3 gagal/2 sukses; interval 30 detik dengan jitter ±20%, timeout 5 detik, freshness 90 detik, backoff dengan delay maksimal 120 detik. Parameter wajib eksplisit pada constructor; ini bukan default produksi, yang tetap TBD pada S02 sebelum M2.1 diterima.
- **Resource dan status:** scheduler lokal memiliki dua job transport per relay dan worker bounded, berbagi ledger outbound dengan generasi; adapter fixture menghormati Context dan menutup socket sebelum release. Snapshot teredaksi memisahkan mode/observasi/circuit/scoped availability; read tidak memicu probe. Readiness tetap default_primary_transport, bukan klaim provider atau admission selalu berhasil. State/snapshot/monitoring memperhitungkan registry relay dan model.
- **Batas:** tidak mengimplementasikan least-inflight, fallback/retry/cooldown, binding, maintenance/shutdown penuh, dashboard, listener/deployment/transport runtime, provider/WARP nyata atau benchmark/RSS. Tidak mengotorisasi commit/push. Hasil AC dan command berada pada [kontrak eligibility](eligibility-relay.md).
- **Acuan:** EP-03-S02, AC-16, C-4/C-9, Q-2/Q-9, D-025/D-026, [arsitektur](ARCHITECTURE.md#eligibility-dan-kapasitas).

## Status pertanyaan terbuka

Seluruh pertanyaan keputusan yang dijadwalkan (O-1–O-10) sudah memiliki jawaban pengguna atau batas terbuka yang tercatat. Provider/model sudah dipilih D-016 dan versi Desktop/CLI bawaan sudah dicatat pada matriks lingkungan. D-019 menetapkan profil resource/metode serta protokol baseline awal; angka baseline aktual, versi SDK/runtime provider terpisah, listener/tailnet, default produk final dan bukti kelayakan integrasi M0 tetap terbuka. Profil bukan hasil pengukuran atau kelulusan M0.

O-1 tetap terbuka untuk angka baseline dan bukti aktual meskipun metode pelaksanaannya sudah ditetapkan D-005/D-019; O-2 terbuka hanya untuk alamat/listener serta detail Tailscale, sedangkan host, CPU image, dan topologi akses sudah ditetapkan D-006; O-3 menetapkan client dan kelas upstream melalui D-007, lalu OpenCode Free/model `deepseek-v4.1-flash-free` melalui D-016; bukti akses gateway dan runtime provider masih terbuka; O-6 menetapkan target performa provisional dan fixture 5×5 (D-008), dilengkapi profil/metode D-019 dengan default produk final belum diputuskan; O-10 menetapkan polling 5 detik dengan stale >15 detik untuk MVP dan menunda push SSE ke P1 (D-009); O-8 menetapkan rilis event-driven tanpa anggaran baru dengan tag `v0.1.0-mvp` (D-010); O-7 menetapkan tidak ada SLA formal dan pemulihan best-effort (D-011); O-9 menetapkan urutan P1 dengan Responses API lebih dulu (D-012); O-4 menetapkan fixed routing pada MVP dengan lifecycle strict diuji lewat fixture (D-013); O-5 menetapkan drain 60 detik dengan batas keras 30 menit yang tercatat (D-014). Task EP-01-T01 belum dinyatakan selesai hanya dari D-001–D-014.
