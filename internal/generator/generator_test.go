package generator

import (
	"path/filepath"
	"testing"

	"auto_i18n/internal/extractor"
	"auto_i18n/internal/tagsplit"
)

func TestPlaceholderSetsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"same single", []string{"{name}"}, []string{"{name}"}, true},
		{"same multiple", []string{"{name}", "%d"}, []string{"{name}", "%d"}, true},
		{"different order", []string{"%d", "{name}"}, []string{"{name}", "%d"}, true},
		{"different counts", []string{"%d"}, []string{"%d", "%d"}, false},
		{"different values", []string{"{name}"}, []string{"{user}"}, false},
		{"one nil one empty", nil, []string{}, true},
		{"duplicates match", []string{"%d", "%d"}, []string{"%d", "%d"}, true},
		{"duplicates mismatch", []string{"%d", "%d"}, []string{"%d", "%s"}, false},
		{"printf with position", []string{"%1$s", "%2$d"}, []string{"%1$s", "%2$d"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := placeholderSetsEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("placeholderSetsEqual(%#v, %#v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestBuildOutputPath(t *testing.T) {
	tests := []struct {
		name      string
		jsonPath  string
		outputDir string
		lang      string
		want      string
	}{
		{
			name:      "simple replacement",
			jsonPath:  filepath.FromSlash("/path/to/about_us_en.json"),
			outputDir: filepath.FromSlash("/path/to"),
			lang:      "zh-CN",
			want:      filepath.FromSlash("/path/to/about_us_zh-CN.json"),
		},
		{
			name:      "different output dir",
			jsonPath:  filepath.FromSlash("/path/to/about_us_en.json"),
			outputDir: filepath.FromSlash("/other/dir"),
			lang:      "ja",
			want:      filepath.FromSlash("/other/dir/about_us_ja.json"),
		},
		{
			name:      "multiple underscores in name",
			jsonPath:  filepath.FromSlash("/path/my_app_messages_en.json"),
			outputDir: filepath.FromSlash("/path"),
			lang:      "ko",
			want:      filepath.FromSlash("/path/my_app_messages_ko.json"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				JSONPath:  tt.jsonPath,
				OutputDir: tt.outputDir,
			}
			got := g.buildOutputPath(tt.lang)
			if got != tt.want {
				t.Errorf("buildOutputPath(%q) = %q, want %q", tt.lang, got, tt.want)
			}
		})
	}
}

func TestExpandedEntryCount(t *testing.T) {
	tests := []struct {
		name    string
		entries []extractor.FlatEntry
		meta    map[string]tagsplit.SplitMetaEntry
		want    int
	}{
		{
			name: "no meta",
			entries: []extractor.FlatEntry{
				{KeyPath: "a"},
				{KeyPath: "b"},
			},
			meta: nil,
			want: 2,
		},
		{
			name: "with split entries",
			entries: []extractor.FlatEntry{
				{KeyPath: "a"},
				{KeyPath: "b"},
				{KeyPath: "c"},
			},
			meta: map[string]tagsplit.SplitMetaEntry{
				"b": {Template: []string{"<span>", "</span>", ""}, SegCount: 2},
			},
			want: 4,
		},
		{
			name: "multiple split entries",
			entries: []extractor.FlatEntry{
				{KeyPath: "x"},
				{KeyPath: "y"},
			},
			meta: map[string]tagsplit.SplitMetaEntry{
				"x": {Template: []string{"<br>", ""}, SegCount: 2},
				"y": {Template: []string{"<b>", "</b>", ""}, SegCount: 2},
			},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{}
			got := g.expandedEntryCount(tt.entries, tt.meta)
			if got != tt.want {
				t.Errorf("expandedEntryCount = %d, want %d", got, tt.want)
			}
		})
	}
}
