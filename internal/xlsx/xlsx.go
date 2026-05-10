package xlsx

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

type XLSXReader struct {
	Path string
}

type XLSXWriter struct {
	Path string
}

type SheetData struct {
	SourceLang  string
	TargetLangs []string
	Rows        [][]string
	SplitMeta   map[string][]string
}

func NewWriter(path string) *XLSXWriter {
	return &XLSXWriter{Path: path}
}

func NewReader(path string) *XLSXReader {
	return &XLSXReader{Path: path}
}

func (w *XLSXWriter) Write(sourceLang string, sourceValues []string, targetLangs []string) error {
	return w.WriteWithMeta(sourceLang, sourceValues, targetLangs, nil)
}

func (w *XLSXWriter) WriteWithMeta(sourceLang string, sourceValues []string, targetLangs []string, splitMeta map[string][]string) error {
	os.Remove(w.Path)

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"

	headers := []string{sourceLang}
	for _, lang := range targetLangs {
		if lang != sourceLang {
			headers = append(headers, lang)
		}
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#ffffff"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4f46e5"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	evenStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#f8f9fc"}},
	})

	lastCol := mustColumnName(len(headers))

	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", col)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", lastCol), headerStyle)

	for rowIdx, val := range sourceValues {
		rowNum := rowIdx + 2
		for ci := 1; ci <= len(headers); ci++ {
			col, _ := excelize.ColumnNumberToName(ci)
			cellVal := ""
			if ci == 1 {
				cellVal = val
			}
			f.SetCellStr(sheet, fmt.Sprintf("%s%d", col, rowNum), cellVal)
		}
		if rowIdx%2 == 1 {
			cellRef := fmt.Sprintf("A%d", rowNum)
			endRef := fmt.Sprintf("%s%d", lastCol, rowNum)
			f.SetCellStyle(sheet, cellRef, endRef, evenStyle)
		}
	}

	firstCol, _ := excelize.ColumnNumberToName(1)
	f.SetColWidth(sheet, firstCol, lastCol, 30)

	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		ActivePane:  "bottomLeft",
		TopLeftCell: "A2",
	})

	if len(splitMeta) > 0 {
		metaData, _ := json.Marshal(splitMeta)
		f.NewSheet("_meta")
		f.SetCellValue("_meta", "A1", string(metaData))
	}

	return f.SaveAs(w.Path)
}

func (r *XLSXReader) Read() (*SheetData, error) {
	f, err := excelize.OpenFile(r.Path)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}

	if len(rows) < 1 {
		return nil, fmt.Errorf("empty xlsx file")
	}

	headers := rows[0]
	if len(headers) < 2 {
		return nil, fmt.Errorf("xlsx must have at least 2 language columns, got %d", len(headers))
	}

	data := &SheetData{
		SourceLang:  headers[0],
		TargetLangs: headers[1:],
		Rows:        make([][]string, 0, len(rows)-1),
		SplitMeta:   readSplitMeta(f),
	}

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}
		vals := make([]string, len(headers))
		for j := 0; j < len(headers) && j < len(row); j++ {
			vals[j] = row[j]
		}
		data.Rows = append(data.Rows, vals)
	}

	return data, nil
}

func readSplitMeta(f *excelize.File) map[string][]string {
	sheets := f.GetSheetList()
	hasMeta := false
	for _, s := range sheets {
		if s == "_meta" {
			hasMeta = true
			break
		}
	}
	if !hasMeta {
		return nil
	}

	val, err := f.GetCellValue("_meta", "A1")
	if err != nil || val == "" {
		return nil
	}

	var meta map[string][]string
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		return nil
	}
	return meta
}

func mustColumnName(n int) string {
	name, err := excelize.ColumnNumberToName(n)
	if err != nil {
		panic(err)
	}
	return name
}
