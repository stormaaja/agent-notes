package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/stormaaja/agent-notes/internal/note"
	"github.com/stormaaja/agent-notes/internal/notejson"
	"github.com/stormaaja/agent-notes/internal/search"
	"github.com/stormaaja/agent-notes/internal/store"
)

func toJSON(s *store.Store, n note.Note, withBody bool) notejson.Note {
	return notejson.From(n, s.Path(n.ID), withBody)
}

func (r *runner) printNote(s *store.Store, n note.Note) error {
	if r.json {
		return r.printJSON(toJSON(s, n, true))
	}
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\ntitle: %s\n", n.ID, n.Title)
	if n.Description != "" {
		fmt.Fprintf(&b, "description: %s\n", n.Description)
	}
	fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(n.Tags, ", "))
	fmt.Fprintf(&b, "created: %s\nupdated: %s\n", n.Created.Format(time.RFC3339), n.Updated.Format(time.RFC3339))
	fmt.Fprintf(&b, "path: %s\n---\n", s.Path(n.ID))
	if n.Body != "" {
		b.WriteString("\n" + n.Body + "\n")
	}
	_, err := fmt.Fprint(r.env.Stdout, b.String())
	return err
}

// printSaved reports a created or updated note.
func (r *runner) printSaved(s *store.Store, verb string, n note.Note) error {
	if r.json {
		return r.printJSON(toJSON(s, n, true))
	}
	_, err := fmt.Fprintf(r.env.Stdout, "%s %s (%s)\n", verb, n.ID, s.Path(n.ID))
	return err
}

func (r *runner) printResults(s *store.Store, results []search.Result, withScore, withBody bool) error {
	if r.json {
		out := make([]notejson.Note, len(results))
		for i, res := range results {
			out[i] = toJSON(s, res.Note, withBody)
			if withScore {
				out[i].Score = &res.Score
			}
		}
		return r.printJSON(out)
	}
	if len(results) == 0 {
		_, err := fmt.Fprintln(r.env.Stdout, "No notes found.")
		return err
	}
	var b strings.Builder
	for _, res := range results {
		n := res.Note
		b.WriteString(n.ID + "  " + n.Title)
		if len(n.Tags) > 0 {
			b.WriteString("  [" + strings.Join(n.Tags, ", ") + "]")
		}
		b.WriteString("\n")
		if n.Description != "" {
			b.WriteString("    " + n.Description + "\n")
		}
	}
	_, err := fmt.Fprint(r.env.Stdout, b.String())
	return err
}
