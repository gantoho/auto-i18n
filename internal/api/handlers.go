package api

import (
	"archive/zip"
	"bytes"
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

	xlsxPath := filepath.Join(tmpDir, strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))+".xlsx")
	writer := xlsx.NewWriter(xlsxPath)
	if err := writer.Write(result.SourceLang, values, langs); err != nil {
		http.Error(w, fmt.Sprintf("write xlsx: %v", err), http.StatusInternalServerError)
		return
	}

	xlsxData, err := os.ReadFile(xlsxPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx: %v", err), http.StatusInternalServerError)
		return
	}

	downloadName := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)) + ".xlsx"
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
	if err := gen.Run(); err != nil {
		http.Error(w, fmt.Sprintf("generate: %v", err), http.StatusInternalServerError)
		return
	}

	base := strings.TrimSuffix(jsonHeader.Filename, filepath.Ext(jsonHeader.Filename))
	parts := strings.Split(base, "_")
	prefix := strings.Join(parts[:len(parts)-1], "_")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files, _ := filepath.Glob(filepath.Join(tmpDir, prefix+"_*.json"))
	for _, f := range files {
		if f == jsonPath {
			continue
		}
		fi, _ := os.Stat(f)
		if fi == nil {
			continue
		}
		fh, _ := zip.FileInfoHeader(fi)
		fh.Name = filepath.Base(f)
		fh.Method = zip.Deflate
		w, _ := zw.CreateHeader(fh)
		data, _ := os.ReadFile(f)
		w.Write(data)
	}
	zw.Close()

	downloadName := prefix + "_translations.zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
	w.Write(buf.Bytes())
}


