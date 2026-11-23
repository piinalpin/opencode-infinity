# PRD — opencode-infinity

Versi: 0.1.13  
Tanggal: 22 November 2026  
Status: Draft revisi; belum disetujui  
Ukuran: Full PRD  
Pemilik produk: pengguna sendiri — D-004  
Bahasa: Indonesia

## 1. Ringkasan dan visi

opencode-infinity adalah gateway API yang dapat di-host sendiri untuk menghubungkan coding agent dengan API model AI melalui pool koneksi Cloudflare WARP. Agent memakai custom provider dengan endpoint HTTP/HTTPS biasa dan token gateway. Gateway memilih relay SOCKS5, meneruskan request ke upstream yang dikonfigurasi operator, lalu mengalirkan respons kembali ke agent.

Visinya adalah memusatkan konfigurasi provider dan pengelolaan jalur keluar agar developer dapat menggunakan agent tanpa mengatur container WARP pada setiap client. Operator memantau relay pool melalui web UI dashboard serta memperoleh aturan kegagalan yang menjaga streaming dan konsistensi percakapan.

Penggunaan awal adalah pribadi: pengguna sendiri menjalankan gateway dan pool WARP dalam container di lokal atau homelab. Host uji adalah mesin Linux x86_64 dengan Docker amd64 dan Compose; akses homelab melalui Tailscale di host dengan `tailscale serve` untuk HTTPS (D-006). Upstream MVP tetap memakai Chat Completions yang kompatibel OpenAI dengan client utama OpenCode Desktop (D-007); Responses API dan adapter native Anthropic tetap P1. Nama “infinity” adalah nama proyek; kuota, biaya, akses model, dan batas penggunaan tetap mengikuti upstream.

Referensi utama produk adalah [warp-relay-pool](https://github.com/trefeon/warp-relay-pool), sesuai arahan pengguna. Revisi 0.1.2 mempertahankan ide gateway melalui pool WARP dan memasukkan web UI dashboard pemantauan sebagai kebutuhan produk. Dashboard ditempatkan pada MVP **(agent's call: penempatan fase)**; kebutuhan dashboard dan sumber referensinya merupakan arahan pengguna. Format full PRD dipertahankan **(agent's call: pilihan format)**. Tidak ada implementasi, integrasi client, atau hasil benchmark yang dinyatakan selesai.

## 2. Dasar dokumen dan status keputusan

- **Arahan pengguna:** perbaiki PRD tanpa mengubah ide; gunakan warp-relay-pool sebagai sumber referensi; gateway harus memiliki web UI dashboard untuk memantau relay pool.
- **Dasar revisi:** PRD 0.1.0–0.1.1 dan klarifikasi pengguna pada versi 0.1.2. Kebutuhan dashboard sudah dinyatakan pengguna; rincian dan target penerimaannya tetap berupa draft.
- **Usulan:** penanda **(agent's call)** menunjukkan asumsi, prioritas hasil pemetaan, atau angka usulan yang belum disahkan pengguna. Ini juga berlaku pada usulan yang diwarisi dari versi 0.1.0.
- **Data belum tersedia:** ditulis `TBD` dengan pertanyaan dan pemilik keputusan. Dokumen tidak mengarang baseline, jumlah pengguna, anggaran, atau tanggal rilis.
- **Keputusan pengguna terkini:** [D-001–D-003](DECISIONS.md) menetapkan deployment lokal/homelab yang siap container, akses homelab melalui Tailscale, serta penggunaan awal pribadi; [D-005](DECISIONS.md) menetapkan metode baseline; [D-006](DECISIONS.md) menetapkan host amd64 dan Tailscale di host; [D-007](DECISIONS.md) menetapkan client utama Desktop dan upstream Chat Completions. Detail teknis yang belum dijawab tetap terbuka.
- **Dokumen pendamping:** [Catatan teknis](PRD-technical-notes.md) mempertahankan rincian desain dan seluruh AC-01–AC-19 dari naskah awal, serta menambahkan AC-20–AC-22 untuk dashboard. PRD ini menjadi acuan kebutuhan produk; pilihan implementasi tetap perlu validasi.

## 3. Masalah dan konteks

Developer yang mengoperasikan coding agent membutuhkan endpoint provider biasa, sementara pengelolaan banyak relay menambah konfigurasi dan keputusan operasional pada client. Operator juga perlu membedakan gangguan transport dari kegagalan credential, kuota, atau model upstream.

Masalah yang ditangani:

1. Konfigurasi jalur keluar tersebar di client dan menyulitkan perubahan terpusat.
2. Operator sulit mengetahui relay yang sehat, aktif, penuh, atau gagal beserta penyebabnya.
3. Perpindahan jalur tanpa aturan dapat memutus streaming atau percakapan yang memerlukan jalur konsisten.
4. Penanganan error yang menyamakan jaringan, kuota, dan autentikasi dapat memicu retry yang tidak tepat.
5. Credential upstream perlu dikelola di server secara terpisah dari token akses agent.

Proyek [warp-relay-pool](https://github.com/trefeon/warp-relay-pool) merupakan sumber referensi utama untuk konsep pool WARP, pemisahan relay, serta konteks operasinya. Dokumentasi arsitekturnya menjelaskan listener per relay, koneksi SOCKS5, dan protokol header khusus [S2]. opencode-infinity menyediakan kontrak API provider langsung sehingga agent tidak perlu memilih port WARP atau menyusun header target, disertai web UI dashboard pemantauan relay pool sesuai kebutuhan pengguna.

Referensi menjadi dasar konteks produk, sedangkan perilaku gateway mengikuti kebutuhan dalam PRD ini. Penetapan referensi tidak dengan sendirinya mengubah kontrak retry, affinity, atau cakupan menjadi seluruh fitur proyek asal. Dashboard adalah kebutuhan eksplisit opencode-infinity; sumber referensi tidak digunakan sebagai bukti bahwa dashboard tersebut sudah tersedia atau telah diimplementasikan.

**Bukti kebutuhan:** masalah di atas berasal dari uraian draft awal. Frekuensi kejadian, waktu konfigurasi saat ini, dan contoh kegagalan pengguna belum tersedia: `TBD`, pemilik produk, O-1. Referensi teknis tidak menggantikan validasi kebutuhan pengguna.

**Cara kerja saat ini:** draft menggambarkan pengaturan relay pada client atau adaptasi protokol relay; penggunaan aktual oleh calon pengguna belum dikonfirmasi. **Alasan waktu pengerjaan:** tidak ada tenggat kalender; rilis bersifat event-driven saat gerbang M3 lulus (D-010), sedangkan bukti waktu aktual tetap dicatat melalui O-1.

## 4. Pengguna dan perjalanan utama

| Pengguna | Kebutuhan utama | Hasil yang diharapkan |
|---|---|---|
| Developer yang memakai OpenCode, yaitu pengguna sendiri | Menghubungkan agent ke model tanpa mengelola jalur WARP pada setiap request | Percakapan, streaming, dan tool calling berjalan melalui custom provider |
| Operator deployment pribadi, yaitu orang yang sama | Memantau relay pool melalui web UI dashboard serta mengelola provider, credential, dan maintenance pada satu deployment | Kondisi pool dan penyebab gangguan dapat dipahami dari browser tanpa mengubah konfigurasi setiap client |

Perjalanan utama, diringkas dari US-01–US-07 pada draft awal:

1. Operator menyiapkan gateway, upstream, credential, dan relay; developer menambahkan provider, memilih alias model, lalu menyelesaikan percakapan dan satu siklus tool call.
2. Developer menerima respons streaming; pembatalan dari client menghentikan pekerjaan upstream dan melepaskan kapasitas gateway.
3. Saat request gagal, operator mengetahui kategori penyebab, upstream, dan relay yang digunakan untuk menentukan tindakan berikutnya.
4. Percakapan yang memerlukan konsistensi tetap memakai jalur yang terikat; kehilangan jalur menghasilkan kegagalan eksplisit agar sesi dapat ditangani dengan benar.
5. Operator melakukan drain relay sebelum maintenance atau memperbarui credential di server; stream pada relay lain dan konfigurasi provider client tetap terjaga.
6. Operator membuka dashboard yang terlindungi akses admin, meninjau ringkasan pool, memeriksa relay yang bermasalah, dan melihat waktu pembaruan data untuk menentukan tindakan operasional.

## 5. Tujuan dan ukuran keberhasilan

Seluruh baseline masih `TBD`; tidak ada hasil pengukuran yang diklaim. Target berikut menerjemahkan tujuan dan acceptance criteria awal serta kebutuhan dashboard menjadi ukuran yang dapat diperiksa **(agent's call: rumusan target, belum disetujui)**. Sesuai D-005, baseline diisi dari catatan manual penggunaan saat ini lalu diukur ulang secara terinstrumentasi pada M0; pengguna bertindak sebagai operator/developer, dengan bantuan agent untuk instrumentasi dan perhitungan. Target numerik tetap provisional sampai baseline M0 terkumpul.

| ID | Tujuan | Metrik | Baseline | Target penerimaan | Waktu verifikasi |
|---|---|---|---|---|---|
| G-1 | Integrasi agent sederhana | Jumlah client wajib yang menuntaskan percakapan dan tool calling | TBD | OpenCode CLI dan Desktop keduanya lulus; versi dicatat (AC-01–02) | M3; tanggal TBD |
| G-2 | Transport transparan | Kelulusan skenario JSON, SSE terfragmentasi, dan tool-call deltas | TBD | Seluruh skenario kompatibilitas wajib lulus tanpa perubahan semantik (AC-03–04) | M1, diverifikasi kembali M3 |
| G-3 | Routing dapat ditelusuri | Kelengkapan metadata request dan status relay pada skenario penerimaan | TBD | Upstream, relay, request aktif, dan alasan ketidaktersediaan dapat ditelusuri pada seluruh skenario wajib | M2 |
| G-4 | Kegagalan ditangani konsisten | Jumlah pengulangan generasi otomatis setelah dispatch ambigu atau stream dimulai | TBD | Nol pengulangan pada skenario wajib (AC-08); kegagalan lain mengikuti kategori dan batas retry | M2 |
| G-5 | Konfigurasi terpusat | Jumlah perubahan konfigurasi agent saat credential upstream diperbarui | TBD | Nol perubahan pada agent; request berikutnya berhasil setelah perubahan server diterapkan sesuai prosedur MVP | M3 |
| G-6 | Identitas jaringan jelas | Kelengkapan observasi relay dan penjelasan keterbatasannya | TBD | Setiap relay menampilkan identitas, hasil observasi IPv4/IPv6 atau status tidak tersedia, dan waktu observasi; duplikasi ditampilkan (AC-17) | M2 |
| G-7 | Relay pool dapat dipantau melalui web UI | Kelulusan skenario ringkasan pool, detail relay, pembaruan data, dan akses dashboard | TBD | Seluruh relay terkonfigurasi dapat dipantau dari browser; perubahan status dan kegagalan pembaruan dikenali; akses tanpa otorisasi ditolak (AC-20–AC-22) | M2, diverifikasi kembali M3 |

**Batas keberhasilan:** lulus fixture tidak membuktikan kompatibilitas client nyata. Penambahan relay tidak dihitung sebagai penambahan kuota, hak akses model, atau jumlah IP unik. Standar penerimaan tambahan ditetapkan pada Q-1–Q-11.

## 6. Ruang lingkup dan non-goals

### 6.1 MVP / P0

Cakupan awal dipertahankan: API Chat Completions, JSON/SSE dan tool-call deltas, katalog alias model lokal, satu atau beberapa upstream, pool SOCKS5 WARP, fixed routing sebagai default, opsi least-inflight untuk model stateless yang tervalidasi, health check, circuit breaker, timeout, kapasitas, affinity, drain, pemisahan token, log metadata, metrik, status operasional, serta panduan deployment dan integrasi OpenCode CLI/Desktop. Web UI dashboard pemantauan relay pool ditambahkan ke MVP melalui C-14 **(agent's call: penempatan fase)**, mengikuti kebutuhan eksplisit pengguna.

Batas deployment awal adalah satu deployment pribadi yang siap container di lokal atau homelab. Host uji, image `linux/amd64`, Docker Compose, serta Tailscale di host dengan `tailscale serve` ditetapkan D-006; listener tetap loopback kecuali melalui jalur Tailscale tersebut.

### 6.2 Tahap lanjutan / P1

Kemampuan P1: Responses API setelah uji kontrak terpisah, kompatibilitas protokol header warp-relay-pool, controller maintenance WARP terpisah, plugin session ID dinamis, serta export konfigurasi client dan integrasi secret store OS. Dashboard pemantauan dipindahkan ke MVP. Kemampuan lain pada P1 tetap berada di luar syarat kelulusan MVP.

Hot reload tetap dicatat sebagai kemungkinan pengembangan berikutnya dari naskah awal; prioritas dan fasenya belum ditetapkan. MVP tetap menerapkan perubahan konfigurasi melalui restart gateway.

### 6.3 Di luar MVP

- Residential proxy, IP dedicated, pemilihan negara, dan jaminan IP unik.
- Pembuatan atau rotasi akun/API key untuk menggandakan kuota; penambahan akses model atau penghapusan batas upstream.
- Translasi API Anthropic/Gemini native ke Chat Completions.
- Agent runtime, eksekusi tool, pengelolaan workspace, dan penyimpanan riwayat chat.
- Browser automation untuk autentikasi provider.
- Marketplace publik, billing pelanggan, isolasi tenant penuh, dan deployment multi-host.
- Request hedging, yaitu mengirim generasi yang sama ke beberapa upstream sekaligus.
- Public forward proxy, CONNECT, dan target URL bebas dari client.

**Kondisi yang dianggap gagal meskipun request berhasil:** koneksi langsung diam-diam saat relay gagal, retry yang berpotensi menggandakan generasi, kehilangan affinity tanpa pemberitahuan, kebocoran secret/konten ke log, atau klaim IP/kompatibilitas yang belum dibuktikan. Batas ini dirangkum dari aturan draft awal.

## 7. Kapabilitas dan prioritas

ID kapabilitas dipertahankan untuk keterlacakan. Pemetaan P0 menjadi Must dan P1 menjadi Should adalah **(agent's call: label prioritas)**. C-14 kini menjadi Must pada MVP **(agent's call: prioritas/fase)** untuk memenuhi permintaan dashboard pengguna; ID tidak diubah. Kriteria di bawah menyatakan hasil produk; detail implementasi dan pengujian tetap di catatan pendamping.

| ID | Kapabilitas | Prioritas | Fase | Kriteria keberhasilan |
|---|---|---|---|---|
| C-1 | Developer menghubungkan OpenCode ke gateway dan memilih model | Must | MVP | CLI dan Desktop menyelesaikan percakapan serta tool calling dengan endpoint dan token gateway; versi uji tercatat |
| C-2 | Developer menerima respons JSON dan streaming yang setara dengan upstream | Must | MVP | Konten, tool-call deltas, field tambahan yang didukung, dan usage diteruskan tanpa perubahan semantik atau menunggu stream lengkap |
| C-3 | Operator memusatkan upstream dan katalog alias model | Must | MVP | Alias mengarah ke upstream/model yang dikonfigurasi; client tidak perlu mengetahui credential atau alamat relay |
| C-4 | Operator memakai pool WARP dengan routing dan kapasitas terkendali | Must | MVP | Request hanya memakai relay eligible; fixed menjadi default dan least-inflight terbatas pada model stateless yang tervalidasi; tidak ada direct fallback |
| C-5 | Pengguna memperoleh penanganan kegagalan yang jelas dan konsisten | Must | MVP | Auth, kuota, transport, timeout, dan upstream error dibedakan; dispatch ambigu dan stream aktif tidak memicu generasi ulang otomatis |
| C-6 | Developer mempertahankan jalur percakapan yang memerlukan affinity | Must | MVP | Binding yang didukung menjaga relay dan generasi identitas; kehilangan binding ditolak secara eksplisit; keterbatasan client tanpa sinyal sesi dijelaskan. MVP memakai fixed routing, sedangkan lifecycle strict dibuktikan lewat fixture sampai C-15 tersedia (D-013) |
| C-7 | Operator melakukan maintenance tanpa mengganggu relay lain | Must | MVP | Relay yang di-drain tidak menerima assignment baru; stream relay lain berlanjut; binding strict dan penghentian paksa memerlukan penanganan eksplisit |
| C-8 | Operator memisahkan akses client, admin, dan credential upstream | Must | MVP | Token terpisah berlaku sesuai perannya; credential upstream tetap di server dan dapat diperbarui tanpa mengganti konfigurasi provider client |
| C-9 | Operator memahami kondisi jaringan dan request | Must | MVP | Status, log metadata, dan metrik menunjukkan relay, kesehatan, kapasitas, cooldown, serta observasi IP dengan batas pengamatan yang jelas |
| C-10 | Operator memasang dan mengoperasikan deployment satu host | Must | MVP | Artefak deployment, validasi konfigurasi, prosedur operasional, troubleshooting, serta panduan CLI/Desktop tersedia dan dapat direproduksi |
| C-11 | Developer memakai Responses API | Should | P1 | Kontrak dan uji kompatibilitas Responses API lulus secara terpisah sebelum dinyatakan didukung |
| C-12 | Operator mempertahankan integrasi stack relay lama | Should | P1 | Protokol header warp-relay-pool dapat dipakai melalui integrasi yang diuji tanpa menjadi dependensi wajib MVP |
| C-13 | Operator mengotomasi maintenance WARP | Should | P1 | Controller terpisah melakukan drain dan pemeriksaan perubahan identitas tanpa melanggar binding aktif |
| C-14 | Operator memantau relay pool melalui web UI dashboard | Must | MVP | Ringkasan pool dan seluruh relay terkonfigurasi dapat ditinjau dari browser, termasuk kesehatan, request aktif/kapasitas, IP teramati, cooldown, alasan gangguan, dan waktu pembaruan; data diperbarui otomatis serta akses dibatasi untuk admin |
| C-15 | Developer memperoleh affinity per percakapan melalui plugin | Should | P1 | Plugin client menyediakan session ID dinamis yang dapat dipakai untuk binding percakapan |
| C-16 | Operator menyederhanakan distribusi konfigurasi dan pengelolaan secret | Should | P1 | Export konfigurasi client dan integrasi secret store OS dapat digunakan; credential upstream tetap dikelola server |

### 7.1 Cakupan pemantauan dashboard

Kebutuhan web UI berasal dari pengguna. Rincian hasil pemantauan berikut merupakan penjabaran dari observabilitas C-9 **(agent's call: rincian cakupan)**:

- **Ringkasan pool:** jumlah relay menurut state, request aktif, dan kapasitas yang tersedia.
- **Kondisi tiap relay:** identitas/generasi relay, state kesehatan dan drain, penggunaan kapasitas, probe terakhir, serta alasan relay tidak dapat dipilih.
- **Identitas jaringan:** IPv4/IPv6 yang teramati, waktu observasi, kondisi alamat tidak tersedia, dan duplikasi antar-relay beserta batas pengamatannya.
- **Kondisi request/upstream:** jumlah request dan error, latensi yang tersedia, serta status cooldown dan kategori kegagalan yang relevan tanpa menampilkan konten percakapan atau secret.
- **Kesegaran informasi:** pembaruan otomatis, waktu pembaruan terakhir, dan penanda ketika data tidak lagi dapat diperbarui. Data lama tidak ditampilkan seolah-olah merupakan hasil pemeriksaan terbaru.

Dashboard menggunakan batas akses admin yang privat. Kebutuhan C-14 mencakup pemantauan; tombol pengubahan konfigurasi atau lifecycle container belum menjadi kebutuhan dashboard yang dinyatakan pengguna. Kemampuan maintenance C-7 tetap berlaku melalui prosedur operasionalnya. Target interval pembaruan dan toleransi keterlambatan masih `TBD`, O-10.

## 8. Standar kualitas

Angka yang diwarisi dari draft 0.1.0 tetap menjadi target rancangan **(agent's call)**, bukan hasil benchmark atau komitmen yang telah disetujui. Q-4, Q-5, dan host benchmark disetujui sebagai target penerimaan provisional melalui D-008, dengan fixture 5 relay × 5 slot. Q-11 menjabarkan kualitas dashboard **(agent's call)**. Semua Q berikut adalah syarat penerimaan MVP **(agent's call: label Must)**.

| ID | Aspek | Target dan cara verifikasi |
|---|---|---|
| Q-1 | Streaming dan transparansi | JSON/SSE serta tool-call deltas mempertahankan makna dan urutan; event pertama diteruskan tanpa menunggu event berikutnya; field tambahan yang didukung tetap lewat |
| Q-2 | Jalur transport | Seluruh request generasi menggunakan SOCKS5 terpilih, resolusi tujuan melalui proxy, dan verifikasi TLS hostname upstream; fixture membuktikan tidak ada direct fallback |
| Q-3 | Retry dan konsistensi | Tidak ada pengulangan saat POST mungkin terkirim atau respons mulai diteruskan; fallback non-strict hanya saat belum terkirim terbukti, maksimal dua attempt total; kehilangan strict binding menghasilkan error |
| Q-4 | Overhead | p95 tambahan ≤50 ms dibanding jalur SOCKS5 yang sama tanpa gateway, diukur memakai upstream lokal dan SOCKS5 fixture dengan konfigurasi benchmark tercatat |
| Q-5 | Kapasitas dan memori | 20 stream bersamaan selama 30 menit pada host 2 vCPU/2 GiB dengan fixture dan kapasitas relay memadai; RSS gateway ≤256 MiB, di luar WARP dan agent |
| Q-6 | Pembatalan dan lifecycle | Pembatalan upstream ≤2 detik setelah disconnect terdeteksi; timeout header, idle stream, dan total terpisah; deadline drain default 60 detik yang hanya menolak admission baru, dengan batas keras 30 menit dan catatan stream/binding yang diputus; tidak ada penghentian paksa diam-diam (D-014) |
| Q-7 | Keamanan akses | Token gateway/admin/upstream terpisah; endpoint model memerlukan token; admin privat; target hanya dari konfigurasi server; redirect tidak diikuti; body default dibatasi 10 MiB termasuk setelah decoding |
| Q-8 | Privasi | Prompt, respons, body tool, secret, authorization, dan raw session ID tidak dicatat secara default; tidak ada penyimpanan riwayat percakapan; operator tetap berada dalam trust boundary |
| Q-9 | Diagnosis | Semua error gateway sebelum stream memakai format seragam dan request ID; error upstream tetap dapat dikenali; status membedakan proses, SOCKS5, HTTPS keluar, dan ketersediaan upstream tanpa menganggap probe sebagai bukti seluruhnya |
| Q-10 | Reproduksibilitas dan kejujuran klaim | Versi client, runtime provider, image/digest, konfigurasi uji, serta benchmark dicatat; IP teramati tidak diklaim identik untuk semua tujuan; usage yang tidak tersedia tidak dilaporkan sebagai nol |
| Q-11 | Keandalan pemantauan web | Nilai dashboard konsisten dengan sumber status gateway pada pengujian terkontrol; pembaruan berjalan otomatis; kegagalan mengambil data ditandai bersama waktu pembaruan terakhir; akses admin diwajibkan dan secret/konten percakapan tidak ditampilkan. Target interval serta keterlambatan maksimum: polling 5 detik dengan stale >15 detik pada MVP, push SSE/WebSocket ditunda ke P1 (D-009) |

Tidak ada SLA ketersediaan produksi formal maupun RTO/RPO yang dijanjikan; pemulihan bersifat best-effort dan waktu aktual dicatat sebagai observasi bila tersedia (D-011). Default probe, timeout, dan konfigurasi rinci tetap tersedia sebagai usulan pada catatan teknis.

## 9. Tahapan dan gerbang penerimaan

M0–M3 adalah urutan pengerjaan dalam fase MVP, bukan rilis yang masing-masing mengurangi cakupan Must. Tanggal seluruh milestone `TBD`; estimasi kalender menunggu M0 dan constraint host/upstream sesuai draft awal.

| Milestone/fase | Fokus | Gerbang penerimaan | Target waktu |
|---|---|---|---|
| M0 — Validasi kelayakan | SOCKS5, DNS, SSE, WARP pada host Linux, upstream/model awal, dan versi OpenCode | Satu request nyata melewati satu WARP; fixture membuktikan tidak ada direct fallback; pilihan uji dan hambatan kompatibilitas tercatat | TBD |
| M1 — Gateway dasar | Bagian dasar C-1–C-3 dan C-8 | AC-03–06, AC-10–11 dan kontrak dasar lulus; integrasi client nyata tetap menjadi gerbang M3 | TBD |
| M2 — Pool dan operasi | C-4–C-7, C-9, dan C-14, beserta kualitas terkait | AC-07–09, AC-12–18, dan AC-20–AC-22 lulus; relay pool dapat dipantau lewat dashboard | TBD |
| M3 — Penerimaan MVP | C-1–C-10 dan C-14 lengkap, Q-1–Q-11, deployment dan dokumentasi | AC-01–AC-22 seluruhnya lulus; G-1–G-7 diverifikasi; versi, hasil, dan batas dukungan tercatat | TBD |
| P1 — Pengembangan lanjutan | C-11, C-12, C-13, C-15, C-16 | Setiap kemampuan menjalani kontrak dan uji penerimaan tersendiri sebelum dinyatakan didukung; urutan detail TBD | TBD |

Gerbang M3 menggabungkan cakupan penerimaan yang sudah ada; pemetaan eksplisit ke G/C/Q adalah **(agent's call: keterlacakan)**. Semua target masih perlu dibuktikan saat implementasi.

## 10. Dependensi dan batasan

| Dependensi/batasan | Implikasi produk | Validasi yang diperlukan |
|---|---|---|
| Upstream kompatibel OpenAI Chat Completions, model, dan credential milik operator | Akses model, kuota, biaya, dan kebijakan data tetap mengikuti upstream | Pilih upstream/model awal dan pastikan protokolnya memenuhi MVP pada M0 |
| Pool WARP dan koneksi SOCKS5 yang disediakan operator | Gateway MVP tidak mengambil alih lifecycle container WARP | Validasi jalur, izin jaringan, dan perilaku IPv4/IPv6 pada host target |
| Dukungan custom provider pada versi OpenCode yang diuji | Kompatibilitas protokol tidak otomatis membuktikan CLI/Desktop berfungsi | Catat versi CLI, Desktop, serta runtime provider dan jalankan integrasi nyata |
| Host Linux tunggal dan Docker Compose sebagai asumsi awal (agent's call) | Dukungan host native multi-OS dan multi-host berada di luar MVP | Konfirmasi host dan arsitektur CPU pada O-2 |
| Akses loopback atau privat sebagai default (agent's call) | Admin dan jalur transport internal tetap privat | Buktikan konfigurasi deployment sesuai batas akses |
| Sumber status gateway untuk web UI dashboard | Dashboard mencerminkan data operasional yang sama dengan gateway, dengan otorisasi admin | Validasi kesesuaian data, pembaruan, dan akses dashboard pada M2 |
| Sinyal sesi dari client untuk strict affinity per percakapan | Tanpa sinyal sesi, fixed routing bersama tidak memberi isolasi percakapan atau pengenalan sesi lama setelah restart | Putuskan kebutuhan model dan integrasi sesi pada O-3–O-4 |
| Penggunaan image/source proyek referensi bersifat opsional | Lisensi dan perilaku proyek referensi tidak otomatis menjadi kontrak gateway | Pin versi/digest, periksa lisensi, dan catat hasil saat komponen diadopsi |

Go, format konfigurasi YAML, dan contoh dua relay tetap merupakan usulan teknis awal **(agent's call)**, bukan syarat produk tambahan. Rincian dipertahankan pada catatan teknis. Client dapat berjalan di OS yang didukung versi agent terkait; dukungan deployment lokal melalui VM Docker perlu validasi terpisah.

## 11. Risiko dan mitigasi

Peluang kejadian belum diukur; tidak ada probabilitas atau tingkat risiko numerik yang diklaim.

| Risiko | Dampak | Mitigasi yang dipertahankan |
|---|---|---|
| Provider menolak jaringan Cloudflare | Relay sehat tetapi request tetap ditolak | Catat ketidakcocokan upstream; jangan menjanjikan WARP sebagai residential proxy |
| Beberapa relay memiliki IPv4 keluar yang sama atau IPv6 tidak tersedia | Jumlah container tidak mewakili jumlah identitas jaringan unik | Tampilkan hasil observasi, duplikasi, dan batasnya |
| Identitas keluar berubah di tengah percakapan | Model tertentu kehilangan konteks atau menolak lanjutan | Strict affinity, generasi identitas, invalidation eksplisit; nyatakan tidak kompatibel bila syarat tidak dapat dibuktikan |
| Batas penggunaan berlaku pada akun/key/project | Menambah atau mengganti relay tidak memulihkan kuota | Klasifikasikan cooldown secara konservatif; tanpa restart WARP atau pergantian key otomatis karena 429 |
| POST telah diproses sebelum timeout | Retry menggandakan generasi atau biaya | Jangan ulang dispatch ambigu; jelaskan interaksi retry client dan gateway |
| Image WARP memerlukan privilege atau host/VM tidak sesuai | Deployment gagal atau butuh akses berlebih | Validasi M0; privilege jaringan dibatasi pada WARP; gateway tanpa Docker socket/NET_ADMIN |
| Client/model pertama membutuhkan Responses API | Chat Completions MVP tidak memenuhi integrasi awal | Validasi sebelum M0 selesai; perubahan prioritas Responses memerlukan keputusan produk, tidak diasumsikan dalam revisi ini |
| Header sesi dinamis tidak tersedia | Affinity per percakapan tidak dapat dibuktikan | Fixed routing dengan batas yang jelas; plugin sesi tetap P1 |
| Source/image referensi berubah | Integrasi dan reproduksibilitas menurun | Pin versi/digest dan periksa lisensi saat adopsi |
| Dashboard menampilkan data lama saat koneksi status gagal | Operator dapat salah menganggap relay masih sehat | Tampilkan waktu pembaruan terakhir serta kegagalan pembaruan; validasi AC-21 |

## 12. Rencana rilis dan pengukuran

Rilis MVP mengikuti gerbang M0–M3 untuk penggunaan pribadi pada container lokal atau homelab. Pengguna awal sudah ditetapkan sebagai pengguna sendiri melalui D-003; penanggung jawab rilis adalah pengguna sendiri, tanpa tanggal kalender atau anggaran baru, dan rilis ditandai `v0.1.0-mvp` saat gerbang M3 lulus (D-010). Tidak ada rencana peluncuran publik yang diasumsikan.

Bukti penerimaan menggunakan fixture upstream/SOCKS5 yang deterministik, pengujian CLI/Desktop nyata, pengujian dashboard melalui browser, dan laporan benchmark yang dapat direproduksi. Smoke test upstream nyata dilakukan secara eksplisit menggunakan credential operator. Fixture tidak menggantikan bukti integrasi client atau perilaku WARP pada host target.

Hasil G-1–G-7 dicatat dalam laporan penerimaan dengan versi dan konfigurasi uji. Log metadata dan metrik operasional mendukung diagnosis; prompt, respons, dan secret tetap tidak dicatat secara default. Struktur laporan ini adalah **(agent's call: cara mendokumentasikan bukti)**.

Paket rilis mencakup binary/image gateway beserta web UI dashboard, Compose contoh minimal dua relay, contoh konfigurasi/secret tanpa credential nyata, prosedur validasi/startup/status/drain/maintenance/shutdown dan akses dashboard, panduan OpenCode CLI/Desktop, troubleshooting, suite acceptance, dan laporan benchmark. Dua relay adalah contoh deployment **(agent's call)**, bukan bukti kapasitas benchmark 20 stream.

## 13. Pertanyaan terbuka

Pemilik berupa peran adalah usulan **(agent's call)**; nama penanggung jawab `TBD`. Pertanyaan di bawah menutup celah keputusan yang belum memiliki bukti tanpa membuka kembali ide produk.

| ID | Pertanyaan/keputusan | Pemilik | Memblokir? |
|---|---|---|---|
| O-1 | Pemilik keputusan produk/target adalah pengguna sendiri sesuai D-004. Apa bukti masalah aktual dan baseline G-1–G-7, serta bagaimana pengukuran teknis dilakukan? | Pengguna sebagai operator/developer; bantuan agent untuk instrumentasi | Metode dan pelaksana ditetapkan D-005; angka baseline tetap TBD sampai M0. Tidak memblokir fixture, menahan klaim dampak dan pengesahan target |
| O-2 | D-001/D-002 sudah menetapkan container lokal/homelab dan akses homelab via Tailscale. CPU image, host/runtime uji aktual, dan topologi akses mana yang dipilih? | Operator/pengguna; D-006 | Host amd64, Docker Compose, dan Tailscale di host ditetapkan D-006. Terbuka untuk alamat/port listener, hostname tailnet, dan detail sertifikat |
| O-3 | Upstream/model dan versi OpenCode CLI/Desktop/runtime provider mana yang diuji? Apakah model memerlukan Responses API atau strict affinity? | Pengguna + agent; D-007 | Client utama Desktop dan kelas upstream Chat Completions ditetapkan D-007. Provider/model persis, versi Desktop, dan kebutuhan strict affinity dicatat saat M0 |
| O-4 | Bagaimana membuktikan lifecycle sesi baru/lanjutan, TTL/kedaluwarsa, serta penolakan sesi strict lama setelah restart dengan kemampuan client yang tersedia? | Pengguna + agent; D-013 | Fixed routing pada MVP; lifecycle strict diuji lewat fixture. Dukungan client menunggu C-15 |
| O-5 | Apa perilaku shutdown setelah deadline drain ketika stream atau binding strict masih aktif, agar tidak bertentangan dengan larangan penghentian paksa diam-diam? | Pengguna + agent; D-014 | Drain 60 detik + batas keras 30 menit dengan catatan eksplisit. Nilai `stop_grace_period` dan exit code dicatat pada EP-04-T05 |
| O-6 | Apakah target performa dan seluruh default usulan sesuai kebutuhan, dan konfigurasi fixture mana yang menyediakan kapasitas untuk benchmark 20 stream? | Pengguna + agent; D-008 | Target Q-4/Q-5 dan fixture 5×5 ditetapkan provisional. Terbuka untuk default timeout/probe final dan ukuran aset/image |
| O-7 | Apakah diperlukan SLA ketersediaan dan target waktu pemulihan produksi; jika ya, berapa targetnya? | Pengguna; D-011 | Tidak ada SLA formal; pemulihan best-effort. Selesai untuk MVP |
| O-8 | Pengguna awal adalah pengguna sendiri sesuai D-003. Siapa penanggung jawab rilis, berapa anggaran, dan kapan target rilis? | Pengguna; D-010 | Penanggung jawab, anggaran, dan model rilis ditetapkan (event-driven, `v0.1.0-mvp`). Catatan rilis dan distribusi ditetapkan EP-06-T05 |
| O-9 | Setelah MVP, bagaimana urutan C-11, C-12, C-13, C-15, C-16; apakah hot reload perlu diprioritaskan? | Pengguna; D-012 | Urutan ditetapkan C-11 → C-15 → C-16 → C-13 → C-12; hot reload tidak diprioritaskan |
| O-10 | Berapa target interval pembaruan dashboard dan keterlambatan maksimum dari data status gateway ke tampilan browser? | Pengguna + agent; D-009 | Polling 5 detik/stale >15 detik ditetapkan untuk MVP; push SSE menjadi P1. Terbuka hanya untuk target P1 push |

## 14. Log keputusan dan perubahan

| Sumber/versi | Catatan | Status |
|---|---|---|
| Arahan pengguna pada revisi 0.1.1 | Perbaiki PRD tanpa mengubah ide | Arahan pengguna |
| PRD 0.1.0 | Gateway HTTP/HTTPS untuk coding agent melalui pool SOCKS5 WARP; pembagian MVP/P1 dan non-goals sebagaimana naskah awal | Dasar draft dipertahankan; bukan bukti persetujuan pengguna terhadap seluruh rincian |
| PRD 0.1.0 | Go, Linux/Compose, dua relay contoh, credential di server, akses privat, serta default dan target angka | Usulan awal (agent's call); belum disahkan |
| PRD 0.1.1 | Full PRD, ID G/C/Q/O, pemetaan prioritas, pemisahan catatan teknis, dan rumusan metrik | Perbaikan dokumen (agent's call); status tetap draft |
| Klarifikasi pengguna pada revisi 0.1.2 | warp-relay-pool menjadi sumber referensi utama; gateway harus memiliki web UI dashboard untuk memantau relay pool | Arahan pengguna |
| PRD 0.1.2 | C-14 menjadi Must/MVP, dengan G-7, Q-11, AC-20–AC-22 dan rincian pemantauan | Penjabaran kebutuhan dan prioritas/fase (agent's call); seluruh draft belum disetujui |
| Klarifikasi pengguna pada revisi 0.1.3 | Deployment lokal atau homelab siap container, akses homelab melalui Tailscale yang disiapkan pengguna, dan penggunaan awal pribadi | D-001–D-003; keputusan eksplisit pengguna, tanpa menyetujui seluruh draft |
| Jawaban O-1 | Pengguna sendiri menjadi pemilik keputusan produk dan penentu target keberhasilan | D-004; bagian bukti/baseline dan pelaksanaan pengukuran tetap terbuka |
| Jawaban O-1 | Baseline diisi dua tahap: catatan manual penggunaan saat ini, lalu pengukuran terinstrumentasi pada M0; pelaksana pengguna dengan bantuan agent | D-005; angka baseline dan target numerik tetap terbuka |
| Jawaban O-2 | Host uji Linux x86_64 (Docker amd64, Compose 5.7.1), image `linux/amd64`, Tailscale di host dengan `tailscale serve`; listener loopback | D-006; listener final dan detail tailnet tetap terbuka |
| Jawaban O-3 | Client utama OpenCode Desktop, upstream Chat Completions kompatibel OpenAI, fixed routing; CLI sebelum M3 | D-007; provider/model/versi persis dicatat saat M0 |
| Jawaban O-6 | Target Q-4 p95 ≤50 ms, Q-5 20 stream × 30 menit pada 2 vCPU/2 GiB, RSS gateway ≤256 MiB; fixture benchmark 5 relay × 5 slot | D-008; provisional, default timeout/probe menunggu pengujian |
| Jawaban O-10 | Dashboard MVP polling 5 detik, stale >15 detik, kegagalan fetch ditandai; push SSE/WebSocket ditunda ke P1 | D-009; target dan auth push P1 masih terbuka |
| Jawaban O-8 | Rilis event-driven saat gerbang M3 lulus, penanggung jawab pengguna, tanpa anggaran baru, tag `v0.1.0-mvp` | D-010; catatan rilis dan distribusi menyusul di EP-06-T05 |
| Jawaban O-7 | Tidak ada SLA formal; pemulihan best-effort, waktu aktual dicatat sebagai observasi | D-011; tidak ada klaim uptime atau alerting wajib |
| Jawaban O-9 | Urutan P1: C-11 → C-15 → C-16 → C-13 → C-12; hot reload tidak diprioritaskan | D-012; penanggung jawab tiap epic dicatat saat EP-07-T01 |
| Jawaban O-4 | MVP fixed routing; lifecycle strict dibuktikan via fixture dengan header sesi eksplisit; client nyata menunggu C-15 | D-013; skema header dan TTL dicatat pada EP-04-T01 |
| Jawaban O-5 | Drain 60 detik hanya menolak admission; stream aktif sampai batas keras 30 menit, diputus dengan catatan eksplisit; `stop_grace_period` lebih besar dari batas | D-014; nilai final grace period dan exit code menyusul |

Riwayat:

- **0.1.0 — 22 November 2026:** draft awal; kebutuhan produk dan rincian desain berada dalam satu dokumen.
- **0.1.1 — 22 November 2026:** struktur, ukuran keberhasilan, keterlacakan, asumsi, gerbang rilis, dan pertanyaan berpenanggung jawab diperjelas; ide dan cakupan dipertahankan; detail teknis beserta AC-01–AC-19 dipindahkan utuh ke dokumen pendamping.
- **0.1.2 — 22 November 2026:** warp-relay-pool ditetapkan sebagai referensi utama sesuai arahan pengguna; dashboard pemantauan C-14 dimasukkan ke MVP; tujuan, kualitas, acceptance, milestone, risiko, dan deliverable diselaraskan tanpa mengubah ID yang sudah ada.
- **0.1.3 — 22 November 2026:** D-001–D-003 dicatat; persona dan target deployment awal diselaraskan; O-2/O-8 tetap terbuka untuk bagian yang belum dijawab.
- **0.1.4 — 22 November 2026:** D-005 dicatat; metode dan pelaksana pengukuran baseline G-1–G-7 ditetapkan, tanpa mengesahkan angka target; O-1 tetap terbuka untuk angka aktual.
- **0.1.5 — 22 November 2026:** D-006 dicatat; host uji amd64, Docker Compose, dan Tailscale di host ditetapkan untuk O-2; asumsi Linux/Compose pada deployment diperbarui.
- **0.1.6 — 22 November 2026:** D-007 dicatat; client utama Desktop dan upstream Chat Completions ditetapkan untuk O-3, tanpa mengesahkan provider/model atau angka target.
- **0.1.7 — 22 November 2026:** D-008 dicatat; target Q-4/Q-5 dan fixture benchmark 5×5 disetujui sebagai target provisional.
- **0.1.8 — 22 November 2026:** D-009 dicatat; dashboard MVP memakai polling 5 detik dengan stale >15 detik, push SSE/WebSocket dipindahkan ke P1.
- **0.1.9 — 22 November 2026:** D-010 dicatat; rilis MVP bersifat event-driven, penanggung jawab pengguna, tanpa anggaran baru, tag `v0.1.0-mvp`.
- **0.1.10 — 22 November 2026:** D-011 dicatat; tidak ada SLA formal, pemulihan best-effort, tanpa klaim uptime.
- **0.1.11 — 22 November 2026:** D-012 dicatat; urutan P1 ditetapkan dengan C-11 lebih dulu dan hot reload tidak diprioritaskan.
- **0.1.12 — 22 November 2026:** D-013 dicatat; MVP memakai fixed routing dan lifecycle strict dibuktikan lewat fixture sampai C-15 tersedia.
- **0.1.13 — 22 November 2026:** D-014 dicatat; drain 60 detik dengan batas keras 30 menit dan catatan eksplisit menggantikan kebijakan shutdown yang tertunda.

## 15. Referensi

Ditinjau kembali pada revisi 22 November 2026. Referensi adalah konteks/integrasi, bukan hasil pengujian opencode-infinity.

- [S1 — warp-relay-pool](https://github.com/trefeon/warp-relay-pool): sumber referensi utama yang ditetapkan pengguna untuk konsep dan konteks operasional relay pool WARP.
- [S2 — Overview: how it works](https://github.com/trefeon/warp-relay-pool/blob/main/docs/01-overview-how-it-works.md): listener per relay, SOCKS5, dan protokol header khusus pada proyek referensi.
- [S3 — OpenCode Providers](https://opencode.ai/docs/providers/): dokumentasi custom provider; konfigurasi spesifik versi dan kompatibilitas CLI/Desktop tetap harus diuji. Pengecekan dokumentasi melalui Context7 tidak menggantikan gerbang M3.
- [Catatan teknis opencode-infinity](PRD-technical-notes.md): rincian versi 0.1.0 yang dipertahankan, termasuk user stories dan AC-01–AC-19, serta tambahan penerimaan dashboard AC-20–AC-22 pada versi 0.1.2.
