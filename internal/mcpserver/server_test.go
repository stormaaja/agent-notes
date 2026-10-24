package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stormaaja/agent-notes/internal/note"
	"github.com/stormaaja/agent-notes/internal/store"
)

var t0 = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

type fixture struct {
	t       *testing.T
	dir     string
	store   *store.Store
	session *mcp.ClientSession
}

// newFixture connects a client to a server over an empty notes dir. The
// clock advances one minute per call.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "notes")
	calls := 0
	st := store.New(dir, func() time.Time {
		calls++
		return t0.Add(time.Duration(calls) * time.Minute)
	})
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := New(st, "test", nil).Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return &fixture{t: t, dir: dir, store: st, session: cs}
}

func (f *fixture) create(title, desc, body string, tags ...string) note.Note {
	f.t.Helper()
	n, err := f.store.Create(note.Note{Title: title, Description: desc, Body: body, Tags: tags})
	if err != nil {
		f.t.Fatal(err)
	}
	return n
}

// call invokes a tool and returns its result; protocol errors fail the test.
func (f *fixture) call(name string, args any) *mcp.CallToolResult {
	f.t.Helper()
	res, err := f.session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		f.t.Fatalf("%s: protocol error: %v", name, err)
	}
	return res
}

// callOK invokes a tool, requires success and decodes the structured result into out.
func (f *fixture) callOK(name string, args, out any) {
	f.t.Helper()
	res := f.call(name, args)
	if res.IsError {
		f.t.Fatalf("%s(%v): tool error: %s", name, args, text(res))
	}
	data, err := json.Marshal(res.StructuredContent)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		f.t.Fatalf("%s: decode %s: %v", name, data, err)
	}
}

// callErr invokes a tool and requires a tool error whose text contains want.
func (f *fixture) callErr(name string, args any, want string) {
	f.t.Helper()
	res := f.call(name, args)
	if !res.IsError {
		f.t.Fatalf("%s(%v): want tool error, got %s", name, args, text(res))
	}
	if !strings.Contains(text(res), want) {
		f.t.Errorf("%s(%v): error %q does not contain %q", name, args, text(res), want)
	}
}

func text(res *mcp.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}

type noteOut struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Body        *string   `json:"body"`
	Path        string    `json:"path"`
	Score       *int      `json:"score"`
}

type searchOut struct {
	Results []noteOut `json:"results"`
	Total   int       `json:"total"`
}

func ids(notes []noteOut) []string {
	out := []string{}
	for _, n := range notes {
		out = append(out, n.ID)
	}
	return out
}

func TestListTools(t *testing.T) {
	f := newFixture(t)
	res, err := f.session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
		if len(tool.Description) < 50 {
			t.Errorf("%s: description too short: %q", tool.Name, tool.Description)
		}
	}
	slices.Sort(names)
	want := []string{"notes_add", "notes_delete", "notes_get", "notes_search", "notes_update"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("tools = %q, want %q", names, want)
	}
}

func TestInstructions(t *testing.T) {
	f := newFixture(t)
	info := f.session.InitializeResult()
	if !strings.Contains(info.Instructions, "notes_search") {
		t.Errorf("instructions do not mention notes_search: %q", info.Instructions)
	}
	if info.ServerInfo.Name != "agent-notes" || info.ServerInfo.Version != "test" {
		t.Errorf("server info = %+v", info.ServerInfo)
	}
}

func TestAddAndGet(t *testing.T) {
	f := newFixture(t)
	var added noteOut
	f.callOK("notes_add", map[string]any{
		"title":       "Go error wrapping",
		"description": "Wrap with %w",
		"tags":        []string{"Go", "errors"},
		"body":        "Use fmt.Errorf.\n",
	}, &added)
	if added.ID != "go-error-wrapping" || added.Title != "Go error wrapping" ||
		added.Description != "Wrap with %w" || !reflect.DeepEqual(added.Tags, []string{"errors", "go"}) ||
		added.Body == nil || *added.Body != "Use fmt.Errorf." {
		t.Errorf("added = %+v", added)
	}
	if added.Path != filepath.Join(f.dir, "go-error-wrapping.md") {
		t.Errorf("path = %q", added.Path)
	}
	if _, err := os.Stat(added.Path); err != nil {
		t.Errorf("note file: %v", err)
	}

	var got noteOut
	f.callOK("notes_get", map[string]any{"id": "go-error-wrapping"}, &got)
	if !reflect.DeepEqual(got, added) {
		t.Errorf("get = %+v, want %+v", got, added)
	}
}

func TestAddMinimal(t *testing.T) {
	f := newFixture(t)
	var added noteOut
	f.callOK("notes_add", map[string]any{"title": "Only title"}, &added)
	if added.ID != "only-title" || len(added.Tags) != 0 || added.Tags == nil {
		t.Errorf("added = %+v", added)
	}
}

func TestSearch(t *testing.T) {
	f := newFixture(t)
	f.create("Redis cache TTL", "Keep TTLs short", "Long TTLs hide stale data.", "redis", "cache")
	f.create("Deploy checklist", "Steps before deploy", "Flush the redis cache.", "ops")
	f.create("Go errors", "Wrap errors", "", "go")

	var out searchOut
	f.callOK("notes_search", map[string]any{"query": "redis cache"}, &out)
	if got, want := ids(out.Results), []string{"redis-cache-ttl", "deploy-checklist"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ranked ids = %q, want %q", got, want)
	}
	if out.Total != 2 {
		t.Errorf("total = %d, want 2", out.Total)
	}
	for _, r := range out.Results {
		if r.Score == nil || *r.Score <= 0 {
			t.Errorf("%s: score = %v", r.ID, r.Score)
		}
		if r.Body != nil {
			t.Errorf("%s: body included without full", r.ID)
		}
	}
	if *out.Results[0].Score <= *out.Results[1].Score {
		t.Errorf("scores not descending: %d, %d", *out.Results[0].Score, *out.Results[1].Score)
	}

	f.callOK("notes_search", map[string]any{"query": "redis", "tags": []string{"OPS"}}, &out)
	if got, want := ids(out.Results), []string{"deploy-checklist"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tag filter ids = %q, want %q", got, want)
	}

	f.callOK("notes_search", map[string]any{"tags": []string{"redis", "ops"}}, &out)
	if len(out.Results) != 0 {
		t.Errorf("tags are ANDed; got %q", ids(out.Results))
	}

	f.callOK("notes_search", map[string]any{"query": "redis", "full": true}, &out)
	if out.Results[0].Body == nil || *out.Results[0].Body != "Long TTLs hide stale data." {
		t.Errorf("full body = %v", out.Results[0].Body)
	}

	f.callOK("notes_search", map[string]any{"query": "nomatch"}, &out)
	if out.Results == nil || len(out.Results) != 0 || out.Total != 0 {
		t.Errorf("no match = %+v", out)
	}
}

func TestSearchListsAllNewestFirst(t *testing.T) {
	f := newFixture(t)
	f.create("First", "", "")
	f.create("Second", "", "")
	var out searchOut
	f.callOK("notes_search", map[string]any{}, &out)
	if got, want := ids(out.Results), []string{"second", "first"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %q, want %q", got, want)
	}
}

func TestSearchLimit(t *testing.T) {
	f := newFixture(t)
	for i := range 25 {
		f.create("Note "+string(rune('a'+i)), "", "")
	}
	var out searchOut
	f.callOK("notes_search", map[string]any{}, &out)
	if len(out.Results) != 20 || out.Total != 25 {
		t.Errorf("default limit: %d results, total %d; want 20, 25", len(out.Results), out.Total)
	}
	f.callOK("notes_search", map[string]any{"limit": 3}, &out)
	if got, want := ids(out.Results), []string{"note-y", "note-x", "note-w"}; !reflect.DeepEqual(got, want) {
		t.Errorf("limit 3 ids = %q, want %q", got, want)
	}
	f.callErr("notes_search", map[string]any{"limit": -1}, "limit")
}

func TestUpdatePartial(t *testing.T) {
	f := newFixture(t)
	orig := f.create("Redis cache TTL", "Keep TTLs short", "Body text", "redis", "cache")

	var got noteOut
	f.callOK("notes_update", map[string]any{"id": orig.ID, "add_tags": []string{"Ops"}, "remove_tags": []string{"cache"}}, &got)
	if got.Title != orig.Title || got.Description != orig.Description || got.Body == nil || *got.Body != orig.Body {
		t.Errorf("untouched fields changed: %+v", got)
	}
	if !reflect.DeepEqual(got.Tags, []string{"ops", "redis"}) {
		t.Errorf("tags = %q", got.Tags)
	}
	if !got.Updated.After(orig.Updated) || !got.Created.Equal(orig.Created) {
		t.Errorf("timestamps: created %v updated %v (orig %v)", got.Created, got.Updated, orig.Updated)
	}

	f.callOK("notes_update", map[string]any{"id": orig.ID, "title": "Valkey cache TTL", "description": "", "body": "New body"}, &got)
	if got.ID != orig.ID || got.Title != "Valkey cache TTL" || got.Description != "" || *got.Body != "New body" ||
		!reflect.DeepEqual(got.Tags, []string{"ops", "redis"}) {
		t.Errorf("after second update: %+v", got)
	}

	f.callOK("notes_update", map[string]any{"id": orig.ID, "tags": []string{"x", "Y"}}, &got)
	if !reflect.DeepEqual(got.Tags, []string{"x", "y"}) {
		t.Errorf("replaced tags = %q", got.Tags)
	}
	f.callOK("notes_update", map[string]any{"id": orig.ID, "tags": []string{}}, &got)
	if len(got.Tags) != 0 {
		t.Errorf("cleared tags = %q", got.Tags)
	}

	stored, err := f.store.Get(orig.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Title != "Valkey cache TTL" || stored.Body != "New body" {
		t.Errorf("stored = %+v", stored)
	}
}

func TestDelete(t *testing.T) {
	f := newFixture(t)
	n := f.create("Doomed", "", "")
	var out struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	}
	f.callOK("notes_delete", map[string]any{"id": n.ID}, &out)
	if out.ID != "doomed" || !out.Deleted {
		t.Errorf("delete = %+v", out)
	}
	if _, err := os.Stat(f.store.Path(n.ID)); !os.IsNotExist(err) {
		t.Errorf("file still exists: %v", err)
	}
}

func TestToolErrors(t *testing.T) {
	f := newFixture(t)
	f.create("Existing", "", "")
	tests := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"notes_get", map[string]any{"id": "missing"}, "not found"},
		{"notes_get", map[string]any{"id": "Bad ID"}, "invalid note id"},
		{"notes_delete", map[string]any{"id": "missing"}, "not found"},
		{"notes_update", map[string]any{"id": "missing", "title": "x"}, "not found"},
		{"notes_update", map[string]any{"id": "existing"}, "nothing to update"},
		{"notes_update", map[string]any{"id": "existing", "title": "  "}, "title is empty"},
		{"notes_add", map[string]any{"title": " "}, "title is empty"},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			f.t = t
			f.callErr(tt.tool, tt.args, tt.want)
		})
	}
}

func TestConcurrentUpdatesAreSerialized(t *testing.T) {
	f := newFixture(t)
	n := f.create("Shared", "", "")
	const calls = 20
	errs := make(chan error, calls)
	for i := range calls {
		go func() {
			res, err := f.session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "notes_update",
				Arguments: map[string]any{"id": n.ID, "add_tags": []string{fmt.Sprintf("t%02d", i)}},
			})
			if err == nil && res.IsError {
				err = fmt.Errorf("tool error: %s", text(res))
			}
			errs <- err
		}()
	}
	for range calls {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	stored, err := f.store.Get(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Tags) != calls {
		t.Errorf("tags = %q, want %d tags; concurrent updates were lost", stored.Tags, calls)
	}
}
