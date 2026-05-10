package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func SetupRoutes(mux *http.ServeMux, webFS http.FileSystem) {
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/langs", langsHandler)
	mux.HandleFunc("/api/extract", extractHandler)
	mux.HandleFunc("/api/generate", generateHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		f, err := webFS.Open("index.html")
		if err != nil {
			http.Error(w, fmt.Sprintf("internal error: %v", err), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, f)
	})
}
