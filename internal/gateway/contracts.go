package gateway

import (
	"net/http"
	"opencode-infinity/internal/config"
)

// Dispatcher adalah efek jaringan server tepercaya. Request memuat key dan tidak boleh di-dump.
// Implementasi wajib SOCKS5 sesuai relay, DNS tujuan lewat proxy, TLS hostname verification,
// tanpa redirect/direct fallback/retry ambigu, dan menghormati Context serta ownership Body.
// Timeout koneksi/header dan batas socket adapter mengikuti snapshot; deadline header berakhir sesudah header,
// sedangkan context request tetap berlaku sepanjang body. S04 membuktikan kontrak ini pada adapter fixture.
// shortcut: adapter jaringan runtime belum tersedia; S03 membuktikan wiring adapter fixture lokal.
type Dispatcher interface {
	Dispatch(*http.Request, config.Relay) (*http.Response, error)
}

// RequestNotSentError marks an error that proves no application request reached the upstream.
// The dispatcher must release the failed attempt's network resources before returning it.
// Other dispatch errors are ambiguous and must never trigger a retry.
type RequestNotSentError interface {
	error
	RequestNotSent() bool
}

// Failure hanya memuat kategori tetap, tanpa input atau error jaringan mentah.
type Failure struct {
	Code   string
	Status int
}

func (f *Failure) Error() string { return f.Code }
