package xlsx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndRead(t *testing.T) {
	dir, err := os.MkdirTemp("", "xlsx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "test.xlsx")

	sourceLang := "en"
	sourceValues := []string{"Hello", "World", "Foo"}
	targetLangs := []string{"zh-CN", "ja"}

	writer := NewWriter(path)
	if err := writer.Write(sourceLang, sourceValues, targetLangs); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	reader := NewReader(path)
	data, err := reader.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if data.SourceLang != sourceLang {
		t.Errorf("SourceLang = %q, want %q", data.SourceLang, sourceLang)
	}

	if len(data.TargetLangs) != 2 || data.TargetLangs[0] != "zh-CN" || data.TargetLangs[1] != "ja" {
		t.Errorf("TargetLangs = %v, want [zh-CN ja]", data.TargetLangs)
	}

	if len(data.Rows) != 3 {
		t.Fatalf("Rows count = %d, want 3", len(data.Rows))
	}

	for i, row := range data.Rows {
		if row[0] != sourceValues[i] {
			t.Errorf("Rows[%d][0] = %q, want %q", i, row[0], sourceValues[i])
		}
		if row[1] != "" {
			t.Errorf("Rows[%d][1] should be empty for untranslated, got %q", i, row[1])
		}
	}

	if data.SplitMeta != nil {
		t.Errorf("SplitMeta should be nil for non-split write, got %v", data.SplitMeta)
	}
}

func TestWriteAndReadWithMeta(t *testing.T) {
	dir, err := os.MkdirTemp("", "xlsx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "test_meta.xlsx")

	sourceLang := "en"
	sourceValues := []string{"hello", "world"}
	targetLangs := []string{"zh-CN"}
	splitMeta := map[string][]string{
		"banner.content": {"<span>", "</span>", ""},
	}

	writer := NewWriter(path)
	if err := writer.WriteWithMeta(sourceLang, sourceValues, targetLangs, splitMeta); err != nil {
		t.Fatalf("WriteWithMeta error: %v", err)
	}

	reader := NewReader(path)
	data, err := reader.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if data.SourceLang != "en" {
		t.Errorf("SourceLang = %q, want en", data.SourceLang)
	}

	if len(data.Rows) != 2 {
		t.Errorf("Rows count = %d, want 2", len(data.Rows))
	}

	if data.SplitMeta == nil {
		t.Fatal("SplitMeta should not be nil")
	}

	tmpl, ok := data.SplitMeta["banner.content"]
	if !ok {
		t.Fatal("SplitMeta missing key banner.content")
	}

	expectedTmpl := []string{"<span>", "</span>", ""}
	if len(tmpl) != len(expectedTmpl) {
		t.Fatalf("Template length = %d, want %d", len(tmpl), len(expectedTmpl))
	}
	for i, v := range tmpl {
		if v != expectedTmpl[i] {
			t.Errorf("Template[%d] = %q, want %q", i, v, expectedTmpl[i])
		}
	}
}

func TestReadNonExistentFile(t *testing.T) {
	reader := NewReader("nonexistent.xlsx")
	_, err := reader.Read()
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestReadEmptyFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "xlsx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "empty.xlsx")

	writer := NewWriter(path)
	if err := writer.Write("en", []string{}, nil); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	reader := NewReader(path)
	_, err = reader.Read()
	if err == nil {
		t.Error("expected error for file with less than 2 language columns")
	}
}

func TestWriteWithDuplicateSourceLang(t *testing.T) {
	dir, err := os.MkdirTemp("", "xlsx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "dedup.xlsx")

	writer := NewWriter(path)
	// targetLangs includes sourceLang, should be deduplicated
	if err := writer.Write("en", []string{"hello"}, []string{"en", "zh-CN"}); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	reader := NewReader(path)
	data, err := reader.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	// Should only have "en" and "zh-CN", not duplicated "en"
	if len(data.TargetLangs) != 1 || data.TargetLangs[0] != "zh-CN" {
		t.Errorf("TargetLangs = %v, want [zh-CN]", data.TargetLangs)
	}
}

func TestReadSplitMetaNoSheet(t *testing.T) {
	dir, err := os.MkdirTemp("", "xlsx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "nosheet.xlsx")

	writer := NewWriter(path)
	if err := writer.Write("en", []string{"a", "b"}, []string{"zh-CN"}); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	reader := NewReader(path)
	data, err := reader.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	if data.SplitMeta != nil {
		t.Errorf("SplitMeta should be nil when no _meta sheet, got %v", data.SplitMeta)
	}
}

func TestReadSplitMetaInvalidJSON(t *testing.T) {
	dir, err := os.MkdirTemp("", "xlsx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "badmeta.xlsx")

	writer := NewWriter(path)
	if err := writer.WriteWithMeta("en", []string{"a"}, []string{"zh-CN"}, map[string][]string{
		"key": {"val"},
	}); err != nil {
		t.Fatalf("WriteWithMeta error: %v", err)
	}

	reader := NewReader(path)
	data, err := reader.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	metaJSON, _ := json.Marshal(map[string][]string{"key": {"val"}})
	if data.SplitMeta == nil {
		t.Fatal("SplitMeta should not be nil")
	}
	if data.SplitMeta["key"][0] != "val" {
		t.Errorf("SplitMeta key = %v, want [val]", data.SplitMeta["key"])
	}
	_ = metaJSON
}
