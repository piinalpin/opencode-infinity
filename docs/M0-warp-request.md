# M0 — Request melalui WARP pada host target

Tanggal: 23 November 2026 · Task: [EP-01-T04](tickets/2026-11-22-ep-01-kelayakan-dan-keputusan-mvp.md#ep-01-t04)

Status: **parsial, preflight native belum lulus**. Belum ada registrasi WARP, request model, atau bukti integrasi aktual. Status persetujuan tiket tetap `approved`; T04 dan gerbang M0 belum selesai.

## Lingkup yang disetujui

Pengguna menyetujui satu relay fixture baru yang terisolasi, bukan pemakaian ulang profil/container aktif. Identitas WARP dibuat hanya sesudah pemeriksaan lokal lulus. Batas tindakan pada [D-018](DECISIONS.md#d-018-spike-warp-t04-terisolasi-dan-checklist-berdasarkan-bukti): maksimal satu POST registrasi, satu PATCH enable, empat GET non-generasi, dan satu POST generasi total tanpa retry. Model, origin, dan header mengikuti D-016.

Jalur yang akan dibuktikan adalah OpenCode CLI bawaan Desktop `2.3.0` → penerus loopback test-only → SOCKS5 relay fixture → TLS `opencode.ai` → `/inference/openai/v1/chat/completions`, model `deepseek-v4.1-flash-free`, stream aktif, maksimum 512 token output. Runner hanya terhubung ke jaringan internal; relay memiliki uplink tersendiri. Tidak ada port host yang dipublikasikan, Docker socket, profil relay, atau akun pengguna yang dipasang pada runner.

[Dokumentasi Free](https://opencode.ai/docs/console/api/inference/) membatasi akses dari dalam OpenCode. Karena itu spike memakai CLI native, environment whitelist dan XDG sementara, tanpa membaca config, credential store, atau riwayat pengguna. Header client sendiri tidak dianggap bukti izin akses. [Source rilis 2.3.0](https://github.com/anomalyco/opencode/blob/v2.3.0/packages/core/src/plugin/provider/opencode.ts) memasang key publik untuk mode signed-out; spike tidak membuat key palsu atau mengganti model ketika preflight gagal.

## Artefak dan pemeriksaan lokal

| Artefak | Fungsi dan batas |
|---|---|
| [Kontrol transport](../tests/spike/warp-request/control_test.go) dan [tesnya](../tests/spike/warp-request/control_integration_test.go) | Fixed endpoint/model, nonce, body terbatas, latch persisten sebelum dispatch, TLS hostname, domain SOCKS5, no redirect/direct fallback, stream diteruskan dengan flush |
| [Adapter native](../tests/spike/warp-request/native_test.go) | Konfigurasi sementara, satu langkah agent, tools dibatasi, stdout/stderr dibaca terbatas lalu dibuang; laporan hanya hitungan/kategori |
| [Bootstrap](../tests/spike/warp-request/bootstrap.py) dan [tes sintetis](../tests/spike/warp-request/bootstrap_test.py) | Validasi response registrasi, satu POST + satu PATCH, tanpa retry; key/token dalam memory, profil dikirim melalui stdin ke volume privat baru |
| [Compose](../tests/spike/warp-request/compose.yaml) | Runner non-root/read-only/cap-drop ALL; relay hanya NET_ADMIN; network internal dan uplink terpisah; resource dibatasi |
| [Dockerfile relay](../deploy/microwarp/Dockerfile) dan [entrypoint](../deploy/microwarp/entrypoint.sh) | Image berhasil dibangun; kernel WireGuard belum dijalankan dengan profil; IPv4-only, IPv6 disabled wajib, firewall fail closed, startup tanpa registrasi/probe otomatis |
| [Provenance](m0/ep-01-t04/provenance.json) | Commit, lisensi, base digest, SHA-256 native, effective APK, dan ID image lokal; image lokal belum memiliki RepoDigest hasil publish |
| [Hasil pemeriksaan](m0/ep-01-t04/checks.json) | Command, hasil lokal, kegagalan native, budget aktual, serta batas pembuktian |

Default `go test ./...` hanya menjalankan fixture lokal. Native smoke memerlukan opt-in `T04_NATIVE_BIN` dan dijalankan di runner terisolasi. Pengulangan race tidak menyalakan native, Docker, registrasi, atau provider. Fixture loopback T03 tetap memiliki guard semula; helper SOCKS spike terpisah hanya untuk pengujian ini.

Tes lokal membuktikan persistent latch atomik; CONNECT domain dan TLS; forwarding sebelum barrier terminal; SSE kosong/rusak/truncated ditolak; dead relay, header timeout, POST ambigu, stream putus, 403/429/redirect tidak diulang; cancellation dan perebutan budget konkuren. Metadata handshake berhenti sebelum TLS dan tidak merekam payload. Penghitung `LocalWrites` adalah observasi transport lokal, bukan pengakuan dispatch oleh provider.

Suite lokal terakhir: 27 tes utama lulus (57 termasuk subtest), satu native opt-in dilewati; vet dan format lulus. Dua tes Python lulus. Pengulangan race 50 kali menghasilkan 1.350 tes utama lulus, nol kegagalan/data race. Run race pertama menemukan dua assertion yang keliru menolak respons overload 503; assertion dikoreksi agar memeriksa kategori 409/503 secara tepat, tetap menuntut satu dispatch dan delapan penolakan. Hasil suite lokal ini terpisah dari kegagalan smoke native di bawah.

## Hambatan preflight native

CLI membaca konfigurasi sementara dan kini mengirim request ke penerus loopback, tetapi penerus menolaknya dengan kategori `tools_disabled`. OpenCode masih membawa definisi tool meskipun agent dibatasi. Observasi terakhir: satu request ditolak, nol admission, nol dial relay, nol POST provider. Guard tidak dilonggarkan dan budget generasi belum dipakai.

Penelusuran sebelumnya menemukan kebijakan `integration.use` yang ikut menghapus provider, lalu override `apiKey: false` yang tidak cocok dengan resolusi credential native. Perbaikan mengikuti source rilis yang dipakai. Tiga upaya GREEN belum menghasilkan smoke test yang lulus; diagnosis berikutnya menunggu persetujuan revisi langkah native sesuai skill TDD. Tidak ada hasil generasi atau respons upstream yang disimpan sebagai log/fixture.

## Status acceptance

| AC T04 | Status | Bukti yang masih diperlukan |
|---|---|---|
| AC1 | Belum terbukti | Native smoke lulus, profil baru, `warp=on`, model tersedia, dan satu respons generasi valid melalui relay |
| AC2 | Fixture lokal saja | CONNECT domain `opencode.ai:443` dan verifikasi TLS pada request WARP yang sama |
| AC3 | Fixture lokal saja | Data generasi aktual tiba pada pembacaan terpisah sebelum terminal/EOF; single batch tidak dianggap bukti incremental |
| AC4 | Fixture lokal saja | GET negatif pada relay milik fixture setelah tunnel turun/relay berhenti, tanpa direct fallback; tidak mengklaim fault POST ambigu pada provider nyata |

Tidak ada ✅ pada AC T04. Penanda pada T01–T03 menunjukkan bukti yang sudah tersedia: T01 AC3/AC4 pada tingkat dokumentasi, T02 AC1–AC4 untuk validasi stack independen, T03 AC1–AC5 pada fixture lokal. T01 AC1/AC2, akses Free melalui gateway, JSON/non-stream, detail listener/tailnet, serta angka baseline tetap terbuka.

## Batas reproduksi

Pemeriksaan lokal memakai distribusi resmi Go `1.27.4` dengan `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, dan cache sementara. Jalankan `go test -count=1 -timeout=120s ./...`, `go vet ./...`, serta `python3 -m unittest discover -s tests/spike/warp-request -p '*_test.py'`. Listener loopback memerlukan lingkungan yang mengizinkannya.

Fixture live belum siap dijalankan ulang: orchestrator dan tes live belum diselesaikan karena gate native masih gagal. Command default Compose sengaja mengembalikan `t04_live_not_ready` dengan exit 78; smoke terisolasi telah memverifikasi hasil ini. Jangan menjalankan bootstrap manual untuk melewati gate itu. Mengulang run atau menghapus file budget tidak memberi izin registrasi/generasi tambahan. Tidak ada perubahan pada tujuh container existing, routing host, Tailscale, atau konfigurasi Desktop pengguna. Cache/image build tersisa bukan bukti WARP aktif.

Cleanup memverifikasi label project `opencode-infinity-t04-ce3544de`, menghapus satu runner berstatus `Created` yang belum pernah berjalan beserta jaringan internal kosongnya. Tidak ada container/network/volume fixture yang tertinggal; jumlah container existing tetap tujuh. Image build lokal dan cache sementara dipertahankan untuk penelusuran, tanpa profil WARP atau credential pengguna.
