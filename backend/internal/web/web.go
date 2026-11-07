// Package web отдаёт собранный фронтенд, встроенный в бинарник.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Папку dist заполняет сборка фронтенда (Dockerfile, `make web`). В git лежит только заглушка index.html.
//
//go:embed all:dist
var dist embed.FS

// Handler отдаёт файлы из dist, а для любого другого пути — index.html (маршруты SPA).
func Handler() http.Handler {
	files, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if f, err := files.Open(p); err == nil {
			f.Close()
			if strings.HasPrefix(p, "assets/") {
				// Имена файлов в assets содержат хэш содержимого, их можно кешировать надолго.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
