package web

import (
	"embed"
	"io/fs"
)

//go:embed all:app
var assets embed.FS

// GetAppFS returns an fs.FS sub-filesystem scoped to the app/ directory.
func GetAppFS() (fs.FS, error) {
	return fs.Sub(assets, "app")
}

// ReadAppFile reads a file directly from the embedded app/ directory.
func ReadAppFile(name string) ([]byte, error) {
	return assets.ReadFile("app/" + name)
}
