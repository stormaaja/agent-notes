// Package notejson defines the JSON shape of notes shared by the CLI and the MCP server.
package notejson

import (
	"time"

	"github.com/stormaaja/agent-notes/internal/note"
)

// Note is a note as printed by --json and returned by the MCP tools.
type Note struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Body        string    `json:"body,omitempty"`
	Path        string    `json:"path"`
	Score       *int      `json:"score,omitempty"`
}

// From converts n stored at path. The body is included only if withBody is set.
func From(n note.Note, path string, withBody bool) Note {
	j := Note{
		ID:          n.ID,
		Title:       n.Title,
		Description: n.Description,
		Tags:        n.Tags,
		Created:     n.Created,
		Updated:     n.Updated,
		Path:        path,
	}
	if j.Tags == nil {
		j.Tags = []string{}
	}
	if withBody {
		j.Body = n.Body
	}
	return j
}
