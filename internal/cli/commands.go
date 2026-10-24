package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/stormaaja/agent-notes/internal/note"
	"github.com/stormaaja/agent-notes/internal/search"
)

var initCmd = command{
	name:    "init",
	summary: "create the notes directory and print its path",
	run:     runInit,
}

func runInit(r *runner, args []string) error {
	fs := r.flagSet("init")
	if _, err := r.parseN(fs, args); err != nil {
		return err
	}
	dir, err := r.notesDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if r.json {
		return r.printJSON(struct {
			Dir string `json:"dir"`
		}{dir})
	}
	_, err = fmt.Fprintln(r.env.Stdout, dir)
	return err
}

var addCmd = command{
	name:    "add",
	usage:   "--title TITLE [--description TEXT] [--tags a,b] [--body TEXT | --body - | --body-file PATH]",
	summary: "create a note and print its id",
	help: `Flags:
  --title TITLE        note title; its slug becomes the id (required)
  --description TEXT   one-line summary
  --tags a,b           comma-separated tags; repeatable
  --body TEXT          Markdown body; "-" reads it from stdin
  --body-file PATH     read the body from a file
`,
	run: runAdd,
}

func runAdd(r *runner, args []string) error {
	fs := r.flagSet("add")
	var n note.Note
	var tags listFlag
	var body bodyFlags
	fs.StringVar(&n.Title, "title", "", "")
	fs.StringVar(&n.Description, "description", "", "")
	fs.Var(&tags, "tags", "")
	body.register(fs)
	if _, err := r.parseN(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(n.Title) == "" {
		return usagef("add", "--title is required")
	}
	var err error
	if n.Body, _, err = r.readBody(fs, body); err != nil {
		return err
	}
	n.Tags = tags
	s, err := r.store()
	if err != nil {
		return err
	}
	if n, err = s.Create(n); err != nil {
		return err
	}
	return r.printSaved(s, "created", n)
}

var showCmd = command{
	name:    "show",
	usage:   "<id>",
	summary: "print a note",
	run:     runShow,
}

func runShow(r *runner, args []string) error {
	pos, err := r.parseN(r.flagSet("show"), args, "id")
	if err != nil {
		return err
	}
	s, err := r.store()
	if err != nil {
		return err
	}
	n, err := s.Get(pos[0])
	if err != nil {
		return err
	}
	return r.printNote(s, n)
}

const findHelp = `Flags:
  --tag t   only notes with this tag; repeatable or comma-separated, all must match
  --full    include the body in --json output
`

var listCmd = command{
	name:    "list",
	usage:   "[--tag t]... [--full]",
	summary: "list notes, most recently updated first",
	help:    findHelp,
	run:     func(r *runner, args []string) error { return runFind(r, "list", args) },
}

var searchCmd = command{
	name:    "search",
	usage:   "[--tag t]... [--full] [keywords...]",
	summary: "find notes containing all keywords, best match first",
	help:    findHelp,
	run:     func(r *runner, args []string) error { return runFind(r, "search", args) },
}

func runFind(r *runner, name string, args []string) error {
	fs := r.flagSet(name)
	var tags listFlag
	fs.Var(&tags, "tag", "")
	full := fs.Bool("full", false, "")
	var terms []string
	var err error
	if name == "search" {
		terms, err = r.parse(fs, args)
	} else {
		_, err = r.parseN(fs, args)
	}
	if err != nil {
		return err
	}
	s, err := r.store()
	if err != nil {
		return err
	}
	notes, warnings, err := s.List()
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(r.env.Stderr, "warning: %v\n", w)
	}
	results := search.Search(notes, search.Query{Tags: tags, Terms: terms})
	return r.printResults(s, results, name == "search", *full)
}
