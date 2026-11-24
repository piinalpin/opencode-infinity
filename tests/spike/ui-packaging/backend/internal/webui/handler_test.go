package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func sampleAssets() fs.FS {
	return fstest.MapFS{
		"index.html":     {Data: []byte(`<html><script src="/assets/app.js"></script><link rel="stylesheet" href="/assets/app.css"></html>`)},
		"assets/app.js":  {Data: []byte(`document.title = "Fixture";`)},
		"assets/app.css": {Data: []byte(`body { color: navy; }`)},
	}
}

func TestLocalAssets(t *testing.T) {
	h, err := NewHandler(sampleAssets(), []byte("synthetic-admin-password-1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, mime, body string }{
		{"/", "text/html", "<html>"},
		{"/assets/app.js", "javascript", "document.title"},
		{"/assets/app.css", "text/css", "color: navy"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.SetBasicAuth("admin", "synthetic-admin-password-1")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Type"), tc.mime) || !strings.Contains(w.Body.String(), tc.body) {
				t.Fatalf("asset status=%d mime=%q", w.Code, w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestAdminAuthorization(t *testing.T) {
	h, err := NewHandler(sampleAssets(), []byte("synthetic-admin-password-1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/", "/assets/app.js", "/assets/app.css", "/missing", "/assets/../index.html", "/%61ssets/app.js?token=ignored"} {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
			for _, credential := range []string{"missing", "wrong", "gateway", "malformed", "wrong-user"} {
				t.Run(method+target+"/"+credential, func(t *testing.T) {
					r := httptest.NewRequest(method, target, nil)
					switch credential {
					case "wrong":
						r.SetBasicAuth("admin", "incorrect-password")
					case "gateway":
						r.Header.Set("Authorization", "Bearer synthetic-gateway-token")
					case "malformed":
						r.Header.Set("Authorization", "Basic !!!")
					case "wrong-user":
						r.SetBasicAuth("gateway", "synthetic-admin-password-1")
					}
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
						t.Fatalf("unauthorized status=%d", w.Code)
					}
					for _, canary := range []string{"synthetic-admin-password", "synthetic-gateway-token", "<html>", "document.title", "color: navy"} {
						if strings.Contains(w.Body.String(), canary) {
							t.Fatal("unauthorized_response_leak")
						}
					}
				})
			}
		}
	}
}

func TestInvalidSecretFailsClosed(t *testing.T) {
	for _, secret := range []string{"", "short", strings.Repeat("x", 257), strings.Repeat("x", 23) + "\n"} {
		if h, err := NewHandler(sampleAssets(), []byte(secret)); err == nil || h != nil {
			t.Fatal("invalid_secret_accepted")
		}
	}
}

func TestOnlyReadOnlyBuiltAssets(t *testing.T) {
	assets := sampleAssets().(fstest.MapFS)
	assets[".env"] = &fstest.MapFile{Data: []byte("private-canary")}
	h, err := NewHandler(assets, []byte("synthetic-admin-password-1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/assets/", 404}, {"GET", "/.env", 404}, {"GET", "/assets/missing.js", 404},
		{"GET", "/assets/app.js.map", 404}, {"GET", "/assets/../index.html", 404},
		{"POST", "/", 405}, {"HEAD", "/assets/app.js", 200},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.SetBasicAuth("admin", "synthetic-admin-password-1")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d", w.Code, tc.status)
			}
			if strings.Contains(w.Body.String(), "private-canary") {
				t.Fatal("private_file_leak")
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing_response_policy")
			}
			if tc.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("head_has_body")
			}
		})
	}
}

func TestIncompleteAssetsFailClosed(t *testing.T) {
	for _, missing := range []string{"index.html", "assets/app.js", "assets/app.css"} {
		assets := sampleAssets().(fstest.MapFS)
		delete(assets, missing)
		if h, err := NewHandler(assets, []byte("synthetic-admin-password-1")); err == nil || h != nil {
			t.Fatal("incomplete_assets_accepted")
		}
	}
}
