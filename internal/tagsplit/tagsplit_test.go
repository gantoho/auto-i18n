package tagsplit

import (
	"reflect"
	"testing"
)

func TestHasTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"plain text", "Hello World", false},
		{"with span tag", "<span>hello</span>", true},
		{"with style attribute", "<span style='color: red;'>text</span>", true},
		{"self-closing tag", "line1<br>line2", true},
		{"tag with attributes", "<a href='/link'>click</a>", true},
		{"angled bracket not a tag", "5 > 3 and 2 < 4", false},
		{"empty string", "", false},
		{"only tag", "<br>", true},
		{"multiple tags", "<div><span>text</span></div>", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasTags(tt.input)
			if got != tt.want {
				t.Errorf("HasTags(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "Hello World", "Hello World"},
		{"span tag", "<span>hello</span>", "hello"},
		{"span with attributes", "<span style='color: red;'>text</span>", "text"},
		{"self-closing tag", "Line1<br>Line2", "Line1Line2"},
		{"tag at start", "<br>Hello", "Hello"},
		{"tag at end", "Hello<br>", "Hello"},
		{"nested tags", "<div><span>text</span></div>", "text"},
		{"no tags", "plain text only", "plain text only"},
		{"empty string", "", ""},
		{"only tags", "<br><br>", ""},
		{"tag with trailing spaces", "  <span>hello</span>  ", "hello"},
		{"multiple text nodes", "<b>bold</b> and <i>italic</i>", "bold and italic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripHTML(tt.input)
			if got != tt.want {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		segments []string
		template []string
	}{
		{
			name:     "plain text no tags",
			input:    "Hello World",
			segments: []string{"Hello World"},
			template: []string{"", ""},
		},
		{
			name:     "tag at start",
			input:    "<span style='color: red;'>Find the important updates</span> and adjustments on our platform.",
			segments: []string{"", "Find the important updates", " and adjustments on our platform."},
			template: []string{"<span style='color: red;'>", "</span>", ""},
		},
		{
			name:     "tag in middle",
			input:    "Hello <span>world</span>!",
			segments: []string{"Hello ", "world", "!"},
			template: []string{"<span>", "</span>", ""},
		},
		{
			name:     "tag at end",
			input:    "Click <a href='/link'>here</a>",
			segments: []string{"Click ", "here", ""},
			template: []string{"<a href='/link'>", "</a>", ""},
		},
		{
			name:     "tag only no surrounding text",
			input:    "<span>world</span>",
			segments: []string{"", "world", ""},
			template: []string{"<span>", "</span>", ""},
		},
		{
			name:     "self closing tag",
			input:    "Line1<br>Line2",
			segments: []string{"Line1", "Line2"},
			template: []string{"<br>", ""},
		},
		{
			name:     "multiple sequential tags",
			input:    "<b>bold</b><i>italic</i>",
			segments: []string{"", "bold", "", "italic", ""},
			template: []string{"<b>", "</b>", "<i>", "</i>", ""},
		},
		{
			name:     "empty string",
			input:    "",
			segments: []string{""},
			template: []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := Split(tt.input)
			if !reflect.DeepEqual(info.Segments, tt.segments) {
				t.Errorf("Segments mismatch\ngot:  %#v\nwant: %#v", info.Segments, tt.segments)
			}
			if !reflect.DeepEqual(info.Template, tt.template) {
				t.Errorf("Template mismatch\ngot:  %#v\nwant: %#v", info.Template, tt.template)
			}
		})
	}
}

func TestReassemble(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		template []string
		want     string
	}{
		{
			name:     "tag at start with trailing text",
			segments: []string{"", "Find the important updates", " and adjustments on our platform."},
			template: []string{"<span style='color: red;'>", "</span>", ""},
			want:     "<span style='color: red;'>Find the important updates</span> and adjustments on our platform.",
		},
		{
			name:     "tag only no surrounding text",
			segments: []string{"", "highlighted", ""},
			template: []string{"<span>", "</span>", ""},
			want:     "<span>highlighted</span>",
		},
		{
			name:     "no tags plain text",
			segments: []string{"Hello World"},
			template: []string{"", ""},
			want:     "Hello World",
		},
		{
			name:     "self closing tag",
			segments: []string{"Line1", "Line2"},
			template: []string{"<br>", ""},
			want:     "Line1<br>Line2",
		},
		{
			name:     "tag in middle",
			segments: []string{"Hello ", "world", "!"},
			template: []string{"<span>", "</span>", ""},
			want:     "Hello <span>world</span>!",
		},
		{
			name:     "multiple self closing tags",
			segments: []string{"A", "B", "C"},
			template: []string{"<br>", "<br>", ""},
			want:     "A<br>B<br>C",
		},
		{
			name:     "multiple sequential tags",
			segments: []string{"", "bold", "", "italic", ""},
			template: []string{"<b>", "</b>", "<i>", "</i>", ""},
			want:     "<b>bold</b><i>italic</i>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reassemble(tt.segments, tt.template)
			if got != tt.want {
				t.Errorf("Reassemble mismatch\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
