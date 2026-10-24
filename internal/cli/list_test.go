package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// seed adds alpha, beta and gamma, updated in that order.
func seed(e *testEnv) {
	e.t.Helper()
	e.mustRun("add", "--title", "Alpha", "--description", "First note", "--tags", "go", "--body", "redis cache tips")
	e.mustRun("add", "--title", "Beta", "--tags", "go,redis", "--body", "cache invalidation")
	e.mustRun("add", "--title", "Gamma", "--body", "unrelated")
}

func ids(notes []jsonNote) []string {
	out := []string{}
	for _, n := range notes {
		out = append(out, n.ID)
	}
	return out
}

func TestList(t *testing.T) {
	e := newEnv(t)
	seed(e)
	want := "gamma  Gamma\nbeta  Beta  [go, redis]\nalpha  Alpha  [go]\n    First note\n"
	if out := e.mustRun("list"); out != want {
		t.Errorf("list =\n%s\nwant\n%s", out, want)
	}
}

func TestListByTag(t *testing.T) {
	e := newEnv(t)
	seed(e)
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"list", "--tag", "go"}, []string{"beta", "alpha"}},
		{[]string{"list", "--tag", "GO,redis"}, []string{"beta"}},
		{[]string{"list", "--tag", "go", "--tag", "redis"}, []string{"beta"}},
		{[]string{"list", "--tag", "missing"}, []string{}},
	} {
		var got []jsonNote
		decodeJSON(t, e.mustRun(append(tc.args, "--json")...), &got)
		if !reflect.DeepEqual(ids(got), tc.want) {
			t.Errorf("%q: ids = %q, want %q", tc.args, ids(got), tc.want)
		}
	}
}

func TestListJSON(t *testing.T) {
	e := newEnv(t)
	seed(e)
	var got []jsonNote
	decodeJSON(t, e.mustRun("--json", "list"), &got)
	if len(got) != 3 {
		t.Fatalf("got %d notes, want 3", len(got))
	}
	gamma := got[0]
	want := jsonNote{ID: "gamma", Title: "Gamma", Tags: []string{}, Created: gamma.Created, Updated: gamma.Updated, Path: e.notePath("gamma")}
	if !reflect.DeepEqual(gamma, want) || gamma.Created.IsZero() {
		t.Errorf("list --json[0] =\n%#v\nwant\n%#v", gamma, want)
	}
	if out := e.mustRun("list", "--json"); strings.Contains(out, `"body"`) || strings.Contains(out, `"score"`) || !strings.Contains(out, `"tags": []`) {
		t.Errorf("list --json shape:\n%s", out)
	}

	decodeJSON(t, e.mustRun("list", "--json", "--full"), &got)
	if got[0].Body != "unrelated" {
		t.Errorf("list --full body = %q, want unrelated", got[0].Body)
	}
}

func TestListEmpty(t *testing.T) {
	e := newEnv(t)
	if out := e.mustRun("list"); out != "No notes found.\n" {
		t.Errorf("list = %q", out)
	}
	if out := e.mustRun("list", "--json"); out != "[]\n" {
		t.Errorf("list --json = %q", out)
	}
	if _, err := os.Stat(e.notesDir()); !os.IsNotExist(err) {
		t.Errorf("list created notes dir: %v", err)
	}
}

func TestSearch(t *testing.T) {
	e := newEnv(t)
	seed(e)
	want := "beta  Beta  [go, redis]\nalpha  Alpha  [go]\n    First note\n"
	if out := e.mustRun("search", "redis", "cache"); out != want {
		t.Errorf("search =\n%s\nwant\n%s", out, want)
	}
	if out := e.mustRun("search", "nothing-matches"); out != "No notes found.\n" {
		t.Errorf("search without hits = %q", out)
	}
}

func TestSearchJSONInterspersed(t *testing.T) {
	e := newEnv(t)
	seed(e)
	var got []jsonNote
	decodeJSON(t, e.mustRun("search", "redis", "--tag", "go", "cache", "--json"), &got)
	if want := []string{"beta", "alpha"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("ids = %q, want %q", ids(got), want)
	}
	// beta: redis tag 4 + cache body 1; alpha: redis body 1 + cache body 1.
	for i, score := range []int{5, 2} {
		if got[i].Score == nil || *got[i].Score != score {
			t.Errorf("%s score = %v, want %d", got[i].ID, got[i].Score, score)
		}
		if got[i].Body != "" {
			t.Errorf("%s has body without --full", got[i].ID)
		}
	}

	decodeJSON(t, e.mustRun("--json", "search", "--full", "--tag=redis"), &got)
	if len(got) != 1 || got[0].Body != "cache invalidation" || got[0].Score == nil || *got[0].Score != 0 {
		t.Errorf("search --full --tag=redis = %#v", got)
	}

	decodeJSON(t, e.mustRun("search", "--json", "--", "--tag"), &got)
	if len(got) != 0 {
		t.Errorf("search -- --tag treated --tag as a flag: %#v", got)
	}
}

func TestListWarnsOnCorruptFiles(t *testing.T) {
	e := newEnv(t)
	seed(e)
	if err := os.WriteFile(filepath.Join(e.notesDir(), "broken.md"), []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"list"}, {"search", "cache"}, {"list", "--json"}} {
		out, errOut, code := e.run(args...)
		assertCode(t, args, code, 0, errOut)
		if !strings.HasPrefix(errOut, "warning: ") || !strings.Contains(errOut, "broken.md") {
			t.Errorf("%q: stderr = %q, want warning about broken.md", args, errOut)
		}
		if !strings.Contains(out, "beta") {
			t.Errorf("%q: stdout = %q, want remaining notes", args, out)
		}
	}
}

func TestListUsageErrors(t *testing.T) {
	e := newEnv(t)
	for _, args := range [][]string{{"list", "stray"}, {"list", "--tag"}, {"search", "--bogus"}, {"show", "--full"}} {
		_, errOut, code := e.run(args...)
		assertCode(t, args, code, 2, errOut)
	}
}
