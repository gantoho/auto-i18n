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
	{Code: "ar", Name: "Arabic", Group: []string{"america", "other"}},       // 阿根廷
	{Code: "bg", Name: "Bulgarian", Group: []string{"europe"}},              // 保加利亚
	{Code: "cn", Name: "Simplified Chinese", Group: []string{"asia"}},       // 中文简体
	{Code: "de", Name: "German", Group: []string{"europe"}},                 // 德国
	{Code: "es", Name: "Spanish", Group: []string{"europe"}},                // 西班牙
	{Code: "fa", Name: "Farsi", Group: []string{"asia", "other"}},           // 波斯语
	{Code: "fr", Name: "French", Group: []string{"europe"}},                 // 法国
	{Code: "id", Name: "Indonesian", Group: []string{"asia"}},               // 印尼
	{Code: "in", Name: "Hindi", Group: []string{"asia"}},                    // 印度
	{Code: "it", Name: "Italian", Group: []string{"europe"}},                // 意大利
	{Code: "jp", Name: "Japanese", Group: []string{"asia"}},                 // 日本
	{Code: "kr", Name: "Korean", Group: []string{"asia"}},                   // 韩国
	{Code: "lv", Name: "Latvian", Group: []string{"europe"}},                // 拉脱维亚
	{Code: "my", Name: "Malay", Group: []string{"asia"}},                    // 马来西亚
	{Code: "nl", Name: "Dutch", Group: []string{"europe"}},                  // 荷兰
	{Code: "pl", Name: "Polish", Group: []string{"europe"}},                 // 波兰
	{Code: "pt", Name: "Portuguese", Group: []string{"europe"}},             // 葡萄牙
	{Code: "ro", Name: "Romanian", Group: []string{"europe"}},               // 罗马尼亚
	{Code: "ru", Name: "Russian", Group: []string{"europe"}},                // 俄罗斯
	{Code: "th", Name: "Thai", Group: []string{"asia"}},                     // 泰国
	{Code: "tw", Name: "Traditional Chinese", Group: []string{"asia"}},      // 中文繁体
	{Code: "vn", Name: "Vietnamese", Group: []string{"asia"}},               // 越南
}

func langsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(supportedLangs)
}
