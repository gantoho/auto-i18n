package extractor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"auto_i18n/internal/tagsplit"
)

var (
	nonTranslatableSuffixes = []string{
		"_src", "_link", "_no", "_url", "_path", "_href",
		"_img", "_icon", "_class", "_id", "_key",
		"_mail", "email",
	}

	urlRegex         = regexp.MustCompile(`^(https?://|/|\./|\.\./)`)
	extRegex         = regexp.MustCompile(`\.\w+$`)
	numberRegex      = regexp.MustCompile(`^[+-]?\d*\.?\d+$`)
	emailRegex       = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	placeholderRegex = regexp.MustCompile(`\{[^}]+\}|%[dsfvebboxtT]|%[0-9]+\$[dsf]|\{\{[^}]+\}\}`)
)

type Extractor struct {
	JSONPath  string
	SplitTags bool
}

type FlatEntry struct {
	KeyPath         string
	Value           string
	HasPlaceholders bool
	SubIndex        int
}

type ExtractResult struct {
	SourceLang string
	Entries    []FlatEntry
	SplitMeta  map[string]tagsplit.SplitMetaEntry
}

func New(jsonPath string) *Extractor {
	return &Extractor{JSONPath: jsonPath}
}

func (e *Extractor) DetectLangFromFilename() string {
	base := filepath.Base(e.JSONPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	parts := strings.Split(name, "_")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

func (e *Extractor) Run() (*ExtractResult, error) {
	data, err := os.ReadFile(e.JSONPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	entries, err := flattenBytes(data)
	if err != nil {
		return nil, fmt.Errorf("flatten json: %w", err)
	}

	result := &ExtractResult{
		SourceLang: e.DetectLangFromFilename(),
		Entries:    entries,
	}

	if e.SplitTags {
		result.applySplitTags()
	}

	return result, nil
}

func (r *ExtractResult) applySplitTags() {
	newEntries := make([]FlatEntry, 0, len(r.Entries))
	meta := make(map[string]tagsplit.SplitMetaEntry)

	for _, entry := range r.Entries {
		if tagsplit.HasTags(entry.Value) {
			info := tagsplit.Split(entry.Value)
			segCount := 0
			indices := make([]int, 0)
			for i, seg := range info.Segments {
				if seg == "" {
					continue
				}
				segCount++
				indices = append(indices, i)
				newEntries = append(newEntries, FlatEntry{
					KeyPath:         entry.KeyPath,
					Value:           seg,
					HasPlaceholders: entry.HasPlaceholders,
					SubIndex:        i,
				})
			}
			meta[entry.KeyPath] = tagsplit.SplitMetaEntry{
				Template: info.Template,
				SegCount: segCount,
				Indices:  indices,
			}
		} else {
			newEntries = append(newEntries, entry)
		}
	}

	r.Entries = newEntries
	if len(meta) > 0 {
		r.SplitMeta = meta
	}
}

func ExtractEntries(data []byte) ([]FlatEntry, error) {
	return flattenBytes(data)
}

func flattenBytes(data []byte) ([]FlatEntry, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	entries := make([]FlatEntry, 0)
	pathStack := make([]string, 0)

	if err := readValue(dec, &pathStack, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func readValue(dec *json.Decoder, pathStack *[]string, entries *[]FlatEntry) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}

	switch tt := t.(type) {
	case json.Delim:
		switch tt {
		case '{':
			return readObject(dec, pathStack, entries)
		case '[':
			return readArray(dec, pathStack, entries)
		}

	case string:
		if !isSkippableValue(tt) {
			hasPH := placeholderRegex.MatchString(tt)
			*entries = append(*entries, FlatEntry{
				KeyPath:         strings.Join(*pathStack, "."),
				Value:           tt,
				HasPlaceholders: hasPH,
			})
		}
	}

	return nil
}

func readObject(dec *json.Decoder, pathStack *[]string, entries *[]FlatEntry) error {
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return err
		}
		key := keyToken.(string)

		if isNonTranslatableKey(key) {
			var skip interface{}
			if err := dec.Decode(&skip); err != nil {
				return err
			}
			continue
		}

		*pathStack = append(*pathStack, key)
		if err := readValue(dec, pathStack, entries); err != nil {
			return err
		}
		*pathStack = (*pathStack)[:len(*pathStack)-1]
	}

	_, err := dec.Token()
	return err
}

func readArray(dec *json.Decoder, pathStack *[]string, entries *[]FlatEntry) error {
	idx := 0
	for dec.More() {
		*pathStack = append(*pathStack, strconv.Itoa(idx))
		if err := readValue(dec, pathStack, entries); err != nil {
			return err
		}
		*pathStack = (*pathStack)[:len(*pathStack)-1]
		idx++
	}

	_, err := dec.Token()
	return err
}

func ExtractPlaceholders(s string) []string {
	matches := placeholderRegex.FindAllString(s, -1)
	if matches == nil {
		return []string{}
	}
	return matches
}

func isNonTranslatableKey(key string) bool {
	for _, suffix := range nonTranslatableSuffixes {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

func isSkippableValue(val string) bool {
	if len(val) == 0 {
		return true
	}

	if numberRegex.MatchString(val) {
		return true
	}

	if emailRegex.MatchString(val) {
		return true
	}

	if urlRegex.MatchString(val) {
		if extRegex.MatchString(val) {
			return true
		}
	}

	return false
}
