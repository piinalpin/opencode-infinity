package httpapi

import (
	"net/http"
	"opencode-infinity/internal/gateway"
	"path"
	"strings"
)

// Tolak bentuk yang akan di-redirect ServeMux tanpa memantulkan path/query client.
// Escape dipertahankan agar %2e dan %2f tetap mengikuti matching per segmen.
func canonicalRoutes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped := r.URL.EscapedPath()
		clean := path.Clean(escaped)
		if strings.HasSuffix(escaped, "/") && clean != "/" {
			clean += "/"
		}
		if !strings.HasPrefix(escaped, "/") || clean != escaped {
			writeModelError(w, newRequestID(), &gateway.Failure{Code: "invalid_request", Status: http.StatusBadRequest})
			return
		}
		next.ServeHTTP(w, r)
	})
}
