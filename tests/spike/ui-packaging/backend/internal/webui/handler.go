package webui

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"io/fs"
	"net/http"
	"strings"
)

func NewHandler(assets fs.FS, secret []byte) (http.Handler, error) {
	if len(secret) < 24 || len(secret) > 256 {
		return nil, errors.New("invalid_admin_secret")
	}
	for _, b := range secret {
		if b < 33 || b > 126 {
			return nil, errors.New("invalid_admin_secret")
		}
	}
	for _, pattern := range []string{"index.html", "assets/*.js", "assets/*.css"} {
		names, err := fs.Glob(assets, pattern)
		if err != nil || len(names) == 0 {
			return nil, errors.New("incomplete_ui_assets")
		}
		for _, name := range names {
			info, err := fs.Stat(assets, name)
			if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
				return nil, errors.New("incomplete_ui_assets")
			}
		}
	}
	files := http.FileServerFS(assets)
	expectedUser := sha256.Sum256([]byte("admin"))
	expectedSecret := sha256.Sum256(secret)
	// shortcut: Basic auth stateless khusus fixture lokal; sesi produksi ditetapkan EP-05-T01.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		user, password, ok := r.BasicAuth()
		gotUser := sha256.Sum256([]byte(user))
		gotSecret := sha256.Sum256([]byte(password))
		valid := subtle.ConstantTimeCompare(gotUser[:], expectedUser[:]) & subtle.ConstantTimeCompare(gotSecret[:], expectedSecret[:])
		if !ok || valid != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="T05 fixture", charset="UTF-8"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		asset := strings.HasPrefix(name, "assets/") && fs.ValidPath(name) && !strings.Contains(strings.TrimPrefix(name, "assets/"), "/") && (strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".css"))
		if r.URL.Path != "/" && !asset {
			http.NotFound(w, r)
			return
		}
		if asset {
			info, err := fs.Stat(assets, name)
			if err != nil || !info.Mode().IsRegular() {
				http.NotFound(w, r)
				return
			}
		}
		files.ServeHTTP(w, r)
	}), nil
}
