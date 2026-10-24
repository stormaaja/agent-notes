package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/stormaaja/agent-notes/internal/memory"
	"github.com/stormaaja/agent-notes/internal/note"
	"github.com/stormaaja/agent-notes/internal/store"
)

// importedTag marks notes created by import-memory. Only notes carrying it
// are updated by a later import.
const importedTag = "imported-memory"

var importMemoryCmd = command{
	name:    "import-memory",
	usage:   "<dir> [--tag t]... [--dry-run]",
	summary: "import Claude Code memory files as notes",
	help: `Imports every *.md file in dir except MEMORY.md, for example
~/.claude/projects/<project>/memory. The memory name becomes the title and,
slugified, the id; the description is kept; tags are imported-memory and the
memory type. Re-importing updates notes whose memory changed and leaves the
rest untouched. Existing notes not created by an import are never changed.

Flags:
  --tag t     add tags to every imported note; repeatable or comma-separated
  --dry-run   report what would happen without writing anything
`,
	run: runImportMemory,
}

const (
	statusCreated   = "created"
	statusUpdated   = "updated"
	statusUnchanged = "unchanged"
	statusSkipped   = "skipped"
)

type importResult struct {
	File   string `json:"file"`
	ID     string `json:"id,omitempty"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func runImportMemory(r *runner, args []string) error {
	fs := r.flagSet("import-memory")
	var tags listFlag
	fs.Var(&tags, "tag", "")
	dryRun := fs.Bool("dry-run", false, "")
	pos, err := r.parseN(fs, args, "dir")
	if err != nil {
		return err
	}
	dir := pos[0]
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(r.env.Cwd, dir)
	}
	if fi, err := os.Stat(dir); err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	s, err := r.store()
	if err != nil {
		return err
	}
	imp := importer{store: s, tags: tags, dryRun: *dryRun, seen: map[string]string{}}
	results := []importResult{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".md" || name == memory.IndexFile || strings.HasPrefix(name, ".") {
			continue
		}
		res, err := imp.importFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		results = append(results, res)
	}
	return r.printImport(results, *dryRun)
}

type importer struct {
	store  *store.Store
	tags   []string
	dryRun bool
	seen   map[string]string // id -> file it was imported from in this run
}

// importFile imports one memory file. Problems with the file itself are
// reported as a skipped result; only failures to write notes are errors.
func (imp *importer) importFile(path string) (importResult, error) {
	res := importResult{File: filepath.Base(path)}
	skip := func(format string, a ...any) (importResult, error) {
		res.Status, res.Reason = statusSkipped, fmt.Sprintf(format, a...)
		return res, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return skip("%v", err)
	}
	m, err := memory.Parse(res.File, data)
	if err != nil {
		return skip("%v", err)
	}
	res.ID = m.ID()
	if from, ok := imp.seen[res.ID]; ok {
		return skip("id %s already imported from %s", res.ID, from)
	}
	imp.seen[res.ID] = res.File

	want := note.Note{
		ID:          res.ID,
		Title:       m.Title(),
		Description: m.Description,
		Tags:        note.NormalizeTags(append([]string{importedTag, m.Type}, imp.tags...)),
		Body:        m.Body,
	}
	old, err := imp.store.Get(res.ID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		res.Status = statusCreated
		if imp.dryRun {
			return res, nil
		}
		_, err = imp.store.Create(want)
		return res, err
	case err != nil:
		return skip("%v", err)
	case !slices.Contains(old.Tags, importedTag):
		return skip("note %s exists and was not imported from memory", res.ID)
	}

	apply := func(n *note.Note) error {
		n.Title, n.Description, n.Body = want.Title, want.Description, want.Body
		// Keep tags added to the note since the last import.
		n.Tags = note.NormalizeTags(append(n.Tags, want.Tags...))
		return nil
	}
	merged := old
	merged.Tags = slices.Clone(old.Tags)
	apply(&merged)
	if merged.Title == old.Title && merged.Description == old.Description &&
		merged.Body == old.Body && slices.Equal(merged.Tags, old.Tags) {
		res.Status = statusUnchanged
		return res, nil
	}
	res.Status = statusUpdated
	if !imp.dryRun {
		_, err = imp.store.Update(res.ID, apply)
	}
	return res, err
}

func (r *runner) printImport(results []importResult, dryRun bool) error {
	if r.json {
		return r.printJSON(results)
	}
	var b strings.Builder
	count := map[string]int{}
	for _, res := range results {
		count[res.Status]++
		if res.Status == statusSkipped {
			fmt.Fprintf(&b, "%s %s: %s\n", res.Status, res.File, res.Reason)
		} else {
			fmt.Fprintf(&b, "%s %s from %s\n", res.Status, res.ID, res.File)
		}
	}
	fmt.Fprintf(&b, "%d created, %d updated, %d unchanged, %d skipped", count[statusCreated],
		count[statusUpdated], count[statusUnchanged], count[statusSkipped])
	if dryRun {
		b.WriteString(" (dry run, nothing written)")
	}
	b.WriteString("\n")
	_, err := fmt.Fprint(r.env.Stdout, b.String())
	return err
}
