package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/sjson"

	"auto_i18n/internal/extractor"
	"auto_i18n/internal/xlsx"
)

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
	XLSXPath  string
	JSONPath  string
	OutputDir string
	Result    *RunResult
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

	if len(entries) != len(data.Rows) {
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

	for langIdx, lang := range data.TargetLangs {
		modified := sourceStr
		colIdx := langIdx + 1
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

			keyPath := entry.KeyPath
			modified, err = sjson.Set(modified, keyPath, translated)
			if err != nil {
				return fmt.Errorf("set value at %s for %s: %w", keyPath, lang, err)
			}
			count++
		}

		pct := float64(count) / float64(totalRows) * 100
		stats := LangStats{
			Lang:          lang,
			Total:         totalRows,
			Translated:    count,
			Missing:       totalRows - count,
			CompletionPct: pct,
			PHWarnings:    phWarnings,
		}
		g.Result.LangStats = append(g.Result.LangStats, stats)

		outPath := g.buildOutputPath(lang)
		if err := os.WriteFile(outPath, []byte(modified+"\n"), 0644); err != nil {
			return fmt.Errorf("write json for %s: %w", lang, err)
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

func (g *Generator) buildOutputPath(lang string) string {
	base := filepath.Base(g.JSONPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	parts := strings.Split(name, "_")
	newName := strings.Join(parts[:len(parts)-1], "_") + "_" + lang + ext
	return filepath.Join(g.OutputDir, newName)
}
