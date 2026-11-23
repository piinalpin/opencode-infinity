# Keputusan opencode-infinity

Versi: 0.1.11  
Tanggal: 22 November 2026

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
- **Belum diputuskan:** nama provider/upstream dan model persisnya, versi Desktop persis, serta apakah model tersebut membutuhkan strict affinity atau Responses API. Ketidakcocokan yang ditemukan saat M0 dicatat untuk keputusan produk dan tidak boleh membuat M0 ditandai lulus diam-diam.
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

## Pertanyaan berikutnya

Seluruh pertanyaan keputusan yang dijadwalkan (O-1–O-10) sudah memiliki jawaban pengguna atau batas terbuka yang tercatat. Bagian yang masih `TBD` adalah angka baseline, provider/model/versi persis, listener/tailnet, default timeout/probe, dan detail pelaksanaan yang menunggu pengukuran M0. Tidak ada keputusan baru yang diusulkan tanpa kebutuhan nyata dari spike.

O-1 tetap terbuka untuk angka baseline dan bukti aktual meskipun metode pelaksanaannya sudah ditetapkan D-005; O-2 terbuka hanya untuk alamat/listener serta detail Tailscale, sedangkan host, CPU image, dan topologi akses sudah ditetapkan D-006; O-3 menetapkan client dan kelas upstream, tetapi provider/model persis serta versi Desktop masih dicatat saat M0 (D-007); O-6 menetapkan target performa provisional dan fixture 5×5 (D-008); O-10 menetapkan polling 5 detik dengan stale >15 detik untuk MVP dan menunda push SSE ke P1 (D-009); O-8 menetapkan rilis event-driven tanpa anggaran baru dengan tag `v0.1.0-mvp` (D-010); O-7 menetapkan tidak ada SLA formal dan pemulihan best-effort (D-011); O-9 menetapkan urutan P1 dengan Responses API lebih dulu (D-012); O-4 menetapkan fixed routing pada MVP dengan lifecycle strict diuji lewat fixture (D-013); O-5 menetapkan drain 60 detik dengan batas keras 30 menit yang tercatat (D-014). Task EP-01-T01 belum dinyatakan selesai hanya dari D-001–D-014.
