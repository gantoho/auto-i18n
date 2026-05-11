package lang

import "testing"

func TestCodeToName(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"cn", "Simplified Chinese"},
		{"jp", "Japanese"},
		{"kr", "Korean"},
		{"en", "English"},   // now in supported list
		{"unknown", "unknown"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := CodeToName(tt.code)
			if got != tt.want {
				t.Errorf("CodeToName(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestNameToCode(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Simplified Chinese", "cn"},
		{"Japanese", "jp"},
		{"Korean", "kr"},
		{"Unknown Name", "Unknown Name"}, // not in list, returns itself
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NameToCode(tt.name)
			if got != tt.want {
				t.Errorf("NameToCode(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRoundtrip(t *testing.T) {
	for _, entry := range Supported {
		name := CodeToName(entry.Code)
		if name != entry.Name {
			t.Errorf("CodeToName(%q) = %q, want %q", entry.Code, name, entry.Name)
		}
		code := NameToCode(entry.Name)
		if code != entry.Code {
			t.Errorf("NameToCode(%q) = %q, want %q", entry.Name, code, entry.Code)
		}
	}
}
