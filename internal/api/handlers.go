package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"auto_i18n/internal/extractor"
	"auto_i18n/internal/generator"
	"auto_i18n/internal/xlsx"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func extractHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("read file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	jsonData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("read file content: %v", err), http.StatusBadRequest)
		return
	}

	langsStr := strings.TrimSpace(r.FormValue("langs"))
	splitTagsStr := r.FormValue("splitTags")
	stripTagsStr := r.FormValue("stripTags")
	var langs []string
	if langsStr != "" {
		for _, l := range strings.Split(langsStr, ",") {
			l = strings.TrimSpace(l)
			if l != "" {
				langs = append(langs, l)
			}
		}
	}

	tmpDir, err := os.MkdirTemp("", "auto-i18n-*")
	if err != nil {
		http.Error(w, fmt.Sprintf("create temp dir: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	jsonPath := filepath.Join(tmpDir, header.Filename)
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("write temp file: %v", err), http.StatusInternalServerError)
		return
	}

	ext := extractor.New(jsonPath)
	if splitTagsStr == "true" && stripTagsStr == "true" {
		http.Error(w, "split-tags 和 strip-tags 不能同时使用", http.StatusBadRequest)
		return
	}
	ext.SplitTags = splitTagsStr == "true"
	ext.StripTags = stripTagsStr == "true"
	result, err := ext.Run()
	if err != nil {
		http.Error(w, fmt.Sprintf("extract: %v", err), http.StatusInternalServerError)
		return
	}

	if len(result.Entries) == 0 {
		http.Error(w, "no translatable content found", http.StatusBadRequest)
		return
	}

	values := make([]string, len(result.Entries))
	for i, e := range result.Entries {
		values[i] = e.Value
	}

	nameLangs := make([]string, len(langs))
	for i, l := range langs {
		nameLangs[i] = CodeToName(l)
	}

	xlsxPath := filepath.Join(tmpDir, strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))+".xlsx")
	writer := xlsx.NewWriter(xlsxPath)
	if result.SplitMeta != nil {
		if err := writer.WriteWithMeta(result.SourceLang, values, nameLangs, result.SplitMeta); err != nil {
			http.Error(w, fmt.Sprintf("write xlsx: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		if err := writer.Write(result.SourceLang, values, nameLangs); err != nil {
			http.Error(w, fmt.Sprintf("write xlsx: %v", err), http.StatusInternalServerError)
			return
		}
	}

	xlsxData, err := os.ReadFile(xlsxPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx: %v", err), http.StatusInternalServerError)
		return
	}

	downloadName := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)) + ".xlsx"
	w.Header().Set("X-I18n-Count", fmt.Sprintf("%d", len(result.Entries)))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	w.Write(xlsxData)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}

	xlsxFile, xlsxHeader, err := r.FormFile("xlsx")
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx file: %v", err), http.StatusBadRequest)
		return
	}
	defer xlsxFile.Close()

	jsonFile, jsonHeader, err := r.FormFile("json")
	if err != nil {
		http.Error(w, fmt.Sprintf("read json file: %v", err), http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	xlsxData, err := io.ReadAll(xlsxFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx content: %v", err), http.StatusBadRequest)
		return
	}
	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("read json content: %v", err), http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "auto-i18n-*")
	if err != nil {
		http.Error(w, fmt.Sprintf("create temp dir: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	xlsxPath := filepath.Join(tmpDir, xlsxHeader.Filename)
	jsonPath := filepath.Join(tmpDir, jsonHeader.Filename)
	if err := os.WriteFile(xlsxPath, xlsxData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("write xlsx: %v", err), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("write json: %v", err), http.StatusInternalServerError)
		return
	}

	reader := xlsx.NewReader(xlsxPath)
	sheetData, err := reader.Read()
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx: %v", err), http.StatusBadRequest)
		return
	}
	if len(sheetData.TargetLangs) == 0 {
		http.Error(w, "xlsx must have at least 2 language columns", http.StatusBadRequest)
		return
	}

	gen := generator.New(xlsxPath, jsonPath, tmpDir)
	gen.DirPattern = r.FormValue("dirPattern")
	if err := gen.Run(); err != nil {
		http.Error(w, fmt.Sprintf("generate: %v", err), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	filepath.Walk(tmpDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || path == jsonPath {
			return nil
		}
		rel, _ := filepath.Rel(tmpDir, path)
		if rel == filepath.Base(jsonPath) {
			return nil
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}
		fh, _ := zip.FileInfoHeader(fi)
		fh.Name = filepath.ToSlash(rel)
		fh.Method = zip.Deflate
		w, _ := zw.CreateHeader(fh)
		data, _ := os.ReadFile(path)
		w.Write(data)
		return nil
	})
	zw.Close()

	langsStr := strings.Join(sheetData.TargetLangs, ",")
	w.Header().Set("X-I18n-Count", fmt.Sprintf("%d", len(sheetData.TargetLangs)))
	w.Header().Set("X-I18n-Langs", langsStr)

	if gen.Result != nil {
		completionParts := make([]string, 0, len(gen.Result.LangStats))
		warnParts := make([]string, 0)
		for _, s := range gen.Result.LangStats {
			completionParts = append(completionParts, fmt.Sprintf("%s:%.0f", s.Lang, s.CompletionPct))
			if len(s.PHWarnings) > 0 {
				warnParts = append(warnParts, fmt.Sprintf("%s:%d", s.Lang, len(s.PHWarnings)))
			}
		}
		w.Header().Set("X-I18n-Completion", strings.Join(completionParts, ","))
		if len(warnParts) > 0 {
			w.Header().Set("X-I18n-PH-Warn", strings.Join(warnParts, ","))
		}
	}

	downloadName := "translations.zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	w.Write(buf.Bytes())
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}

	xlsxFile, _, err := r.FormFile("xlsx")
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx file: %v", err), http.StatusBadRequest)
		return
	}
	defer xlsxFile.Close()

	jsonFile, _, err := r.FormFile("json")
	if err != nil {
		http.Error(w, fmt.Sprintf("read json file: %v", err), http.StatusBadRequest)
		return
	}
	defer jsonFile.Close()

	xlsxData, err := io.ReadAll(xlsxFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx content: %v", err), http.StatusBadRequest)
		return
	}
	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("read json content: %v", err), http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "auto-i18n-*")
	if err != nil {
		http.Error(w, fmt.Sprintf("create temp dir: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	xlsxPath := filepath.Join(tmpDir, "check.xlsx")
	jsonPath := filepath.Join(tmpDir, "source.json")
	if err := os.WriteFile(xlsxPath, xlsxData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("write xlsx: %v", err), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("write json: %v", err), http.StatusInternalServerError)
		return
	}

	reader := xlsx.NewReader(xlsxPath)
	sheetData, err := reader.Read()
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx: %v", err), http.StatusBadRequest)
		return
	}

	entries, err := extractor.ExtractEntries(jsonData)
	if err != nil {
		http.Error(w, fmt.Sprintf("parse json: %v", err), http.StatusBadRequest)
		return
	}

	entryValues := make([]string, len(entries))
	entryPaths := make([]string, len(entries))
	for i, e := range entries {
		entryValues[i] = e.Value
		entryPaths[i] = e.KeyPath
	}

	type RowCheck struct {
		Row       int      `json:"row"`
		Source    string   `json:"source"`
		Found     bool     `json:"found"`
		KeyPath   string   `json:"keyPath,omitempty"`
		Targets   []string `json:"targets"`
		FilledCnt int      `json:"filledCnt"`
		TotalCols int      `json:"totalCols"`
	}

	results := make([]RowCheck, 0, len(sheetData.Rows))
	matched := 0
	totalRows := 0
	totalFilled := 0
	totalCells := 0

	for i, row := range sheetData.Rows {
		src := row[0]
		if src == "" {
			continue
		}
		totalRows++

		found := false
		foundPath := ""
		for j, ev := range entryValues {
			if ev == src {
				found = true
				foundPath = entryPaths[j]
				break
			}
		}
		if found {
			matched++
		}

		targets := make([]string, 0)
		filled := 0
		for ci := 1; ci < len(row); ci++ {
			targets = append(targets, row[ci])
			if row[ci] != "" {
				filled++
			}
		}
		totalFilled += filled
		totalCells += len(row) - 1

		results = append(results, RowCheck{
			Row:       i + 2,
			Source:    src,
			Found:     found,
			KeyPath:   foundPath,
			Targets:   targets,
			FilledCnt: filled,
			TotalCols: len(row) - 1,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sourceLang":  sheetData.SourceLang,
		"targetLangs": sheetData.TargetLangs,
		"totalRows":   totalRows,
		"matchedRows": matched,
		"matchRate":   float64(matched) / float64(totalRows) * 100,
		"fillRate":    float64(totalFilled) / float64(totalCells) * 100,
		"totalFilled": totalFilled,
		"totalCells":  totalCells,
		"results":     results,
	})
}
