package store

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stormaaja/agent-notes/internal/note"
)

var t0 = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

// fakeClock returns t0 plus one minute per call, with sub-second noise
// and a non-UTC zone so truncation and normalization are exercised.
func fakeClock() func() time.Time {
	zone := time.FixedZone("EEST", 3*3600)
	calls := 0
	return func() time.Time {
		tm := t0.Add(time.Duration(calls)*time.Minute + 500*time.Millisecond)
		calls++
		return tm.In(zone)
	}
}

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".notes")
	return New(dir, fakeClock()), dir
}

func mustCreate(t *testing.T, s *Store, n note.Note) note.Note {
	t.Helper()
	got, err := s.Create(n)
	if err != nil {
		t.Fatalf("Create(%q): %v", n.Title, err)
	}
	return got
}

func assertDirFiles(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("dir files = %q, want %q", got, want)
	}
}

func TestCreate(t *testing.T) {
	s, dir := newStore(t)
	got := mustCreate(t, s, note.Note{
		Title:       "  Go error wrapping ",
		Description: "Wrap with %w",
		Tags:        []string{"Go", " errors", "go"},
		Body:        "body",
	})
	want := note.Note{
		ID:          "go-error-wrapping",
		Title:       "Go error wrapping",
		Description: "Wrap with %w",
		Tags:        []string{"errors", "go"},
		Created:     t0,
		Updated:     t0,
		Body:        "body",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Create =\n%#v\nwant\n%#v", got, want)
	}

	data, err := os.ReadFile(filepath.Join(dir, "go-error-wrapping.md"))
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := note.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(onDisk, want) {
		t.Errorf("on disk =\n%#v\nwant\n%#v", onDisk, want)
	}
	for path, mode := range map[string]os.FileMode{dir: 0o755 | os.ModeDir, filepath.Join(dir, "go-error-wrapping.md"): 0o644} {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		// Allow the process umask to clear bits, but never add any.
		if fi.Mode()&^mode != 0 {
			t.Errorf("%s mode = %v, want at most %v", path, fi.Mode(), mode)
		}
	}
	assertDirFiles(t, dir, "go-error-wrapping.md")
}

func TestCreateRequiresTitle(t *testing.T) {
	s, dir := newStore(t)
	for _, title := range []string{"", "   "} {
		if _, err := s.Create(note.Note{Title: title}); !errors.Is(err, ErrEmptyTitle) {
			t.Errorf("Create(title %q) err = %v, want ErrEmptyTitle", title, err)
		}
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dir created on failed Create: %v", err)
	}
}

func TestCreateCollisionAppendsSuffix(t *testing.T) {
	s, dir := newStore(t)
	var ids []string
	for range 3 {
		ids = append(ids, mustCreate(t, s, note.Note{Title: "Same title"}).ID)
	}
	want := []string{"same-title", "same-title-2", "same-title-3"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %q, want %q", ids, want)
	}
	assertDirFiles(t, dir, "same-title-2.md", "same-title-3.md", "same-title.md")
}

func TestCreateWithExplicitID(t *testing.T) {
	s, _ := newStore(t)
	got := mustCreate(t, s, note.Note{ID: "custom-id", Title: "Other title"})
	if got.ID != "custom-id" {
		t.Errorf("ID = %q, want custom-id", got.ID)
	}
	if _, err := s.Create(note.Note{ID: "custom-id", Title: "Again"}); !errors.Is(err, ErrExists) {
		t.Errorf("duplicate explicit id err = %v, want ErrExists", err)
	}
	if _, err := s.Create(note.Note{ID: "../evil", Title: "x"}); !errors.Is(err, ErrInvalidID) {
		t.Errorf("invalid explicit id err = %v, want ErrInvalidID", err)
	}
	if n, err := s.Get("custom-id"); err != nil || n.Title != "Other title" {
		t.Errorf("original note changed: %#v, %v", n, err)
	}
}

func TestGet(t *testing.T) {
	s, _ := newStore(t)
	created := mustCreate(t, s, note.Note{Title: "Hello", Tags: []string{"x"}, Body: "b"})
	got, err := s.Get("hello")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, created) {
		t.Errorf("Get =\n%#v\nwant\n%#v", got, created)
	}
}

func TestGetNotFound(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get on missing dir err = %v, want ErrNotFound", err)
	}
	mustCreate(t, s, note.Note{Title: "Other"})
	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get missing err = %v, want ErrNotFound", err)
	}
}

func TestGetUsesFileNameAsID(t *testing.T) {
	s, dir := newStore(t)
	mkdirs(t, dir)
	data := "---\nid: something-else\ntitle: Hand edited\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "hand-edited.md"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("hand-edited")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "hand-edited" {
		t.Errorf("ID = %q, want hand-edited", got.ID)
	}
}

func TestGetCorruptFile(t *testing.T) {
	s, dir := newStore(t)
	mkdirs(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "broken.md"), []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := s.Get("broken")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want parse error", err)
	}
	if !strings.Contains(err.Error(), "broken.md") {
		t.Errorf("err %q does not name the file", err)
	}
}

func TestInvalidIDs(t *testing.T) {
	s, dir := newStore(t)
	mkdirs(t, filepath.Dir(dir))
	if err := os.WriteFile(filepath.Join(filepath.Dir(dir), "secret.md"), []byte("---\nid: secret\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"", "../secret", "..", ".", "a/b", `a\b`, "/etc/passwd", "UPPER", "with space",
		"-leading", "trailing-", "double--dash", "x.md", "ä",
	} {
		if _, err := s.Get(id); !errors.Is(err, ErrInvalidID) {
			t.Errorf("Get(%q) err = %v, want ErrInvalidID", id, err)
		}
	}
}

func TestListMissingDir(t *testing.T) {
	s, _ := newStore(t)
	notes, warnings, err := s.List()
	if err != nil || len(notes) != 0 || len(warnings) != 0 {
		t.Errorf("List = %v, %v, %v; want empty", notes, warnings, err)
	}
}

func TestListOrder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".notes")
	now := t0
	s := New(dir, func() time.Time { return now })
	mustCreate(t, s, note.Note{Title: "Old"})
	now = t0.Add(time.Hour)
	mustCreate(t, s, note.Note{Title: "Beta"})
	mustCreate(t, s, note.Note{Title: "Alpha"})
	now = t0.Add(30 * time.Minute)
	mustCreate(t, s, note.Note{Title: "Middle"})

	notes, warnings, err := s.List()
	if err != nil || len(warnings) != 0 {
		t.Fatalf("List: %v, warnings %v", err, warnings)
	}
	var ids []string
	for _, n := range notes {
		ids = append(ids, n.ID)
	}
	want := []string{"alpha", "beta", "middle", "old"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %q, want %q", ids, want)
	}
}

func TestListSkipsBadFiles(t *testing.T) {
	s, dir := newStore(t)
	good := mustCreate(t, s, note.Note{Title: "Good"})
	files := map[string]string{
		"broken.md":       "no frontmatter",
		"Bad Name.md":     "---\nid: bad-name\n---\n",
		"notes.txt":       "not markdown",
		".tmp-123.md":     "---\nid: leftover\n---\n",
		"sub/nested.md":   "---\nid: nested\n---\n",
		"unterminated.md": "---\nid: x\n",
	}
	for name, data := range files {
		path := filepath.Join(dir, name)
		mkdirs(t, filepath.Dir(path))
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	notes, warnings, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(notes, []note.Note{good}) {
		t.Errorf("notes = %#v, want only %q", notes, good.ID)
	}
	if len(warnings) != 3 {
		t.Fatalf("warnings = %v, want 3", warnings)
	}
	for i, name := range []string{"Bad Name.md", "broken.md", "unterminated.md"} {
		if !strings.Contains(warnings[i].Error(), name) {
			t.Errorf("warning %d = %q, want mention of %s", i, warnings[i], name)
		}
	}
}

func TestUpdate(t *testing.T) {
	s, dir := newStore(t)
	created := mustCreate(t, s, note.Note{Title: "Original", Tags: []string{"a"}, Body: "old"})
	got, err := s.Update("original", func(n *note.Note) error {
		n.ID = "hijack"
		n.Created = time.Time{}
		n.Title = " Renamed "
		n.Description = "desc"
		n.Tags = append(n.Tags, "B", "a")
		n.Body = "new"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := note.Note{
		ID:          "original",
		Title:       "Renamed",
		Description: "desc",
		Tags:        []string{"a", "b"},
		Created:     created.Created,
		Updated:     t0.Add(time.Minute),
		Body:        "new",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Update =\n%#v\nwant\n%#v", got, want)
	}
	if onDisk, err := s.Get("original"); err != nil || !reflect.DeepEqual(onDisk, want) {
		t.Errorf("Get after update = %#v, %v", onDisk, err)
	}
	assertDirFiles(t, dir, "original.md")
}

func TestUpdateRejectsEmptyTitle(t *testing.T) {
	s, _ := newStore(t)
	created := mustCreate(t, s, note.Note{Title: "Keep"})
	_, err := s.Update("keep", func(n *note.Note) error { n.Title = " "; return nil })
	if !errors.Is(err, ErrEmptyTitle) {
		t.Errorf("err = %v, want ErrEmptyTitle", err)
	}
	if got, _ := s.Get("keep"); !reflect.DeepEqual(got, created) {
		t.Errorf("note changed: %#v", got)
	}
}

func TestUpdateFnError(t *testing.T) {
	s, _ := newStore(t)
	created := mustCreate(t, s, note.Note{Title: "Keep"})
	boom := errors.New("boom")
	_, err := s.Update("keep", func(n *note.Note) error { n.Body = "changed"; return boom })
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want boom", err)
	}
	if got, _ := s.Get("keep"); !reflect.DeepEqual(got, created) {
		t.Errorf("note changed: %#v", got)
	}
}

func TestUpdateMissingAndInvalid(t *testing.T) {
	s, dir := newStore(t)
	called := false
	fn := func(*note.Note) error { called = true; return nil }
	if _, err := s.Update("missing", fn); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing err = %v, want ErrNotFound", err)
	}
	if _, err := s.Update("../x", fn); !errors.Is(err, ErrInvalidID) {
		t.Errorf("invalid err = %v, want ErrInvalidID", err)
	}
	if called {
		t.Error("fn called for missing note")
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("dir created by failed update: %v", err)
	}
}

func TestDelete(t *testing.T) {
	s, dir := newStore(t)
	mustCreate(t, s, note.Note{Title: "Gone"})
	mustCreate(t, s, note.Note{Title: "Stays"})
	if err := s.Delete("gone"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("gone"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete err = %v, want ErrNotFound", err)
	}
	assertDirFiles(t, dir, "stays.md")
	if err := s.Delete("gone"); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete err = %v, want ErrNotFound", err)
	}
	if err := s.Delete("../stays"); !errors.Is(err, ErrInvalidID) {
		t.Errorf("invalid delete err = %v, want ErrInvalidID", err)
	}
}

func TestPath(t *testing.T) {
	s, dir := newStore(t)
	n := mustCreate(t, s, note.Note{Title: "Path me"})
	if got, want := s.Path(n.ID), filepath.Join(dir, "path-me.md"); got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

func TestCreateAndUpdateReturnStoredForm(t *testing.T) {
	s, _ := newStore(t)
	created := mustCreate(t, s, note.Note{Title: "Trim", Body: "\n\nbody\n\n"})
	if got, err := s.Get(created.ID); err != nil || !reflect.DeepEqual(created, got) {
		t.Errorf("Create = %#v, Get = %#v (err %v)", created, got, err)
	}
	updated, err := s.Update(created.ID, func(n *note.Note) error {
		n.Body = "\nnew\n"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.Get(created.ID); err != nil || !reflect.DeepEqual(updated, got) {
		t.Errorf("Update = %#v, Get = %#v (err %v)", updated, got, err)
	}
}
