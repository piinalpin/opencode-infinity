// Package httpapi menerapkan kebijakan HTTP pada batas API/admin.
package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"opencode-infinity/internal/access"
	"reflect"
	"strings"
)

// shortcut: guard komponen; pasang di seluruh route model saat perakitan aplikasi.
func RequireGateway(p *access.Tokens, next http.Handler) (http.Handler, error) {
	return require(p, next, false)
}

// shortcut: guard token admin bukan login browser; sesi/cookie/TTL ditetapkan EP-05-T01/S02.
func RequireAdmin(p *access.Tokens, next http.Handler) (http.Handler, error) {
	return require(p, next, true)
}
func require(p *access.Tokens, next http.Handler, admin bool) (http.Handler, error) {
	if !p.Ready() || nilHandler(next) {
		return nil, errors.New("invalid_auth_handler")
	}
	verify := p.Gateway
	if admin {
		verify = p.Admin
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header)
		if !ok {
			unauthorized(w, admin)
			return
		}
		if admin {
			if !verify(token) {
				unauthorized(w, true)
				return
			}
		} else {
			identity, valid := p.GatewayIdentity(token)
			if !valid {
				unauthorized(w, false)
				return
			}
			r = r.WithContext(access.WithGatewayIdentity(r.Context(), identity))
		}
		next.ServeHTTP(w, r)
	}), nil
}
func nilHandler(h http.Handler) bool {
	if h == nil {
		return true
	}
	v := reflect.ValueOf(h)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
func bearerToken(header http.Header) (string, bool) {
	var value string
	found := false
	for name, values := range header {
		if !strings.EqualFold(name, "Authorization") {
			continue
		}
		if found || len(values) != 1 {
			return "", false
		}
		found = true
		value = values[0]
	}
	if !found || len(value) > 4103 {
		return "", false
	}
	scheme, token, ok := strings.Cut(value, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	for _, b := range []byte(token) {
		if b < 33 || b > 126 {
			return "", false
		}
	}
	return token, true
}

func unauthorized(w http.ResponseWriter, admin bool) {
	var id [16]byte
	// Go1.27 crypto/rand.Read mengisi seluruh byte atau menghentikan proses pada kegagalan RNG.
	_, _ = rand.Read(id[:])
	requestID := "req_" + hex.EncodeToString(id[:])
	code, realm := "invalid_gateway_token", "gateway"
	if admin {
		code, realm = "invalid_admin_token", "admin"
	}
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)
	h.Set("X-Infinity-Error-Source", "gateway")
	h.Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusUnauthorized)
	payload := struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}{RequestID: requestID}
	payload.Error.Message = "Token tidak valid."
	payload.Error.Type = "authentication_error"
	payload.Error.Code = code
	// Error tulis response tidak dicatat dengan request/header rahasia.
	_ = json.NewEncoder(w).Encode(payload)
}
