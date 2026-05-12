package extractor

import (
	"reflect"
	"testing"
)

func TestDetectLangFromFilename(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"about_us_en.json", "en"},
		{"home_zh-CN.json", "zh-CN"},
		{"messages_ja.json", "ja"},
		{"file_name_en.json", "en"},
		{"home.json", ""},
		{"single.json", ""},
		{"deep/path/file_ko.json", "ko"},
		{"a_b_c_de.json", "de"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			e := New(tt.path)
			got := e.DetectLangFromFilename()
			if got != tt.want {
				t.Errorf("DetectLangFromFilename(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestApplyStripTags(t *testing.T) {
	tests := []struct {
		name       string
		entries    []FlatEntry
		wantValues []string
	}{
		{
			name: "strip tags from entry with span",
			entries: []FlatEntry{
				{KeyPath: "banner.content", Value: "<span style='color: red;'>Find the important updates</span> and adjustments on our platform."},
			},
			wantValues: []string{
				"Find the important updates and adjustments on our platform.",
			},
		},
		{
			name: "strip br tag",
			entries: []FlatEntry{
				{KeyPath: "filter.lable", Value: "Filter <br>Date"},
			},
			wantValues: []string{
				"Filter Date",
			},
		},
		{
			name: "no tags kept as-is",
			entries: []FlatEntry{
				{KeyPath: "nav.0.btn_text", Value: "Home"},
				{KeyPath: "nav.1.btn_text", Value: "Client Notices"},
			},
			wantValues: []string{
				"Home",
				"Client Notices",
			},
		},
		{
			name: "multiple entries with and without tags",
			entries: []FlatEntry{
				{KeyPath: "a", Value: "plain"},
				{KeyPath: "b", Value: "<b>bold text</b>"},
				{KeyPath: "c", Value: "also plain"},
			},
			wantValues: []string{
				"plain",
				"bold text",
				"also plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ExtractResult{
				Entries: tt.entries,
			}
			result.applyStripTags()

			values := make([]string, len(result.Entries))
			for i, e := range result.Entries {
				values[i] = e.Value
			}

			if !reflect.DeepEqual(values, tt.wantValues) {
				t.Errorf("values mismatch\ngot:  %#v\nwant: %#v", values, tt.wantValues)
			}
		})
	}
}

func TestIsNonTranslatableKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"bgimg_src", true},
		{"icon_src", true},
		{"btn_link", true},
		{"company_link", true},
		{"category_no", true},
		{"data_url", true},
		{"path_href", true},
		{"img_icon", true},
		{"class_name", false},
		{"btn_text", false},
		{"title", false},
		{"content", false},
		{"email", true},
		{"contact_email", true},
		{"user_mail", true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isNonTranslatableKey(tt.key)
			if got != tt.want {
				t.Errorf("isNonTranslatableKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestIsSkippableValue(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"", true},
		{"12345", true},
		{"3.14", true},
		{"-1", true},
		{"+100", true},
		{"user@company.com", true},
		{"test@example.co.uk", true},
		{"/images/banner.png", true},
		{"/wp-content/themes/puprime_new/images/subtract_icon_grey.webp", true},
		{"https://example.com/file.pdf", true},
		{"http://test.org/file.pdf", true},
		{"./relative/path/file.jpg", true},
		{"../up/one/file.svg", true},
		{"Hello World", false},
		{"Translate this text", false},
		{"Client Notices", false},
		{"123 not a number", false},
		{"user@notanemail", false},
		{"/just/a/path/without/extension", false},
		{"https://example.com/page", false},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			got := isSkippableValue(tt.val)
			if got != tt.want {
				t.Errorf("isSkippableValue(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestExtractPlaceholders(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []string
	}{
		{"no placeholders", "Hello World", []string{}},
		{"curly brace", "Hello {name}", []string{"{name}"}},
		{"double curly brace", "Hello {{name}}", []string{"{{name}"}},
		{"printf format", "Value: %d items", []string{"%d"}},
		{"printf with position", "%1$s has %2$d items", []string{"%1$s", "%2$d"}},
		{"multiple types", "%s and %d and %f", []string{"%s", "%d", "%f"}},
		{"mixed placeholders", "Hello {name}, you have %d messages", []string{"{name}", "%d"}},
		{"placeholder with spaces", "{ user }", []string{"{ user }"}},
		{"printf verbose", "Value: %v", []string{"%v"}},
		{"printf type", "Type: %T", []string{"%T"}},
		{"printf percent", "100%%", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPlaceholders(tt.s)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractPlaceholders(%q) = %#v, want %#v", tt.s, got, tt.want)
			}
		})
	}
}

func TestFlattenBytes(t *testing.T) {
	tests := []struct {
		name string
		json string
		want []FlatEntry
	}{
		{
			name: "simple object",
			json: `{"title": "Hello", "content": "World"}`,
			want: []FlatEntry{
				{KeyPath: "title", Value: "Hello"},
				{KeyPath: "content", Value: "World"},
			},
		},
		{
			name: "nested object",
			json: `{"banner": {"title": "Hi", "content": "Text"}}`,
			want: []FlatEntry{
				{KeyPath: "banner.title", Value: "Hi"},
				{KeyPath: "banner.content", Value: "Text"},
			},
		},
		{
			name: "array of objects",
			json: `{"nav": [{"btn_text": "Home"}, {"btn_text": "About"}]}`,
			want: []FlatEntry{
				{KeyPath: "nav.0.btn_text", Value: "Home"},
				{KeyPath: "nav.1.btn_text", Value: "About"},
			},
		},
		{
			name: "filter skippable values",
			json: `{"title": "Hello", "img_src": "/img.png", "count": "42"}`,
			want: []FlatEntry{
				{KeyPath: "title", Value: "Hello"},
			},
		},
		{
			name: "filter non-translatable keys",
			json: `{"btn_text": "Click", "btn_link": "/home", "title": "Welcome"}`,
			want: []FlatEntry{
				{KeyPath: "btn_text", Value: "Click"},
				{KeyPath: "title", Value: "Welcome"},
			},
		},
		{
			name: "empty string value is skipped",
			json: `{"title": "", "content": "Real"}`,
			want: []FlatEntry{
				{KeyPath: "content", Value: "Real"},
			},
		},
		{
			name: "placeholder detection",
			json: `{"msg": "Hello {name}", "plain": "Hello"}`,
			want: []FlatEntry{
				{KeyPath: "msg", Value: "Hello {name}", HasPlaceholders: true},
				{KeyPath: "plain", Value: "Hello", HasPlaceholders: false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := flattenBytes([]byte(tt.json))
			if err != nil {
				t.Fatalf("flattenBytes error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("entry count\ngot:  %d %+v\nwant: %d %+v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i].KeyPath != tt.want[i].KeyPath || got[i].Value != tt.want[i].Value || got[i].HasPlaceholders != tt.want[i].HasPlaceholders {
					t.Errorf("entry[%d]\ngot:  KeyPath=%q Value=%q HasPH=%v\nwant: KeyPath=%q Value=%q HasPH=%v",
						i, got[i].KeyPath, got[i].Value, got[i].HasPlaceholders,
						tt.want[i].KeyPath, tt.want[i].Value, tt.want[i].HasPlaceholders)
				}
			}
		})
	}
}

func TestExtractEntries(t *testing.T) {
	json := `{"title": "Test", "values": ["a", "b"]}`
	got, err := ExtractEntries([]byte(json))
	if err != nil {
		t.Fatalf("ExtractEntries error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(got), got)
	}
}

func TestFlattenWithNonStringValues(t *testing.T) {
	tests := []struct {
		name string
		json string
		want int
	}{
		{"number values are skipped", `{"count": 42, "title": "hello"}`, 1},
		{"boolean values are skipped", `{"active": true, "title": "hello"}`, 1},
		{"null values are skipped", `{"data": null, "title": "hello"}`, 1},
		{"nested number skips correctly", `{"obj": {"count": 99, "name": "test"}}`, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := flattenBytes([]byte(tt.json))
			if err != nil {
				t.Fatalf("flattenBytes error: %v", err)
			}
			if len(got) != tt.want {
				t.Errorf("entry count = %d, want %d\nentries: %+v", len(got), tt.want, got)
			}
		})
	}
}

func TestFlattenInvalidJSON(t *testing.T) {
	_, err := flattenBytes([]byte(`{invalid json}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestApplySplitTags(t *testing.T) {
	tests := []struct {
		name          string
		entries       []FlatEntry
		wantValues    []string
		wantMetaCount int
	}{
		{
			name: "skip empty leading segment",
			entries: []FlatEntry{
				{KeyPath: "banner.title", Value: "Client Notices"},
				{KeyPath: "banner.content", Value: "<span style='color: red;'>Find the important updates</span> and adjustments on our platform."},
			},
			wantValues: []string{
				"Client Notices",
				"Find the important updates",
				" and adjustments on our platform.",
			},
			wantMetaCount: 1,
		},
		{
			name: "no tags kept as-is",
			entries: []FlatEntry{
				{KeyPath: "nav.0.btn_text", Value: "Home"},
				{KeyPath: "nav.1.btn_text", Value: "Client Notices"},
			},
			wantValues: []string{
				"Home",
				"Client Notices",
			},
			wantMetaCount: 0,
		},
		{
			name: "tag only no surrounding text",
			entries: []FlatEntry{
				{KeyPath: "desc", Value: "<span>highlighted</span>"},
			},
			wantValues: []string{
				"highlighted",
			},
			wantMetaCount: 1,
		},
		{
			name: "self-closing tag split",
			entries: []FlatEntry{
				{KeyPath: "filter.lable", Value: "Filter <br>Date"},
			},
			wantValues: []string{
				"Filter ",
				"Date",
			},
			wantMetaCount: 1,
		},
		{
			name: "multiple entries with and without tags",
			entries: []FlatEntry{
				{KeyPath: "a", Value: "plain"},
				{KeyPath: "b", Value: "<b>bold</b>"},
				{KeyPath: "c", Value: "also plain"},
			},
			wantValues: []string{
				"plain",
				"bold",
				"also plain",
			},
			wantMetaCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ExtractResult{
				Entries: tt.entries,
			}
			result.applySplitTags()

			values := make([]string, len(result.Entries))
			for i, e := range result.Entries {
				values[i] = e.Value
			}

			if !reflect.DeepEqual(values, tt.wantValues) {
				t.Errorf("values mismatch\ngot:  %#v\nwant: %#v", values, tt.wantValues)
			}

			if len(result.SplitMeta) != tt.wantMetaCount {
				t.Errorf("meta count mismatch\ngot:  %d\nwant: %d", len(result.SplitMeta), tt.wantMetaCount)
			}
		})
	}
}
