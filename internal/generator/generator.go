package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/sjson"

	"auto_i18n/internal/extractor"
	"auto_i18n/internal/lang"
	"auto_i18n/internal/tagsplit"
	"auto_i18n/internal/xlsx"
)

func jsonEncodeString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}

func formatJSON(raw string) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "  "); err != nil {
		return nil, err
	}
	result := buf.Bytes()
	result = bytes.TrimRight(result, "\n\r\t ")
	result = append(result, '\n')
	return result, nil
}

type LangStats struct {
	Lang          string
	Total         int
	Translated    int
	Missing       int
	CompletionPct float64
	PHWarnings    []string
}

type RunResult struct {
	LangStats []LangStats
	TotalWarn bool
}

type Generator struct {
	XLSXPath   string
	JSONPath   string
	OutputDir  string
	DirPattern string
	Result     *RunResult
}

func New(xlsxPath, jsonPath, outputDir string) *Generator {
	return &Generator{
		XLSXPath:  xlsxPath,
		JSONPath:  jsonPath,
		OutputDir: outputDir,
	}
}

func (g *Generator) Run() error {
	reader := xlsx.NewReader(g.XLSXPath)
	data, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read xlsx: %w", err)
	}

	if len(data.TargetLangs) == 0 {
		return fmt.Errorf("xlsx must have at least 2 language columns")
	}

	sourceJSON, err := os.ReadFile(g.JSONPath)
	if err != nil {
		return fmt.Errorf("read original json: %w", err)
	}

	entries, err := extractor.ExtractEntries(sourceJSON)
	if err != nil {
		return fmt.Errorf("extract entries from original json: %w", err)
	}

	splitMeta := data.SplitMeta

	if splitMeta != nil {
		expandedCount := g.expandedEntryCount(entries, splitMeta)
		if expandedCount != len(data.Rows) {
			return fmt.Errorf(
				"行数不匹配（拆分标签模式）：源 JSON 展开后应有 %d 条，xlsx 有 %d 行。"+
					"请使用相同的 split-tags 设置重新 extract",
				expandedCount, len(data.Rows),
			)
		}
	} else if len(entries) != len(data.Rows) {
		return fmt.Errorf(
			"行数不匹配：源 JSON 有 %d 条可翻译文案，xlsx 有 %d 行数据。"+
				"可能原因：① xlsx 的源语言列有空行 ② JSON 文件已被修改 ③ xlsx 行被意外删除。请重新执行 extract 生成新的 xlsx",
			len(entries), len(data.Rows),
		)
	}

	totalRows := len(data.Rows)
	sourceStr := string(sourceJSON)
	g.Result = &RunResult{
		LangStats: make([]LangStats, 0, len(data.TargetLangs)),
	}

	for langIdx, langName := range data.TargetLangs {
		modified := sourceStr
		colIdx := langIdx + 1
		count := 0
		missingRows := make([]string, 0)
		phWarnings := make([]string, 0)

		if splitMeta != nil {
			modified, count, missingRows, phWarnings, err = g.applyTranslationsSplit(modified, data, entries, splitMeta, langIdx, colIdx)
		} else {
			modified, count, missingRows, phWarnings, err = g.applyTranslations(modified, data, entries, langIdx, colIdx)
		}
		if err != nil {
			return err
		}

		pct := float64(count) / float64(totalRows) * 100
		stats := LangStats{
			Lang:          langName,
			Total:         totalRows,
			Translated:    count,
			Missing:       totalRows - count,
			CompletionPct: pct,
			PHWarnings:    phWarnings,
		}
		g.Result.LangStats = append(g.Result.LangStats, stats)

		code := lang.NameToCode(langName)
		outPath := g.buildOutputPath(code)
		formatted, err := formatJSON(modified)
		if err != nil {
			return fmt.Errorf("format json for %s: %w", langName, err)
		}
		if err := os.WriteFile(outPath, formatted, 0644); err != nil {
			return fmt.Errorf("write json for %s: %w", langName, err)
		}

		if count == totalRows {
			fmt.Printf("  ✓ %s (100%%)\n", filepath.Base(outPath))
		} else if count > 0 {
			fmt.Printf("  ⚠ %s (%.0f%% - %d/%d, 缺失 %d 条)\n",
				filepath.Base(outPath), pct, count, totalRows, totalRows-count)
		} else {
			fmt.Printf("  ⚠ %s (0%% - 全部未翻译)\n", filepath.Base(outPath))
		}

		if len(missingRows) > 0 && len(missingRows) <= 5 {
			fmt.Printf("    缺失条目: %s\n", strings.Join(missingRows, ", "))
		} else if len(missingRows) > 5 {
			fmt.Printf("    缺失条目: %s ... (共 %d 条)\n", strings.Join(missingRows[:5], ", "), len(missingRows))
		}

		if len(phWarnings) > 0 {
			fmt.Printf("    ⚠ 占位符不匹配 (%d 条): %s\n", len(phWarnings), strings.Join(phWarnings, ", "))
		}
	}

	return nil
}

func (g *Generator) expandedEntryCount(entries []extractor.FlatEntry, meta map[string]tagsplit.SplitMetaEntry) int {
	count := 0
	for _, e := range entries {
		if m, ok := meta[e.KeyPath]; ok {
			count += m.SegCount
		} else {
			count++
		}
	}
	return count
}

func (g *Generator) applyTranslations(modified string, data *xlsx.SheetData, entries []extractor.FlatEntry, langIdx, colIdx int) (string, int, []string, []string, error) {
	var err error
	count := 0
	missingRows := make([]string, 0)
	phWarnings := make([]string, 0)

	for rowIdx, row := range data.Rows {
		translated := ""
		if colIdx < len(row) {
			translated = strings.TrimSpace(row[colIdx])
		}
		if translated == "" {
			missingRows = append(missingRows, entries[rowIdx].KeyPath)
			continue
		}

		entry := entries[rowIdx]
		if entry.HasPlaceholders {
			srcPH := extractor.ExtractPlaceholders(entry.Value)
			tgtPH := extractor.ExtractPlaceholders(translated)
			if !placeholderSetsEqual(srcPH, tgtPH) {
				phWarnings = append(phWarnings, entry.KeyPath)
			}
		}

		modified, err = sjson.SetRaw(modified, entry.KeyPath, jsonEncodeString(translated))
		if err != nil {
			return "", 0, nil, nil, fmt.Errorf("set value at %s: %w", entry.KeyPath, err)
		}
		count++
	}

	return modified, count, missingRows, phWarnings, nil
}

func (g *Generator) applyTranslationsSplit(modified string, data *xlsx.SheetData, entries []extractor.FlatEntry, splitMeta map[string]tagsplit.SplitMetaEntry, langIdx, colIdx int) (string, int, []string, []string, error) {
	var err error
	count := 0
	missingRows := make([]string, 0)
	phWarnings := make([]string, 0)

	rowIdx := 0
	for _, entry := range entries {
		m, isSplit := splitMeta[entry.KeyPath]
		if isSplit {
			segCount := m.SegCount
			translatedSegs := make([]string, segCount)
			allFilled := true

			for si := 0; si < segCount; si++ {
				if rowIdx+si >= len(data.Rows) {
					allFilled = false
					break
				}
				translated := ""
				if colIdx < len(data.Rows[rowIdx+si]) {
					translated = strings.TrimSpace(data.Rows[rowIdx+si][colIdx])
				}
				if translated == "" {
					allFilled = false
					translated = ""
				}
				translatedSegs[si] = translated
			}

			if allFilled {
				var reassembled string
				indices := m.Indices
				if len(indices) == 0 {
					info := tagsplit.Split(entry.Value)
					indices = make([]int, 0, m.SegCount)
					for i, seg := range info.Segments {
						if seg != "" {
							indices = append(indices, i)
						}
					}
				}
				fullSegs := make([]string, len(m.Template))
				for si, idx := range indices {
					if idx < len(fullSegs) {
						fullSegs[idx] = translatedSegs[si]
					}
				}
				reassembled = tagsplit.Reassemble(fullSegs, m.Template)
				modified, err = sjson.SetRaw(modified, entry.KeyPath, jsonEncodeString(reassembled))
				if err != nil {
					return "", 0, nil, nil, fmt.Errorf("set value at %s: %w", entry.KeyPath, err)
				}
				count += segCount
			} else {
				missingRows = append(missingRows, entry.KeyPath)
			}

			rowIdx += segCount
		} else {
			translated := ""
			if rowIdx < len(data.Rows) && colIdx < len(data.Rows[rowIdx]) {
				translated = strings.TrimSpace(data.Rows[rowIdx][colIdx])
			}
			if translated != "" {
				if entry.HasPlaceholders {
					srcPH := extractor.ExtractPlaceholders(entry.Value)
					tgtPH := extractor.ExtractPlaceholders(translated)
					if !placeholderSetsEqual(srcPH, tgtPH) {
						phWarnings = append(phWarnings, entry.KeyPath)
					}
				}
				modified, err = sjson.SetRaw(modified, entry.KeyPath, jsonEncodeString(translated))
				if err != nil {
					return "", 0, nil, nil, fmt.Errorf("set value at %s: %w", entry.KeyPath, err)
				}
				count++
			} else {
				missingRows = append(missingRows, entry.KeyPath)
			}
			rowIdx++
		}
	}

	return modified, count, missingRows, phWarnings, nil
}

func placeholderSetsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ma := make(map[string]int, len(a))
	for _, s := range a {
		ma[s]++
	}
	mb := make(map[string]int, len(b))
	for _, s := range b {
		mb[s]++
	}
	for k, v := range ma {
		if mb[k] != v {
			return false
		}
	}
	return true
}

func (g *Generator) buildOutputPath(langCode string) string {
	base := filepath.Base(g.JSONPath)
	ext := filepath.Ext(base)
	sourceName := strings.TrimSuffix(base, ext)
	parts := strings.Split(sourceName, "_")
	name := strings.Join(parts[:len(parts)-1], "_")
	if name == "" {
		name = sourceName
	}

	pattern := g.DirPattern
	if pattern == "" || !strings.Contains(pattern, "{lang}") {
		pattern = "{name}_{lang}" + ext
	}

	result := pattern
	result = strings.ReplaceAll(result, "{lang}", langCode)
	result = strings.ReplaceAll(result, "{name}", name)
	result = strings.ReplaceAll(result, "{source}", sourceName)
	result = strings.ReplaceAll(result, "{ext}", ext)

	outPath := filepath.Join(g.OutputDir, result)
	os.MkdirAll(filepath.Dir(outPath), 0755)
	return outPath
}
