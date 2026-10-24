package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type jsonNote struct {
	ID          string
	Title       string
	Description string
	Tags        []string
	Created     time.Time
	Updated     time.Time
	Body        string
	Path        string
	Score       *int
}

func (e *testEnv) notePath(id string) string { return filepath.Join(e.notesDir(), id+".md") }

func (e *testEnv) showJSON(id string) jsonNote {
	e.t.Helper()
	var n jsonNote
	decodeJSON(e.t, e.mustRun("show", id, "--json"), &n)
	return n
}

func TestAddAndShow(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("add", "--title", "Go error wrapping", "--description", "Wrap with %w",
		"--tags", "Go,errors", "--tags", "testing", "--body", "Use %w.\n\nAlways.")
	if want := "created go-error-wrapping (" + e.notePath("go-error-wrapping") + ")\n"; out != want {
		t.Errorf("add stdout = %q, want %q", out, want)
	}

	out = e.mustRun("show", "go-error-wrapping")
	want := `---
id: go-error-wrapping
title: Go error wrapping
description: Wrap with %w
tags: [errors, go, testing]
created: 2026-10-24T09:00:00Z
updated: 2026-10-24T09:00:00Z
path: ` + e.notePath("go-error-wrapping") + `
---

Use %w.

Always.
`
	if out != want {
		t.Errorf("show stdout =\n%s\nwant\n%s", out, want)
	}
}

func TestShowMinimalNote(t *testing.T) {
	e := newEnv(t)
	e.mustRun("add", "--title", "Bare")
	want := "---\nid: bare\ntitle: Bare\ntags: []\ncreated: 2026-10-24T09:00:00Z\nupdated: 2026-10-24T09:00:00Z\npath: " +
		e.notePath("bare") + "\n---\n"
	if out := e.mustRun("show", "bare"); out != want {
		t.Errorf("show stdout =\n%s\nwant\n%s", out, want)
	}
}

func TestAddJSON(t *testing.T) {
	e := newEnv(t)
	var got jsonNote
	decodeJSON(t, e.mustRun("--json", "add", "--title", "Redis cache", "--tags", "redis", "--body", "TTL 60s"), &got)
	want := jsonNote{
		ID:      "redis-cache",
		Title:   "Redis cache",
		Tags:    []string{"redis"},
		Created: t0,
		Updated: t0,
		Body:    "TTL 60s",
		Path:    e.notePath("redis-cache"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("add --json =\n%#v\nwant\n%#v", got, want)
	}
	if show := e.showJSON("redis-cache"); !reflect.DeepEqual(show, want) {
		t.Errorf("show --json =\n%#v\nwant\n%#v", show, want)
	}
}

func TestShowJSONShape(t *testing.T) {
	e := newEnv(t)
	e.mustRun("add", "--title", "Bare")
	out := e.mustRun("show", "--json", "bare")
	for _, want := range []string{`"id": "bare"`, `"description": ""`, `"tags": []`, `"created": "2026-10-24T09:00:00Z"`, `"path": `} {
		if !strings.Contains(out, want) {
			t.Errorf("show --json missing %s:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{`"body"`, `"score"`} {
		if strings.Contains(out, unwanted) {
			t.Errorf("show --json has %s for an empty value:\n%s", unwanted, out)
		}
	}
}

func TestAddBodyFromStdin(t *testing.T) {
	e := newEnv(t)
	e.stdin = "line one\nline two\n"
	var added jsonNote
	decodeJSON(t, e.mustRun("add", "--title", "Piped", "--body", "-", "--json"), &added)
	if added.Body != "line one\nline two" {
		t.Errorf("add --json body = %q", added.Body)
	}
	if got := e.showJSON("piped").Body; got != added.Body {
		t.Errorf("stored body = %q, want %q", got, added.Body)
	}
}

func TestAddIgnoresStdinWithoutBodyFlag(t *testing.T) {
	e := newEnv(t)
	e.stdin = "should not be read"
	e.mustRun("add", "--title", "No body")
	if got := e.showJSON("no-body").Body; got != "" {
		t.Errorf("body = %q, want empty", got)
	}
}

func TestAddBodyFile(t *testing.T) {
	e := newEnv(t)
	if err := os.WriteFile(filepath.Join(e.cwd, "body.md"), []byte("# From file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.mustRun("add", "--title", "Filed", "--body-file", "body.md")
	if got := e.showJSON("filed").Body; got != "# From file" {
		t.Errorf("body = %q", got)
	}
}

func TestAddErrors(t *testing.T) {
	e := newEnv(t)
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"add"}, 2},
		{[]string{"add", "--title", "  "}, 2},
		{[]string{"add", "--title", "T", "stray"}, 2},
		{[]string{"add", "--title", "T", "--body", "x", "--body-file", "f"}, 2},
		{[]string{"add", "--title", "T", "--tags"}, 2},
		{[]string{"add", "--title", "T", "--body-file", "missing.md"}, 1},
	} {
		_, errOut, code := e.run(tc.args...)
		assertCode(t, tc.args, code, tc.code, errOut)
		if !strings.HasPrefix(errOut, "error: ") {
			t.Errorf("%q: stderr = %q, want error message", tc.args, errOut)
		}
	}
	if _, err := os.Stat(e.notesDir()); !os.IsNotExist(err) {
		t.Errorf("failed adds created notes dir: %v", err)
	}
}

func TestShowErrors(t *testing.T) {
	e := newEnv(t)
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"show"}, 2},
		{[]string{"show", "a", "b"}, 2},
		{[]string{"show", "missing"}, 3},
		{[]string{"show", "missing", "--json"}, 3},
	} {
		out, errOut, code := e.run(tc.args...)
		assertCode(t, tc.args, code, tc.code, errOut)
		if out != "" {
			t.Errorf("%q: stdout = %q, want empty", tc.args, out)
		}
	}
}
