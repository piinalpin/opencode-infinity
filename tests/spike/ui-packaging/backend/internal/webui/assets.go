package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

// Build UI dan salin dist sebelum kompilasi; aset hilang harus menggagalkan build.
//
//go:embed dist/index.html dist/assets/*.js dist/assets/*.css
var built embed.FS

func EmbeddedHandler(secret []byte) (http.Handler, error) {
	assets, err := fs.Sub(built, "dist")
	if err != nil {
		return nil, err
	}
	return NewHandler(assets, secret)
}
