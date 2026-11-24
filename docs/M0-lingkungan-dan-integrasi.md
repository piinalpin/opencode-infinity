# M0 — Lingkungan dan kebutuhan integrasi awal

Tanggal pemeriksaan: 23 November 2026  
Status: Hasil parsial; provider/model dipilih, kelayakan gateway masih terbuka  
Task: [EP-01-T01](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t01)

Dokumen ini mencatat bukti lingkungan, kebutuhan integrasi, sumber baseline G-1–G-7, serta kondisi yang menahan penerimaan M0. Acuan keputusan tetap [DECISIONS](DECISIONS.md), khususnya D-005–D-007, D-013, dan D-016. Persetujuan rencana pengerjaan tidak mengisi nilai TBD atau membuktikan kompatibilitas upstream.

## 1. Matriks lingkungan

Nilai teramati berikut berasal dari pemeriksaan baca-saja pada host pengembangan. Versi terpasang merupakan bukti lingkungan saat pemeriksaan, bukan pilihan versi stack gateway atau klaim kompatibilitas yang sudah lulus EP-01-T02.

| Komponen | Nilai teramati atau keputusan | Sumber bukti dan batasnya |
|---|---|---|
| OS host | CachyOS, Linux `7.3.4-1-cachyos` | `/etc/os-release` dan `platform.release()` |
| CPU host | `x86_64` | `platform.machine()`; sesuai target image `linux/amd64` pada D-006 |
| Docker server | `29.9.3`, OS `linux`, arsitektur `amd64` | Field `Version`, `Os`, dan `Arch` dari `docker version --format '{{json .Server}}'` |
| Docker Compose | `5.7.1` | `docker compose version --short`; sesuai catatan D-006 |
| Tailscale | `1.104.7`, backend `Running` | `tailscale version` dan field `BackendState` dari status lokal; status bukan bukti gateway dapat dijangkau |
| DNS tailnet | Nama DNS perangkat sudah tersedia | Hanya keberadaan `Self.DNSName` diperiksa; hostname deployment final tetap TBD |
| OpenCode Desktop | Aplikasi `@opencode/desktop` versi `2.3.0`; paket `opencode-desktop-bin 2.3.0-1` | Metadata `/usr/lib/opencode-desktop/package.json` dan kepemilikan executable dari `pacman -Qo /usr/bin/opencode-desktop`; belum diuji melalui browser atau percakapan |
| CLI bawaan Desktop | `2.3.0` | Executable `/usr/lib/opencode-desktop/opencode-cli --version` mengeluarkan `v2.3.0` dengan exit 0; penanda `opencode-cli.version` juga `2.3.0` |
| CLI pada PATH | `opencode` tidak ditemukan | Pemeriksaan executable pada PATH sesi agent; CLI bawaan tercatat terpisah dan belum membuktikan integrasi AC-01 |
| Runtime provider client | Indikasi `@opencode/ai/providers/openai-compatible`; versi package terpisah TBD | Entrypoint ada pada konfigurasi v2 yang diperiksa dan marker executable CLI `2.3.0`; ini belum membuktikan exact version package atau runtime aktif untuk model Free |

Pemeriksaan Docker daemon dan status Tailscale mula-mula tidak tersedia di sandbox. Pemeriksaan baca-saja di luar sandbox berhasil. CLI bawaan juga diperiksa di luar sandbox; verifikasi ini hanya memeriksa versi. Tidak ada generasi, probe health/generasi upstream, perubahan listener, atau tindakan lifecycle container yang dilakukan. Pembacaan dokumentasi serta katalog publik melalui GET dicatat terpisah pada bagian 3.

Output status lengkap, konfigurasi lengkap, credential, data akun, dan hostname tailnet tidak disalin ke dokumen. Keluaran yang dicatat hanya field lingkungan yang diperlukan. Verifikasi versi CLI tidak membuktikan SDK provider, endpoint, streaming, atau tool calling.

## 2. Matriks akses dan upstream

Kelas protokol serta topologi mengikuti keputusan pengguna yang sudah ada. D-016 mencatat pilihan eksplisit OpenCode Free dan model `deepseek-v4.1-flash-free`. Nilai transport yang diberikan pengguna dicatat berikut; ini bukan konfigurasi gateway yang sudah diterapkan atau schema konfigurasi OpenCode yang sudah divalidasi.

```json
{
  "transport": {
    "baseUrl": "https://opencode.ai",
    "headers": {
      "x-opencode-client": "desktop"
    },
    "forceStream": true
  },
  "model": "deepseek-v4.1-flash-free"
}
```

| Input | Nilai/status saat ini | Pemilik dan tindak lanjut |
|---|---|---|
| Topologi akses | Tailscale di host, HTTPS melalui `tailscale serve`, tanpa sidecar | D-006; pengguna/operator menyiapkan akses |
| Listener host | Loopback menurut D-006; alamat/port final TBD | Pengguna/operator, EP-01-T01; port 8787/8788 pada ARCHITECTURE masih usulan |
| Hostname, sertifikat, dan kebijakan akses tailnet | TBD untuk deployment gateway/admin | Pengguna/operator, EP-01-T01; status Tailscale berjalan belum membuktikan konfigurasi Serve atau otorisasi dashboard |
| Provider/upstream pertama | `opencode free` | Pilihan pengguna D-016; bukan persetujuan bahwa gateway sudah kompatibel |
| Origin transport | `https://opencode.ai` | Input pengguna D-016; prefix API terdokumentasi `https://opencode.ai/inference/openai/v1`, endpoint lengkap `/chat/completions` relatif terhadap prefix tersebut |
| Model ID upstream persis | `deepseek-v4.1-flash-free` | Pilihan pengguna D-016; ID juga tercantum pada katalog resmi |
| Protokol yang dipersyaratkan | Chat Completions kompatibel OpenAI, SSE, dan tool calling | D-007; dukungan model aktual belum diverifikasi |
| SDK/adapter provider dan exact version | Kandidat runtime v2 `@opencode/ai/providers/openai-compatible`; exact version package TBD | Katalog menamai SDK API `@ai-sdk/openai-compatible`; dokumentasi v2 memetakan ke runtime native. Keduanya tidak dianggap otomatis sebagai package/version aktif pada client |
| Akses/auth upstream | Free-tier chat dibatasi untuk penggunaan dalam OpenCode menurut dokumentasi resmi | Mekanisme dan kelayakan jalur melalui gateway belum terbukti; jangan menyimpulkan tanpa auth hanya dari label Free. Uji generasi tetap EP-01-T04 dengan otorisasi operator |
| Routing dan kebutuhan sesi | Fixed routing default; tidak ada klaim strict affinity client nyata | D-007/D-013; sumber yang diperiksa belum membuktikan syarat sesi/identitas pada jalur gateway |

Koreksi pemeriksaan konfigurasi: pembacaan awal hanya memeriksa key `provider` v1. Pemeriksaan ulang pada `~/.config/opencode/opencode.jsonc` menemukan map `providers` v2 dengan entri custom dan package `@opencode/ai/providers/openai-compatible`, sedangkan model default tingkat atas tidak diisi. Entri custom itu bukan bukti bahwa provider Free terpilih atau memakai package/version yang sama. Riwayat percakapan dan penyimpanan credential tidak dibaca untuk menebak pilihan uji.

## 3. Kesesuaian protokol dan batas affinity

Provider dan model telah dipilih. Penilaian dokumentasi pada 23 November 2026 menemukan bahwa format endpoint sesuai kelas Chat Completions, tetapi akses Free dan input pemaksaan stream membatasi klaim kompatibilitas gateway.

| Kebutuhan | Hasil penilaian dokumentasi | Batas bukti |
|---|---|---|
| Model dan Chat Completions | S1 mencantumkan `deepseek-v4.1-flash-free` pada `https://opencode.ai/inference/openai/v1/chat/completions` | Keberadaan pada katalog bukan hasil request melalui gateway/WARP |
| Pembatasan akses | S2 menyatakan free-tier chat hanya dari OpenCode; client lain mendapat HTTP 403, termasuk dengan API key | Jalur OpenCode melalui gateway belum dibuktikan diterima. `x-opencode-client: desktop` saja tidak membuktikan syarat akses terpenuhi |
| JSON/non-stream | Input pengguna `forceStream: true` memerlukan evaluasi terhadap C-2/Q-1 | Jangan menganggap dukungan SSE otomatis memenuhi JSON/non-stream atau mengadopsi agregasi seluruh stream sebagai perilaku gateway |
| Streaming dan tool calling | SSE diminta dalam input transport; metadata runtime menunjukkan dukungan protokol compatible secara umum | Tool-call deltas, usage, cancel, dan JSON model ini tetap membutuhkan pembuktian T03/T04; belum ada hasil generasi |
| Kebutuhan affinity | Fixed routing dan batas D-013 tetap berlaku | Dokumentasi yang diperiksa belum membuktikan kebutuhan identitas jaringan/sesi model ini; statusnya belum diketahui, bukan dinyatakan stateless |
| Responses API | Endpoint model yang terdaftar adalah Chat Completions | Belum ada bukti bahwa model ini mewajibkan Responses; C-11 tetap P1 |

Sumber yang diperiksa:

- **S1:** [OpenCode Console models](https://opencode.ai/docs/console/models/), tabel endpoint dan bagian free models; model gratis ini tersedia untuk waktu terbatas.
- **S2:** [OpenCode Inference API](https://opencode.ai/docs/console/api/inference/), bagian Authentication dan Endpoints. Pembatasan akses adalah temuan dari dokumentasi, bukan hasil 403 request generasi pada pekerjaan ini.
- **S3:** [Source pemetaan runtime v2](https://github.com/anomalyco/opencode/blob/dev/packages/core/src/aisdk-native.ts), diambil melalui Context7 dan diverifikasi pada source resmi. Branch `dev` adalah acuan penjelasan, bukan bukti versi package terpasang pada CLI `2.3.0`.
- **S4:** [Katalog model publik](https://opencode.ai/inference/v1/models) dan [kontrak katalog](https://opencode.ai/docs/console/api/models/). Katalog dibaca melalui alat web; GET metadata terpisah dari eksekutor Python tidak berhasil dan tidak dipakai sebagai bukti kemampuan model. Tidak ada POST generasi atau probe berkuota.

MVP menggunakan fixed routing dan tidak mengklaim strict affinity per percakapan pada client nyata sesuai D-013. Fixed scope bersama tidak memberikan isolasi percakapan atau pengenalan sesi lama setelah restart. Lifecycle strict diuji melalui fixture pada EP-04; plugin sinyal sesi client tetap C-15/P1. Tidak ada perubahan model, credential, protokol, atau cakupan otomatis untuk mengatasi ketidakcocokan.

## 4. Sumber dan pencatatan baseline G-1–G-7

D-005 menetapkan dua tahap: catatan manual penggunaan saat ini, lalu pengukuran terinstrumentasi pada fixture M0. Tabel ini memenuhi kebutuhan penetapan sumber serta pemilik pada AC3 EP-01-T01; angka baseline belum diukur. Pengguna merupakan pemilik pengukuran/operator, dan agent membantu instrumentasi serta perhitungan. Pembaruan rujukan 24 November 2026: [hasil EP-01-T06](M0-gerbang-dan-pengukuran.md#5-baseline-g-1g-7) dan D-019 menetapkan protokol awal minimum tiga episode manual terpisah yang tersedia serta tiga pengulangan per skenario fixture wajib yang relevan. Angka aktual tetap TBD; kekurangan sampel/otorisasi dibawa sebelum M1 dan tidak memberi izin request nyata tambahan.

| ID | Metrik PRD | Sumber awal/manual | Sumber pengukuran M0 yang direncanakan | Baseline aktual | Pemilik/tindak lanjut |
|---|---|---|---|---|---|
| G-1 | Jumlah client wajib yang menyelesaikan percakapan dan tool calling | Catatan client/versi yang benar-benar digunakan serta hasil percakapan dan siklus tool call | Catatan client kontrol dan satu jalur uji M0; integrasi gateway CLI/Desktop tetap dibuktikan ulang di M3 | TBD | Pengguna + agent; pilihan versi/provider pada T01, bukti T03/T04, rekap T06 |
| G-2 | Kelulusan JSON, SSE terfragmentasi, dan tool-call deltas | Catatan format dan kegagalan yang teramati tanpa menyalin konten percakapan | Rekaman assertion fixture JSON/SSE dan urutan deltas pada EP-01-T03/T04 | TBD | Pengguna + agent; fixture T03, observasi jalur T04, rekap T06 |
| G-3 | Kelengkapan metadata request dan status relay | Daftar metadata yang tersedia pada alur penggunaan saat ini dan langkah manual untuk diagnosis | Identitas fixture/relay, fase dispatch, hasil, dan waktu pada T03/T04; metadata produk lengkap diverifikasi M2 | TBD | Pengguna + agent; T03/T04, rekap T06 |
| G-4 | Pengulangan generasi otomatis setelah dispatch ambigu atau stream dimulai | Catatan kegagalan dan pengulangan yang dapat dibuktikan; tanpa bukti diberi status belum diketahui | Counter dispatch fixture pada gangguan sebelum pengiriman, ambigu, dan setelah stream dimulai di T03 | TBD | Pengguna + agent; T03, rekap T06 |
| G-5 | Jumlah perubahan konfigurasi agent saat credential upstream diperbarui | Catatan jumlah langkah/perubahan pada prosedur saat ini; nilai credential tidak dicatat | Catatan perubahan konfigurasi client pada fixture credential sintetis; perilaku gateway akhir diverifikasi M3 | TBD | Pengguna + agent; metode dirinci T06; perubahan credential nyata tidak dilakukan pada T01 |
| G-6 | Kelengkapan observasi identitas jaringan relay dan batasnya | Catatan cara memperoleh IPv4/IPv6 teramati, tujuan, waktu, dan kondisi tidak tersedia | Observasi relay WARP pada T04 bila diotorisasi; duplikasi dan IPv6 tidak tersedia tidak otomatis dianggap gagal | TBD | Pengguna/operator + agent; T04, rekap T06 |
| G-7 | Kelulusan skenario pemantauan browser dan akses dashboard | Catatan cara memantau pool saat ini, langkah manual, serta gangguan pembaruan yang diketahui | Inventaris data pemantauan yang tersedia pada fixture/spike; bukan bukti dashboard produk AC-20–AC-22, yang tetap diuji M2/M3 | TBD | Pengguna/operator + agent; sumber pembanding awal direkap T06, pengujian dashboard EP-05 |

Setiap catatan pengukuran memuat ID G, tahap manual/fixture, tanggal dan zona waktu, versi client, jalur/protokol yang dipakai, skenario, jumlah sampel, nilai beserta satuan atau pembilang/penyebut, hasil gagal/tidak tersedia, dan referensi bukti teredaksi. Waktu penyiapan dan jumlah langkah manual dapat menjadi bukti pendukung masalah; keduanya tidak menggantikan metrik G yang berbeda.

Baseline yang belum tersedia tidak diubah menjadi nol atau angka perkiraan. D-005 tetap meminta baseline M0 sebelum M1; EP-01-T06 harus meninjau kelengkapannya. Bila suatu metrik belum dapat diukur pada fixture yang ada, catat hambatan dan bawa kepada pengguna; jangan menutup gerbang dengan menganggap kelulusan produk sudah terbukti. Catatan awal tidak menggantikan penerimaan produk pada milestone yang ditentukan PRD.

## 5. Hambatan dan gerbang penerimaan

| Kondisi | Tindakan dan pemilik | Dampak pada kelulusan |
|---|---|---|
| Provider/model sudah dipilih tetapi akses Free pada gateway belum terbukti | Pilihan D-016 dan batas OpenCode-only dicatat; agent/operator harus menilai jalur yang dimaksud sebelum uji T04 | M0 tertahan; tidak ada klaim kompatibel atau penggantian model otomatis |
| Exact version runtime provider belum teridentifikasi | Agent melengkapi bukti package yang aktif untuk provider Free; validasi versi terkait EP-01-T02 | Matriks AC1 belum lengkap; versi CLI `2.3.0` tidak dianggap sebagai versi SDK terpisah |
| Model membutuhkan Responses API/protokol native yang tidak tersedia di MVP | Catat sumber, kebutuhan yang tidak terpenuhi, dan kombinasi client/model; minta keputusan produk | Model tidak diklaim kompatibel; M0 tertahan sampai keputusan tercatat, tanpa memasukkan C-11 ke MVP diam-diam |
| Model memerlukan strict affinity client nyata atau identitas absolut yang tidak dapat dibuktikan | Catat batas D-013 dan kebutuhan model; pengguna menentukan tindak lanjut | Kombinasi tidak diklaim didukung; fixed routing/fixture tidak dianggap pengganti bukti kebutuhan tersebut |
| Listener, hostname, sertifikat, atau kebijakan akses final belum dicatat | Pengguna/operator melengkapi matriks bagian 2 | Sisa O-2 tetap terbuka; tidak menahan pembuatan fixture, tetapi harus selesai untuk penutupan M0 sesuai ROADMAP |
| Baseline aktual belum diukur | Pengguna + agent melaksanakan D-005 melalui T03/T04 dan rekap T06 | AC3 T01 boleh memuat TBD berpenanggung jawab; baseline aktual tetap diperlukan sebelum M1 |
| Fixture atau request nyata melalui WARP belum dilakukan | Jalankan task T03/T04 sesuai dependensi serta otorisasi pengujian | Pemeriksaan versi/instalasi pada T01 tidak meluluskan gerbang transport M0 |

Model yang dipilih telah dievaluasi pada tingkat dokumentasi: endpoint Chat Completions tercatat, sedangkan batas akses Free dan JSON/non-stream masih menahan klaim dukungan gateway. Tidak ada smoke test atau kegagalan generasi nyata yang diklaim. Temuan ini menjadi catatan negatif AC4; hasilnya tidak disamakan dengan pembuktian transport T03/T04.

## 6. Pemetaan hasil EP-01-T01

| AC | Hasil saat ini | Bukti/batas |
|---|---|---|
| AC1 | Sebagian | Host/client serta provider/model/transport tersedia; exact version SDK aktif dan detail akses final masih terbuka |
| AC2 | Penilaian dokumentasi tersedia | Bagian 3 menjelaskan Chat Completions, batas akses Free, serta kebutuhan pembuktian sesi/stream; kompatibilitas gateway belum lulus |
| ✅ AC3 | Terpenuhi pada tingkat dokumentasi | Ketujuh G memiliki sumber, pemilik, dan baseline TBD pada bagian 4; tidak ada angka atau hasil benchmark yang direka |
| ✅ AC4 | Terpenuhi pada tingkat dokumentasi | Batas akses Free dan risiko JSON/non-stream dicatat sebagai hambatan M0; tidak ada pelonggaran kontrak atau klaim kelulusan dari katalog |

Task masih berstatus `approved`, dengan hasil pengerjaan parsial. Dokumen ini tidak menyatakan EP-01-T01 selesai atau mengubah status task lain, keputusan D-001–D-016, maupun cakupan MVP/P1.

✅ pada AC3/AC4 menunjukkan bukti dokumentasi tersebut. [Pemeriksaan T04](M0-warp-request.md) menambahkan SHA-256 CLI native `2.3.0` dan percobaan konfigurasi terisolasi; preflight belum lulus dan tidak mengisi exact version SDK terpisah, detail akses final, atau kelayakan gateway yang masih terbuka pada AC1/AC2.
