# EP-03-S04 — Kontrak failover tanpa generasi duplikat

Versi: 0.1.0 · Task: [EP-03-S04](tickets/2026-11-22-ep-03-pool-routing-dan-kegagalan.md#ep-03-s04)

## Kontrak dispatch

Coordinator melakukan satu dispatch secara default. Fallback model non-strict hanya boleh dijalankan setelah dispatcher mengembalikan error yang mengimplementasikan `RequestNotSentError` dan `RequestNotSent()` bernilai `true`. Tipe tersebut adalah bukti positif bahwa POST aplikasi belum mencapai upstream; dispatcher juga wajib menutup resource jaringan attempt gagal sebelum mengembalikan bukti. Error umum, timeout menunggu header, respons kosong, atau tidak adanya dispatch yang terlihat bukan bukti.

Coordinator mencoba relay pada `fallback_relays` sesuai urutan konfigurasi. Pool memvalidasi relay tersebut sebagai fallback model, memeriksa eligibility dan kapasitas, lalu memindahkan reservasi secara atomik. Jika tidak ada fallback yang dapat direservasi atau context request sudah dibatalkan, request gagal tanpa koneksi langsung. Jumlah dispatch maksimum dua: satu dispatch awal dan satu fallback.

Fallback hanya dibuat setelah bukti non-delivery. Coordinator membentuk ulang body dari buffer request yang telah dibatasi; `GetBody` tetap nonaktif sehingga transport tidak mendapat izin replay tersembunyi. Kegagalan tanpa bukti tetap ambigu dan tidak diulang.

Setelah response diserahkan kepada handler, coordinator tidak memindahkan JSON atau SSE. Kegagalan saat forwarding membatalkan response aktif; stream pengganti tidak dibuat. Retry client berada di luar hitungan attempt gateway.

## Bukti dan batas

- Unit test memeriksa fallback hanya untuk bukti eksplisit, penolakan bukti false/error ambigu/timeout, batas dua dispatch, serta perpindahan lease dan kapasitas relay.
- Fixture integrasi menolak SOCKS CONNECT primary sebelum tunnel, lalu membuktikan tepat satu dispatch melalui relay fallback dan nol direct dial. Fixture terpisah mengirim POST hingga body dibaca upstream, memutus response sebelum header, dan membuktikan tidak ada dispatch pada relay fallback.
- Ledger fixture menghitung dispatch upstream terpisah dari koneksi SOCKS CONNECT. Body request dan credential tidak disimpan.
- Adapter jaringan produksi belum tersedia pada tree gateway ini. Karena itu, belum ada adapter produksi yang menerbitkan `RequestNotSentError`; timeout header, retry internal adapter/provider, HTTP/2, WARP, dan provider nyata belum dibuktikan. Implementasi ini tidak menyatakan integrasi runtime lulus.

Jalankan verifikasi lokal dengan:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 -timeout=120s ./internal/gateway ./internal/pool ./tests/integration
```
