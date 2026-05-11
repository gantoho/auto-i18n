package api

import (
	"encoding/json"
	"net/http"

	"auto_i18n/internal/lang"
)

func langsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lang.Supported)
}

func CodeToName(code string) string {
	return lang.CodeToName(code)
}

func NameToCode(name string) string {
	return lang.NameToCode(name)
}
