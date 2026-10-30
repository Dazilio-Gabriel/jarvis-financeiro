package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// dist recebe o build do frontend no Dockerfile: em producao o mesmo binario serve a API e as
// telas, na mesma origem (sem CORS). Em desenvolvimento so tem o LEIA.txt e o Vite serve as telas.
//
//go:embed all:dist
var oDist embed.FS

func frontend() http.Handler {
	loArqu, _ := fs.Sub(oDist, "dist")
	loServ := http.FileServerFS(loArqu)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lcCami := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if lcCami == "" {
			lcCami = "index.html"
		}

		// pasta tambem cai no index: o FileServer listaria o conteudo dela
		if loInfo, err := fs.Stat(loArqu, lcCami); err != nil || loInfo.IsDir() {
			// rota do React (/dashboard, /chat...): devolve o index e o router do navegador resolve
			if _, errI := fs.Stat(loArqu, "index.html"); errI != nil {
				http.Error(w, "frontend nao embutido neste binario; em desenvolvimento use o Vite (npm run dev)", http.StatusNotFound)
				return
			}
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}

		// assets do Vite tem hash no nome: podem ficar em cache para sempre; o index nunca
		if strings.HasPrefix(lcCami, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		loServ.ServeHTTP(w, r)
	})
}
