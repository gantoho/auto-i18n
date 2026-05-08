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

type Generator struct {
	XLSXPath  string
	JSONPath  string
	OutputDir string
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
			"entry count mismatch: source JSON has %d translatable entries, xlsx has %d rows",
			len(entries), len(data.Rows),
		)
	}

	sourceStr := string(sourceJSON)

	for langIdx, lang := range data.TargetLangs {
		modified := sourceStr
		colIdx := langIdx + 1
		count := 0

		for rowIdx, row := range data.Rows {
			if colIdx >= len(row) {
				continue
			}
			translated := strings.TrimSpace(row[colIdx])
			if translated == "" {
				continue
			}
			keyPath := entries[rowIdx].KeyPath
			modified, err = sjson.Set(modified, keyPath, translated)
			if err != nil {
				return fmt.Errorf("set value at %s for %s: %w", keyPath, lang, err)
			}
			count++
		}

		outPath := g.buildOutputPath(lang)
		if err := os.WriteFile(outPath, []byte(modified+"\n"), 0644); err != nil {
			return fmt.Errorf("write json for %s: %w", lang, err)
		}

		fmt.Printf("  ✓ %s (%d translations)\n", filepath.Base(outPath), count)
	}

	return nil
}

func (g *Generator) buildOutputPath(lang string) string {
	base := filepath.Base(g.JSONPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	parts := strings.Split(name, "_")
	newName := strings.Join(parts[:len(parts)-1], "_") + "_" + lang + ext
	return filepath.Join(g.OutputDir, newName)
}
