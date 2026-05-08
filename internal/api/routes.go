package api

import "net/http"

func SetupRoutes(mux *http.ServeMux, webFS http.FileSystem) {
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/extract", extractHandler)
	mux.HandleFunc("/api/generate", generateHandler)
	mux.Handle("/", http.FileServer(webFS))
}
