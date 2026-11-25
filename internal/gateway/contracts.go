package gateway

import (
	"net/http"
	"opencode-infinity/internal/config"
)

// Dispatcher adalah efek jaringan server tepercaya. Request memuat key dan tidak boleh di-dump.
// Implementasi wajib SOCKS5 sesuai relay, DNS tujuan lewat proxy, TLS hostname verification,
// tanpa redirect/direct fallback/retry ambigu, dan menghormati Context serta ownership Body.
// shortcut: adapter jaringan runtime belum tersedia; S03 membuktikan wiring adapter fixture lokal.
type Dispatcher interface {
	Dispatch(*http.Request, config.Relay) (*http.Response, error)
}

// Failure hanya memuat kategori tetap, tanpa input atau error jaringan mentah.
type Failure struct {
	Code   string
	Status int
}

func (f *Failure) Error() string { return f.Code }
