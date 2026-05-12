package tagsplit

import (
	"regexp"
	"strings"
)

var tagRegex = regexp.MustCompile(`<[^>]+>`)
var stripTagRegex = regexp.MustCompile(`<[^>]*>`)

type SplitInfo struct {
	Segments []string `json:"segments"`
	Template []string `json:"template"`
}

type SplitMetaEntry struct {
	Template []string `json:"t"`
	SegCount int      `json:"n"`
	Indices  []int    `json:"si,omitempty"`
}

func HasTags(s string) bool {
	return tagRegex.MatchString(s)
}

func StripHTML(s string) string {
	return strings.TrimSpace(stripTagRegex.ReplaceAllString(s, ""))
}

func Split(s string) SplitInfo {
	matches := tagRegex.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return SplitInfo{
			Segments: []string{s},
			Template: []string{"", ""},
		}
	}

	segments := make([]string, 0, len(matches)+1)
	template := make([]string, 0, len(matches)+1)

	pos := 0
	for _, m := range matches {
		if pos < m[0] {
			segments = append(segments, s[pos:m[0]])
		} else {
			segments = append(segments, "")
		}
		template = append(template, s[m[0]:m[1]])
		pos = m[1]
	}

	if pos < len(s) {
		segments = append(segments, s[pos:])
	} else {
		segments = append(segments, "")
	}
	template = append(template, "")

	return SplitInfo{
		Segments: segments,
		Template: template,
	}
}

func Reassemble(segments []string, template []string) string {
	var buf strings.Builder
	maxLen := len(template)
	if len(segments) > maxLen {
		maxLen = len(segments)
	}
	for i := 0; i < maxLen; i++ {
		if i < len(segments) {
			buf.WriteString(segments[i])
		}
		if i < len(template) {
			buf.WriteString(template[i])
		}
	}
	return buf.String()
}
