package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// memDir writes memory files into a fresh directory and returns its path.
func memDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	writeFiles(t, dir, files)
	return dir
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const (
	memTopLevel = "---\nname: Git amend workflow\ndescription: When to amend vs new commits\ntype: feedback\n---\n\nAmend while iterating on one change.\n"
	memMetadata = "---\nname: removal-commit-type-is-feat\ndescription: Removing code is a feat change\nmetadata:\n  node_type: memory\n  type: project\n  originSessionId: abc\n---\n\nUse feat for removals.\n"
	memPlain    = "Prefers light design docs.\n"
)

func standardMemories(t *testing.T) string {
	dir := memDir(t, map[string]string{
		"feedback_git_amend.md":          memTopLevel,
		"removal-commit-type-is-feat.md": memMetadata,
		"user_profile.md":                memPlain,
		"MEMORY.md":                      "- [Git](feedback_git_amend.md)\n",
		"notes.txt":                      "not markdown",
	})
	if err := os.Mkdir(filepath.Join(dir, "sub.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestImportMemory(t *testing.T) {
	e := newEnv(t)
	dir := standardMemories(t)
	out := e.mustRun("import-memory", dir, "--tag", "Team", "--tag", "a,b")
	want := `created git-amend-workflow from feedback_git_amend.md
created removal-commit-type-is-feat from removal-commit-type-is-feat.md
created user-profile from user_profile.md
3 created, 0 updated, 0 unchanged, 0 skipped
`
	if out != want {
		t.Errorf("stdout =\n%s\nwant\n%s", out, want)
	}

	for _, w := range []jsonNote{
		{ID: "git-amend-workflow", Title: "Git amend workflow", Description: "When to amend vs new commits",
			Tags: []string{"a", "b", "feedback", "imported-memory", "team"}, Body: "Amend while iterating on one change."},
		{ID: "removal-commit-type-is-feat", Title: "Removal commit type is feat", Description: "Removing code is a feat change",
			Tags: []string{"a", "b", "imported-memory", "project", "team"}, Body: "Use feat for removals."},
		{ID: "user-profile", Title: "User profile", Tags: []string{"a", "b", "imported-memory", "team"}, Body: "Prefers light design docs."},
	} {
		got := e.showJSON(w.ID)
		w.Created, w.Updated, w.Path = got.Created, got.Updated, e.notePath(w.ID)
		if !reflect.DeepEqual(got, w) {
			t.Errorf("imported note =\n%#v\nwant\n%#v", got, w)
		}
	}
	entries, _ := os.ReadDir(e.notesDir())
	if len(entries) != 3 {
		t.Errorf("notes dir has %d entries, want 3 (MEMORY.md, non-.md and dirs skipped)", len(entries))
	}
}

func TestImportMemoryRelativeDir(t *testing.T) {
	e := newEnv(t)
	writeFiles(t, e.cwd, map[string]string{"mem.md": memTopLevel})
	e.mustRun("import-memory", ".")
	e.showJSON("git-amend-workflow")
}

func TestImportMemoryIdempotent(t *testing.T) {
	e := newEnv(t)
	dir := standardMemories(t)
	e.mustRun("import-memory", dir)
	before := e.showJSON("git-amend-workflow")

	out := e.mustRun("import-memory", dir)
	if !strings.HasSuffix(out, "0 created, 0 updated, 3 unchanged, 0 skipped\n") ||
		!strings.Contains(out, "unchanged git-amend-workflow from feedback_git_amend.md\n") {
		t.Errorf("re-import stdout =\n%s", out)
	}
	if after := e.showJSON("git-amend-workflow"); !reflect.DeepEqual(after, before) {
		t.Errorf("unchanged re-import modified note:\n%#v\nwas\n%#v", after, before)
	}
}

func TestImportMemoryUpdatesChanged(t *testing.T) {
	e := newEnv(t)
	dir := standardMemories(t)
	e.mustRun("import-memory", dir)
	e.mustRun("update", "git-amend-workflow", "--add-tag", "mine")
	before := e.showJSON("git-amend-workflow")

	writeFiles(t, dir, map[string]string{"feedback_git_amend.md": strings.Replace(memTopLevel, "one change", "a single change", 1)})
	out := e.mustRun("import-memory", dir)
	if !strings.Contains(out, "updated git-amend-workflow from feedback_git_amend.md\n") ||
		!strings.HasSuffix(out, "0 created, 1 updated, 2 unchanged, 0 skipped\n") {
		t.Errorf("stdout =\n%s", out)
	}
	got := e.showJSON("git-amend-workflow")
	if got.Body != "Amend while iterating on a single change." {
		t.Errorf("body = %q", got.Body)
	}
	if !got.Created.Equal(before.Created) || !got.Updated.After(before.Updated) {
		t.Errorf("created %v updated %v, before created %v updated %v", got.Created, got.Updated, before.Created, before.Updated)
	}
	if !reflect.DeepEqual(got.Tags, []string{"feedback", "imported-memory", "mine"}) {
		t.Errorf("tags = %q, want manually added tag kept", got.Tags)
	}
}

func TestImportMemorySkips(t *testing.T) {
	e := newEnv(t)
	e.mustRun("add", "--title", "Git amend workflow", "--body", "hand written")
	dir := memDir(t, map[string]string{
		"a.md":       memTopLevel,
		"bad.md":     "---\nname: [oops\n---\n",
		"open.md":    "---\nname: x\n",
		"dup-one.md": "---\nname: Same name\n---\nfirst\n",
		"dup-two.md": "---\nname: same-name\n---\nsecond\n",
	})
	out := e.mustRun("import-memory", dir)
	for _, want := range []string{
		"skipped a.md: note git-amend-workflow exists and was not imported from memory\n",
		"skipped bad.md: memory: invalid frontmatter",
		"created same-name from dup-one.md\n",
		"skipped dup-two.md: id same-name already imported from dup-one.md\n",
		"skipped open.md: memory: unterminated frontmatter\n",
		"1 created, 0 updated, 0 unchanged, 4 skipped\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}
	if got := e.showJSON("git-amend-workflow"); got.Body != "hand written" {
		t.Errorf("hand-written note overwritten: %q", got.Body)
	}
}

func TestImportMemoryDryRun(t *testing.T) {
	e := newEnv(t)
	dir := standardMemories(t)
	out := e.mustRun("import-memory", "--dry-run", dir)
	if !strings.Contains(out, "created git-amend-workflow from feedback_git_amend.md\n") ||
		!strings.HasSuffix(out, "3 created, 0 updated, 0 unchanged, 0 skipped (dry run, nothing written)\n") {
		t.Errorf("dry-run stdout =\n%s", out)
	}
	if _, err := os.Stat(e.notesDir()); !os.IsNotExist(err) {
		t.Fatalf("dry run created notes dir: %v", err)
	}

	e.mustRun("import-memory", dir)
	before := e.showJSON("user-profile")
	writeFiles(t, dir, map[string]string{"user_profile.md": "Changed.\n"})
	out = e.mustRun("import-memory", dir, "--dry-run")
	if !strings.Contains(out, "updated user-profile from user_profile.md\n") {
		t.Errorf("dry-run stdout =\n%s", out)
	}
	if after := e.showJSON("user-profile"); !reflect.DeepEqual(after, before) {
		t.Errorf("dry run modified note")
	}
}

func TestImportMemoryJSON(t *testing.T) {
	e := newEnv(t)
	e.mustRun("add", "--title", "User profile")
	dir := memDir(t, map[string]string{"user_profile.md": memPlain, "b.md": memTopLevel})
	var got []struct {
		File, ID, Status, Reason string
	}
	decodeJSON(t, e.mustRun("--json", "import-memory", dir), &got)
	want := []struct{ File, ID, Status, Reason string }{
		{"b.md", "git-amend-workflow", "created", ""},
		{"user_profile.md", "user-profile", "skipped", "note user-profile exists and was not imported from memory"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("json =\n%+v\nwant\n%+v", got, want)
	}

	var empty []any
	decodeJSON(t, e.mustRun("import-memory", t.TempDir(), "--json"), &empty)
	if empty == nil || len(empty) != 0 {
		t.Errorf("empty dir json = %v, want []", empty)
	}
}

func TestImportMemoryErrors(t *testing.T) {
	e := newEnv(t)
	writeFiles(t, e.cwd, map[string]string{"file.md": "x"})
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"import-memory"}, 2, "missing dir"},
		{[]string{"import-memory", "a", "b"}, 2, "unexpected argument"},
		{[]string{"import-memory", "missing"}, 1, "missing"},
		{[]string{"import-memory", "file.md"}, 1, "not a directory"},
	} {
		_, errOut, code := e.run(tc.args...)
		assertCode(t, tc.args, code, tc.code, errOut)
		if !strings.Contains(errOut, tc.want) {
			t.Errorf("%q: stderr = %q, want %q", tc.args, errOut, tc.want)
		}
	}
	if out := e.mustRun("help"); !strings.Contains(out, "import-memory") {
		t.Errorf("help does not list import-memory:\n%s", out)
	}
}
