package note

import (
	"reflect"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "Go error wrapping", "go-error-wrapping"},
		{"punctuation", "Hello, World! (v2)", "hello-world-v2"},
		{"collapse dashes", "a  --  b", "a-b"},
		{"trim dashes", "  --leading and trailing--  ", "leading-and-trailing"},
		{"underscores and dots", "snake_case.file", "snake-case-file"},
		{"transliterate", "Äiti söi Crème brûlée straße", "aiti-soi-creme-brulee-strasse"},
		{"drop non-latin", "日本語 notes", "notes"},
		{"digits", "Release 1.26", "release-1-26"},
		{"empty", "", "note"},
		{"only symbols", "!!! ???", "note"},
		{"only non-ascii", "日本語", "note"},
		{"cap length", strings.Repeat("a", 80), strings.Repeat("a", 60)},
		{"cap without trailing dash", strings.Repeat("a", 59) + " bcd", strings.Repeat("a", 59)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.in); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, []string{}},
		{"trim lower sort", []string{" Go", "errors ", "API"}, []string{"api", "errors", "go"}},
		{"dedupe case-insensitive", []string{"go", "Go", "GO "}, []string{"go"}},
		{"drop empty", []string{"", "  ", "x"}, []string{"x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTags(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeTags(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEditTags(t *testing.T) {
	tests := []struct {
		name              string
		tags, add, remove []string
		want              []string
	}{
		{"nothing", []string{"b", "a"}, nil, nil, []string{"a", "b"}},
		{"add normalized", []string{"a"}, []string{" B", "a"}, nil, []string{"a", "b"}},
		{"remove case-insensitive", []string{"a", "b"}, nil, []string{"B "}, []string{"a"}},
		{"remove wins over add", []string{"a"}, []string{"b"}, []string{"b"}, []string{"a"}},
		{"nil", nil, nil, nil, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EditTags(tt.tags, tt.add, tt.remove); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EditTags(%q, %q, %q) = %q, want %q", tt.tags, tt.add, tt.remove, got, tt.want)
			}
		})
	}
}
