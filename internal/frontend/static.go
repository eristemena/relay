package frontend

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:embed
var embeddedAssets embed.FS

func NewStaticHandler() (http.Handler, error) {
	assets, err := fs.Sub(embeddedAssets, "embed")
	if err != nil {
		return nil, fmt.Errorf("open embedded frontend assets: %w", err)
	}

	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		cleanPath := path.Clean(strings.TrimPrefix(request.URL.Path, "/"))
		if cleanPath == "." || cleanPath == "" {
			serveAsset(responseWriter, request, assets, "index.html")
			return
		}

		candidates := []string{cleanPath, cleanPath + ".html", path.Join(cleanPath, "index.html")}
		for _, candidate := range candidates {
			if serveAsset(responseWriter, request, assets, candidate) {
				return
			}
		}

		serveAsset(responseWriter, request, assets, "index.html")
	}), nil
}

func serveAsset(responseWriter http.ResponseWriter, request *http.Request, filesystem fs.FS, assetPath string) bool {
	body, err := fs.ReadFile(filesystem, assetPath)
	if err != nil {
		return false
	}

	stat, err := fs.Stat(filesystem, assetPath)
	if err != nil || stat.IsDir() {
		return false
	}

	http.ServeContent(responseWriter, request, stat.Name(), stat.ModTime(), bytes.NewReader(body))
	return true
}
