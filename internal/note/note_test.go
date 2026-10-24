package note

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var ts = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

func sample() Note {
	return Note{
		ID:          "go-error-wrapping",
		Title:       "Go error wrapping",
		Description: "Always wrap errors with %w and context",
		Tags:        []string{"errors", "go"},
		Created:     ts,
		Updated:     ts.Add(time.Hour),
		Body:        "Body in free Markdown...",
	}
}

const sampleFile = `---
id: go-error-wrapping
title: Go error wrapping
description: Always wrap errors with %w and context
tags: [errors, go]
created: 2026-10-24T09:00:00Z
updated: 2026-10-24T10:00:00Z
---

Body in free Markdown...
`

func TestMarshalFormat(t *testing.T) {
	got, err := sample().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sampleFile {
		t.Errorf("Marshal mismatch\ngot:\n%s\nwant:\n%s", got, sampleFile)
	}
}

func TestMarshalNormalizesTimesToUTC(t *testing.T) {
	n := sample()
	n.Created = time.Date(2026, 10, 24, 12, 0, 0, 500, time.FixedZone("EEST", 3*3600))
	got, err := n.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\ncreated: 2026-10-24T09:00:00Z\n") {
		t.Errorf("created not RFC3339 UTC:\n%s", got)
	}
}

func TestMarshalEmptyTagsAndBody(t *testing.T) {
	n := sample()
	n.Tags = nil
	n.Body = ""
	got, err := n.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if !strings.Contains(s, "\ntags: []\n") {
		t.Errorf("want empty flow tags, got:\n%s", s)
	}
	if !strings.HasSuffix(s, "\n---\n") {
		t.Errorf("want file to end at closing delimiter, got:\n%q", s)
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		mod  func(*Note)
	}{
		{"sample", func(*Note) {}},
		{"empty body", func(n *Note) { n.Body = "" }},
		{"no tags", func(n *Note) { n.Tags = []string{} }},
		{"body with delimiter lines", func(n *Note) { n.Body = "intro\n---\nmiddle\n\n---\n\nend" }},
		{"body starting with delimiter", func(n *Note) { n.Body = "---\nnot frontmatter" }},
		{"multiline body with code", func(n *Note) { n.Body = "# Title\n\n    indented code\n\n- item" }},
		{"yaml-special strings", func(n *Note) {
			n.Title = "key: value # not a comment"
			n.Description = "- starts like a list, has \"quotes\" and 'apostrophes'"
			n.Tags = []string{"c++", "yes"}
		}},
		{"empty description", func(n *Note) { n.Description = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := sample()
			tt.mod(&want)
			data, err := want.Marshal()
			if err != nil {
				t.Fatal(err)
			}
			got, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v\n%s", err, data)
			}
			if want.Tags == nil {
				want.Tags = []string{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round trip mismatch\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantBody string
		wantErr  bool
	}{
		{"sample", sampleFile, "Body in free Markdown...", false},
		{"crlf", strings.ReplaceAll(sampleFile, "\n", "\r\n"), "Body in free Markdown...", false},
		{"crlf multiline body", strings.ReplaceAll(sampleFile, "Body in free Markdown...", "a\nb"), "a\nb", false},
		{"no blank separator", strings.Replace(sampleFile, "---\n\n", "---\n", 1), "Body in free Markdown...", false},
		{"no body", strings.TrimSuffix(sampleFile, "\nBody in free Markdown...\n"), "", false},
		{"no trailing newline after delimiter", strings.TrimSuffix(sampleFile, "\n\nBody in free Markdown...\n"), "", false},
		{"body with delimiter", sampleFile + "---\nmore\n", "Body in free Markdown...\n---\nmore", false},
		{"missing frontmatter", "just markdown\n", "", true},
		{"empty input", "", "", true},
		{"unterminated frontmatter", "---\nid: x\ntitle: y\n", "", true},
		{"delimiter must be whole line", "---\nid: x\n----\nbody\n", "", true},
		{"invalid yaml", "---\nid: [unclosed\n---\n", "", true},
		{"missing id", "---\ntitle: y\n---\n", "", true},
		{"bad timestamp", "---\nid: x\ncreated: yesterday\n---\n", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := Parse([]byte(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if n.ID != "go-error-wrapping" || n.Title != "Go error wrapping" {
				t.Errorf("unexpected header: %#v", n)
			}
			if !n.Created.Equal(ts) || n.Created.Location() != time.UTC {
				t.Errorf("created = %v", n.Created)
			}
			if n.Body != tt.wantBody {
				t.Errorf("body = %q, want %q", n.Body, tt.wantBody)
			}
		})
	}
}

func TestParseBlockStyleTags(t *testing.T) {
	in := "---\nid: x\ntags:\n  - a\n  - b\n---\n"
	n, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(n.Tags, []string{"a", "b"}) {
		t.Errorf("tags = %q", n.Tags)
	}
}
