package api

import (
	"encoding/json"
	"net/http"
)

type LangEntry struct {
	Code  string   `json:"code"`
	Name  string   `json:"name"`
	Group []string `json:"group"`
}

var supportedLangs = []LangEntry{
	{Code: "zh-CN", Name: "简体中文", Group: []string{"common", "asia"}},
	{Code: "zh-TW", Name: "繁体中文", Group: []string{"common", "asia"}},
	{Code: "ja", Name: "日本語", Group: []string{"common", "asia"}},
	{Code: "ko", Name: "한국어", Group: []string{"common", "asia"}},
	{Code: "en", Name: "English", Group: []string{"common"}},
	{Code: "fr", Name: "Français", Group: []string{"common", "europe"}},
	{Code: "de", Name: "Deutsch", Group: []string{"common", "europe"}},
	{Code: "es", Name: "Español", Group: []string{"common", "europe"}},
	{Code: "pt", Name: "Português", Group: []string{"europe"}},
	{Code: "it", Name: "Italiano", Group: []string{"europe"}},
	{Code: "ru", Name: "Русский", Group: []string{"europe"}},
	{Code: "nl", Name: "Nederlands", Group: []string{"europe"}},
	{Code: "pl", Name: "Polski", Group: []string{"europe"}},
	{Code: "sv", Name: "Svenska", Group: []string{"europe"}},
	{Code: "ar", Name: "العربية", Group: []string{"other"}},
	{Code: "he", Name: "עברית", Group: []string{"other"}},
	{Code: "th", Name: "ไทย", Group: []string{"asia", "other"}},
	{Code: "vi", Name: "Tiếng Việt", Group: []string{"asia", "other"}},
	{Code: "id", Name: "Bahasa Indonesia", Group: []string{"asia", "other"}},
	{Code: "ms", Name: "Bahasa Melayu", Group: []string{"asia", "other"}},
	{Code: "tr", Name: "Türkçe", Group: []string{"europe", "other"}},
	{Code: "hi", Name: "हिन्दी", Group: []string{"asia", "other"}},
}

func langsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(supportedLangs)
}
