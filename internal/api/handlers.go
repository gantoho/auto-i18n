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
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"

	"auto_i18n/internal/extractor"
	"auto_i18n/internal/generator"
	"auto_i18n/internal/xlsx"
)

var (
	htmlTagRegex         = regexp.MustCompile(`<[^>]*>`)
	htmlCommentRegex     = regexp.MustCompile(`(?is)<!--.*?-->`)
	htmlLineBreakRegex   = regexp.MustCompile(`(?i)<br\s*/?>`)
	htmlBlockRegex       = regexp.MustCompile(`(?i)</?(?:p|div|h[1-6]|li|tr|td|th|blockquote|pre|section|article)[^>]*>`)
	htmlStripBlocksRegex = regexp.MustCompile(`(?is)<(?:script|style)[^>]*>.*?</(?:script|style)>`)
	htmlNavBlocksRegex   = regexp.MustCompile(`(?is)<(?:header|footer|nav|aside)[^>]*>.*?</(?:header|footer|nav|aside)>`)
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

	xlsxData, err := io.ReadAll(xlsxFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx content: %v", err), http.StatusBadRequest)
		return
	}

	webText := strings.TrimSpace(r.FormValue("webtext"))
	specifiedColumn := strings.TrimSpace(r.FormValue("column"))

	if webText == "" {
		http.Error(w, "webtext is required", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "auto-i18n-*")
	if err != nil {
		http.Error(w, fmt.Sprintf("create temp dir: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	xlsxPath := filepath.Join(tmpDir, "check.xlsx")
	if err := os.WriteFile(xlsxPath, xlsxData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("write xlsx: %v", err), http.StatusInternalServerError)
		return
	}

	reader := xlsx.NewReader(xlsxPath)
	sheetData, err := reader.Read()
	if err != nil {
		http.Error(w, fmt.Sprintf("read xlsx: %v", err), http.StatusBadRequest)
		return
	}

	// Split web text into segments (by newline)
	rawSegments := strings.Split(webText, "\n")
	segments := make([]string, 0)
	for _, s := range rawSegments {
		s = strings.TrimSpace(s)
		if s != "" {
			segments = append(segments, s)
		}
	}
	if len(segments) == 0 {
		http.Error(w, "webtext has no non-empty segments", http.StatusBadRequest)
		return
	}

	// Build column info
	type colInfo struct {
		Index   int    `json:"index"`
		Header  string `json:"header"`
		ColName string `json:"colName"`
		Match   int    `json:"match"`
		Values  []string
	}

	totalCols := len(sheetData.TargetLangs) + 1 // source + all targets
	cols := make([]colInfo, totalCols)
	for ci := 0; ci < totalCols; ci++ {
		name, _ := excelize.ColumnNumberToName(ci + 1)
		if ci == 0 {
			cols[ci] = colInfo{Index: ci, Header: sheetData.SourceLang, ColName: name}
		} else {
			cols[ci] = colInfo{Index: ci, Header: sheetData.TargetLangs[ci-1], ColName: name}
		}
	}

	// Fill values and count matches
	for ci := 0; ci < totalCols; ci++ {
		vals := make([]string, 0, len(sheetData.Rows))
		for _, row := range sheetData.Rows {
			if ci < len(row) {
				v := strings.TrimSpace(row[ci])
				if v != "" {
					vals = append(vals, v)
				}
			}
		}
		cols[ci].Values = vals

		// Count how many web segments appear in this column's values
		match := 0
		for _, seg := range segments {
			for _, v := range vals {
				if strings.Contains(v, seg) || strings.Contains(seg, v) {
					match++
					break
				}
			}
		}
		cols[ci].Match = match
	}

	// Auto-detect the best matching column
	bestColIdx := 0
	bestMatch := 0
	for ci := 0; ci < totalCols; ci++ {
		if cols[ci].Match > bestMatch {
			bestMatch = cols[ci].Match
			bestColIdx = ci
		}
	}

	// User-specified column override
	if specifiedColumn != "" {
		specUpper := strings.ToUpper(specifiedColumn)
		for ci := 0; ci < totalCols; ci++ {
			if cols[ci].ColName == specUpper ||
				strings.EqualFold(cols[ci].Header, specifiedColumn) {
				bestColIdx = ci
				bestMatch = cols[ci].Match
				break
			}
		}
	}

	// Check each segment against the matched column
	type checkItem struct {
		Segment string `json:"segment"`
		Found   bool   `json:"found"`
	}

	checks := make([]checkItem, 0, len(segments))
	foundCount := 0
	matchedVals := cols[bestColIdx].Values

	for _, seg := range segments {
		found := false
		for _, v := range matchedVals {
			if strings.Contains(v, seg) || strings.Contains(seg, v) {
				found = true
				break
			}
		}
		if found {
			foundCount++
		}
		checks = append(checks, checkItem{Segment: seg, Found: found})
	}

	// Also show values in matched column that aren't found in web text (extra)
	type extraItem struct {
		Value string `json:"value"`
	}
	extras := make([]extraItem, 0)
	for _, v := range matchedVals {
		matched := false
		for _, seg := range segments {
			if strings.Contains(v, seg) || strings.Contains(seg, v) {
				matched = true
				break
			}
		}
		if !matched {
			extras = append(extras, extraItem{Value: v})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalSegments": len(segments),
		"foundCount":    foundCount,
		"matchRate":     float64(foundCount) / float64(len(segments)) * 100,
		"columns":       cols,
		"matchedColumn": bestColIdx,
		"matchedName":   cols[bestColIdx].ColName + " (" + cols[bestColIdx].Header + ")",
		"checks":        checks,
		"extras":        extras,
		"autoDetected":  specifiedColumn == "" && bestMatch > 0,
	})
}

func fetchURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}

	targetURL := strings.TrimSpace(r.FormValue("url"))
	if targetURL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("create request: %v", err), http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AutoI18n/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("fetch url: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("server returned %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		http.Error(w, fmt.Sprintf("read body: %v", err), http.StatusInternalServerError)
		return
	}

	raw := string(body)

	exclude := strings.TrimSpace(r.FormValue("exclude"))
	if exclude != "" {
		doc, err := html.Parse(strings.NewReader(raw))
		if err == nil {
			parts := strings.Split(exclude, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				if strings.HasPrefix(part, ".") {
					cn := part[1:]
					removeNodes(doc, func(n *html.Node) bool {
						return hasClass(n, cn)
					})
				} else if strings.HasPrefix(part, "#") {
					idn := part[1:]
					removeNodes(doc, func(n *html.Node) bool {
						return hasID(n, idn)
					})
				} else {
					removeNodes(doc, func(n *html.Node) bool {
						return n.Type == html.ElementNode && strings.EqualFold(n.Data, part)
					})
				}
			}
			raw = extractTextFromDOM(doc)
		}
	}

	if exclude == "" {
		raw = htmlStripBlocksRegex.ReplaceAllString(raw, "")
		raw = htmlNavBlocksRegex.ReplaceAllString(raw, "")
		raw = htmlCommentRegex.ReplaceAllString(raw, "")
		raw = htmlLineBreakRegex.ReplaceAllString(raw, "\n")
		raw = htmlBlockRegex.ReplaceAllString(raw, "\n")
	}
	raw = htmlTagRegex.ReplaceAllString(raw, "")

	lines := strings.Split(raw, "\n")
	seen := make(map[string]bool)
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isNumeric(line) {
			continue
		}
		normalized := strings.ToLower(line)
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, line)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"text":  strings.Join(result, "\n"),
		"lines": len(result),
	})
}

// ---- DOM helpers for exclude filtering ----

func hasClass(n *html.Node, className string) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, cls := range strings.Fields(attr.Val) {
				if cls == className {
					return true
				}
			}
		}
	}
	return false
}

func hasID(n *html.Node, idName string) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, attr := range n.Attr {
		if attr.Key == "id" && attr.Val == idName {
			return true
		}
	}
	return false
}

func removeNodes(n *html.Node, match func(*html.Node) bool) {
	if n == nil {
		return
	}
	child := n.FirstChild
	for child != nil {
		next := child.NextSibling
		if match(child) {
			n.RemoveChild(child)
		} else {
			removeNodes(child, match)
		}
		child = next
	}
}

func extractTextFromDOM(n *html.Node) string {
	var buf strings.Builder
	extractTextRecursive(n, &buf)
	return buf.String()
}

var domBlockTags = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "li": true, "tr": true,
	"td": true, "th": true, "blockquote": true, "pre": true,
	"section": true, "article": true, "header": true, "footer": true,
	"nav": true, "aside": true, "br": true,
}

func extractTextRecursive(n *html.Node, buf *strings.Builder) {
	if n == nil {
		return
	}
	if n.Type == html.CommentNode {
		return
	}
	if n.Type == html.ElementNode {
		tag := strings.ToLower(n.Data)
		if tag == "script" || tag == "style" {
			return
		}
	}
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	if n.Type == html.ElementNode && domBlockTags[strings.ToLower(n.Data)] {
		buf.WriteByte('\n')
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextRecursive(c, buf)
	}
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
