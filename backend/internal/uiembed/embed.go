// Package uiembed holds the Angular SPA (browser build) embedded into the access binary.
// At image build the real files replace internal/uiembed/browser/*; for local go run a tiny placeholder remains.
package uiembed

import (
	"embed"
	"io/fs"
)

//go:embed all:browser
var Browser embed.FS

// SPARoot returns FS с index.html в корне: go:embed кладёт файлы как browser/<name>.
func SPARoot() (fs.FS, error) {
	if _, err := fs.Stat(Browser, "index.html"); err == nil {
		return Browser, nil
	}
	return fs.Sub(Browser, "browser")
}
