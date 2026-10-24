// Package mcpserver exposes a notes store as Model Context Protocol tools.
package mcpserver

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stormaaja/agent-notes/internal/note"
	"github.com/stormaaja/agent-notes/internal/notejson"
	"github.com/stormaaja/agent-notes/internal/search"
	"github.com/stormaaja/agent-notes/internal/store"
)

const defaultLimit = 20

const instructions = `Shared project notes from the repository's .notes/ folder. The notes are committed to git, so they reach teammates, other machines and other agents: treat them as the project's memory.
- Before starting a task, call notes_search with keywords or tags and read the relevant notes with notes_get.
- When you learn something durable and non-obvious (a convention, gotcha, decision or how-to), record it with notes_add instead of private memory. Search first and prefer notes_update over creating a duplicate.
- Keep notes concise: clear title, one-line description, a few lowercase tags. Never store secrets, credentials, tokens or personal data.`

type server struct {
	mu    sync.Mutex
	store *store.Store
	log   *slog.Logger
}

// New returns an MCP server named agent-notes serving the notes in st.
// Warnings about unreadable note files go to logger; nil discards them.
func New(st *store.Store, version string, logger *slog.Logger) *mcp.Server {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	s := &server{store: st, log: logger}
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "agent-notes", Version: version},
		&mcp.ServerOptions{Instructions: instructions},
	)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "notes_search",
		Description: searchDescription,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(false)},
	}, serialize(&s.mu, s.search))
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "notes_get",
		Description: "Read one shared project note in full, including its Markdown body and file path. Use the id from notes_search results.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(false)},
	}, serialize(&s.mu, s.get))
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "notes_add",
		Description: addDescription,
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), OpenWorldHint: ptr(false)},
	}, serialize(&s.mu, s.add))
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "notes_update",
		Description: updateDescription,
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, serialize(&s.mu, s.update))
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "notes_delete",
		Description: "Delete a shared project note by id. The note file is removed from .notes/; the deletion reaches others when committed. Only delete notes that are wrong or obsolete.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, serialize(&s.mu, s.delete))
	return srv
}

func ptr[T any](v T) *T { return &v }

// serialize returns h wrapped to run only while holding mu.
func serialize[In, Out any](mu *sync.Mutex, h mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		mu.Lock()
		defer mu.Unlock()
		return h(ctx, req, in)
	}
}

const searchDescription = `Search the repository's shared notes (.notes/, committed to git and shared with the team and other agents). Call this before starting a task and before adding a note, to reuse existing knowledge and avoid duplicates.
- query: keywords separated by spaces. Every keyword must appear (case-insensitive substring) in the title, description, tags or body.
- tags: every listed tag must be present (AND), compared case-insensitively as whole tags.
- Results are ranked best match first (per keyword: title 5, tag 4, description 3, body 1; ties go to the most recently updated). Without query and tags, all notes are listed newest first.
- limit: maximum results (default 20); total reports how many matched. Bodies are omitted unless full is true; use notes_get to read one note.`

const addDescription = `Create a shared project note in .notes/ (a Markdown file committed to git, so teammates and other agents will see it). Record durable, non-obvious knowledge: conventions, gotchas, decisions, how-tos.
- Search with notes_search first; if a related note exists, use notes_update instead of adding a duplicate.
- The id is derived from the title and never changes. Keep the title clear, the description one line and tags few and lowercase.
- Never store secrets, credentials, tokens or personal data. Skip what the code, git history or README already make obvious.`

const updateDescription = `Change an existing shared project note. Only the fields you pass are changed; omitted fields stay as they are. The id never changes, even with a new title.
- tags replaces all tags (an empty list clears them); add_tags and remove_tags edit the current tags instead.
- description "" clears the description; body replaces the whole Markdown body.
Prefer updating an existing note over adding a near-duplicate. Never store secrets, credentials, tokens or personal data.`

type searchInput struct {
	Query string   `json:"query,omitempty" jsonschema:"keywords separated by spaces; all must match"`
	Tags  []string `json:"tags,omitempty" jsonschema:"only notes carrying all of these tags"`
	Limit int      `json:"limit,omitempty" jsonschema:"maximum number of results (default 20)"`
	Full  bool     `json:"full,omitempty" jsonschema:"include each note's body"`
}

type searchOutput struct {
	Results []notejson.Note `json:"results"`
	Total   int             `json:"total" jsonschema:"number of matching notes before the limit"`
}

func (s *server) search(_ context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, searchOutput, error) {
	limit := in.Limit
	if limit < 0 {
		return nil, searchOutput{}, errors.New("limit must not be negative")
	}
	if limit == 0 {
		limit = defaultLimit
	}
	notes, warnings, err := s.store.List()
	if err != nil {
		return nil, searchOutput{}, err
	}
	for _, w := range warnings {
		s.log.Warn("skipping note", "err", w)
	}
	results := search.Search(notes, search.Query{Tags: in.Tags, Terms: []string{in.Query}})
	out := searchOutput{Results: []notejson.Note{}, Total: len(results)}
	for _, r := range results[:min(limit, len(results))] {
		j := notejson.From(r.Note, s.store.Path(r.Note.ID), in.Full)
		j.Score = &r.Score
		out.Results = append(out.Results, j)
	}
	return nil, out, nil
}

type idInput struct {
	ID string `json:"id" jsonschema:"note id, as returned by notes_search"`
}

func (s *server) get(_ context.Context, _ *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, notejson.Note, error) {
	n, err := s.store.Get(in.ID)
	if err != nil {
		return nil, notejson.Note{}, err
	}
	return nil, s.full(n), nil
}

func (s *server) full(n note.Note) notejson.Note {
	return notejson.From(n, s.store.Path(n.ID), true)
}

type addInput struct {
	Title       string   `json:"title" jsonschema:"short descriptive title; its slug becomes the id"`
	Description string   `json:"description,omitempty" jsonschema:"one-line summary"`
	Tags        []string `json:"tags,omitempty" jsonschema:"a few lowercase topic tags"`
	Body        string   `json:"body,omitempty" jsonschema:"Markdown body with the details"`
}

func (s *server) add(_ context.Context, _ *mcp.CallToolRequest, in addInput) (*mcp.CallToolResult, notejson.Note, error) {
	n, err := s.store.Create(note.Note{Title: in.Title, Description: in.Description, Tags: in.Tags, Body: in.Body})
	if err != nil {
		return nil, notejson.Note{}, err
	}
	return nil, s.full(n), nil
}

type updateInput struct {
	ID          string    `json:"id" jsonschema:"id of the note to change"`
	Title       *string   `json:"title,omitempty" jsonschema:"new title; the id stays the same"`
	Description *string   `json:"description,omitempty" jsonschema:"new one-line description; empty clears it"`
	Tags        *[]string `json:"tags,omitempty" jsonschema:"replace all tags; an empty list clears them"`
	AddTags     []string  `json:"add_tags,omitempty" jsonschema:"tags to add"`
	RemoveTags  []string  `json:"remove_tags,omitempty" jsonschema:"tags to remove"`
	Body        *string   `json:"body,omitempty" jsonschema:"new Markdown body, replacing the old one"`
}

func (s *server) update(_ context.Context, _ *mcp.CallToolRequest, in updateInput) (*mcp.CallToolResult, notejson.Note, error) {
	if in.Title == nil && in.Description == nil && in.Tags == nil && in.Body == nil &&
		len(in.AddTags) == 0 && len(in.RemoveTags) == 0 {
		return nil, notejson.Note{}, errors.New("nothing to update; pass at least one field to change")
	}
	n, err := s.store.Update(in.ID, func(n *note.Note) error {
		if in.Title != nil {
			n.Title = *in.Title
		}
		if in.Description != nil {
			n.Description = *in.Description
		}
		if in.Tags != nil {
			n.Tags = *in.Tags
		}
		n.Tags = note.EditTags(n.Tags, in.AddTags, in.RemoveTags)
		if in.Body != nil {
			n.Body = *in.Body
		}
		return nil
	})
	if err != nil {
		return nil, notejson.Note{}, err
	}
	return nil, s.full(n), nil
}

type deleteOutput struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

func (s *server) delete(_ context.Context, _ *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, deleteOutput, error) {
	if err := s.store.Delete(in.ID); err != nil {
		return nil, deleteOutput{}, err
	}
	return nil, deleteOutput{ID: in.ID, Deleted: true}, nil
}
