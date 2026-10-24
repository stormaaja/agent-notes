package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func seedOne(e *testEnv) jsonNote {
	e.t.Helper()
	e.mustRun("add", "--title", "Redis cache", "--description", "TTL rules", "--tags", "go,redis", "--body", "Keep TTL short.")
	return e.showJSON("redis-cache")
}

func TestUpdatePartial(t *testing.T) {
	t1 := t0.Add(time.Minute)
	for _, tc := range []struct {
		name   string
		args   []string
		stdin  string
		change func(*jsonNote)
	}{
		{"add and remove tags", []string{"--add-tag", "cache,Perf", "--remove-tag", "redis", "--add-tag", "x"}, "",
			func(n *jsonNote) { n.Tags = []string{"cache", "go", "perf", "x"} }},
		{"replace tags", []string{"--tags", "a,b"}, "", func(n *jsonNote) { n.Tags = []string{"a", "b"} }},
		{"clear tags", []string{"--tags", ""}, "", func(n *jsonNote) { n.Tags = []string{} }},
		{"title keeps id", []string{"--title", "Valkey cache"}, "", func(n *jsonNote) { n.Title = "Valkey cache" }},
		{"clear description", []string{"--description", ""}, "", func(n *jsonNote) { n.Description = "" }},
		{"body", []string{"--body", "New body"}, "", func(n *jsonNote) { n.Body = "New body" }},
		{"body from stdin", []string{"--body", "-"}, "piped\n", func(n *jsonNote) { n.Body = "piped" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			want := seedOne(e)
			e.stdin = tc.stdin
			out := e.mustRun(append([]string{"update", "redis-cache"}, tc.args...)...)
			if w := "updated redis-cache (" + e.notePath("redis-cache") + ")\n"; out != w {
				t.Errorf("stdout = %q, want %q", out, w)
			}
			tc.change(&want)
			want.Updated = t1
			if got := e.showJSON("redis-cache"); !reflect.DeepEqual(got, want) {
				t.Errorf("after update =\n%#v\nwant\n%#v", got, want)
			}
		})
	}
}

func TestUpdateBodyFileAndJSON(t *testing.T) {
	e := newEnv(t)
	want := seedOne(e)
	if err := os.WriteFile(filepath.Join(e.cwd, "b.md"), []byte("from file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got jsonNote
	decodeJSON(t, e.mustRun("update", "--body-file", "b.md", "redis-cache", "--json"), &got)
	want.Body, want.Updated = "from file", t0.Add(time.Minute)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("update --json =\n%#v\nwant\n%#v", got, want)
	}
}

func TestUpdateErrors(t *testing.T) {
	e := newEnv(t)
	seedOne(e)
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"update"}, 2, "missing id"},
		{[]string{"update", "redis-cache"}, 2, "nothing to update"},
		{[]string{"update", "redis-cache", "--json", "--dir", "."}, 2, "nothing to update"},
		{[]string{"update", "redis-cache", "--title", " "}, 2, "title"},
		{[]string{"update", "redis-cache", "--body", "x", "--body-file", "y"}, 2, "mutually exclusive"},
		{[]string{"update", "missing", "--title", "x"}, 3, "not found"},
		{[]string{"update", "redis-cache", "--body-file", "missing.md"}, 1, "missing.md"},
	} {
		_, errOut, code := e.run(tc.args...)
		assertCode(t, tc.args, code, tc.code, errOut)
		if !strings.Contains(errOut, tc.want) {
			t.Errorf("%q: stderr = %q, want %q", tc.args, errOut, tc.want)
		}
	}
	if got := e.showJSON("redis-cache"); !got.Updated.Equal(t0) {
		t.Errorf("failed updates modified the note: %#v", got)
	}
}

func TestDelete(t *testing.T) {
	e := newEnv(t)
	seedOne(e)
	if out := e.mustRun("delete", "redis-cache"); out != "deleted redis-cache\n" {
		t.Errorf("delete stdout = %q", out)
	}
	if _, err := os.Stat(e.notePath("redis-cache")); !os.IsNotExist(err) {
		t.Errorf("note file still exists: %v", err)
	}
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"show", "redis-cache"}, 3},
		{[]string{"delete", "redis-cache"}, 3},
		{[]string{"delete"}, 2},
		{[]string{"delete", "a", "b"}, 2},
	} {
		_, errOut, code := e.run(tc.args...)
		assertCode(t, tc.args, code, tc.code, errOut)
	}
}

func TestDeleteJSON(t *testing.T) {
	e := newEnv(t)
	seedOne(e)
	var got struct {
		ID      string
		Deleted bool
	}
	decodeJSON(t, e.mustRun("delete", "--json", "redis-cache"), &got)
	if got.ID != "redis-cache" || !got.Deleted {
		t.Errorf("delete --json = %#v", got)
	}
}
