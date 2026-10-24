package search

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/stormaaja/agent-notes/internal/note"
)

var base = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

func mk(id string, age int, title, desc, body string, tags ...string) note.Note {
	if tags == nil {
		tags = []string{}
	}
	return note.Note{
		ID: id, Title: title, Description: desc, Body: body, Tags: tags,
		Created: base, Updated: base.Add(-time.Duration(age) * time.Hour),
	}
}

func ids(rs []Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Note.ID
	}
	return out
}

func scores(rs []Result) []int {
	out := make([]int, len(rs))
	for i, r := range rs {
		out[i] = r.Score
	}
	return out
}

func TestSearchFiltering(t *testing.T) {
	notes := []note.Note{
		mk("go-errors", 1, "Go error wrapping", "Use %w with fmt.Errorf", "errors.Is and errors.As", "go", "errors"),
		mk("go-testing", 2, "Table driven tests", "Testing idioms in Go", "use t.Run", "go", "testing"),
		mk("docker", 3, "Docker build cache", "Speed up builds", "layer ordering", "docker", "ci"),
	}
	tests := []struct {
		name string
		q    Query
		want []string
	}{
		{"single tag", Query{Tags: []string{"go"}}, []string{"go-errors", "go-testing"}},
		{"all tags required", Query{Tags: []string{"go", "testing"}}, []string{"go-testing"}},
		{"tag no match", Query{Tags: []string{"go", "docker"}}, []string{}},
		{"tag case and space", Query{Tags: []string{"  GO "}}, []string{"go-errors", "go-testing"}},
		{"tag is exact not substring", Query{Tags: []string{"err"}}, []string{}},
		{"term in title", Query{Terms: []string{"docker"}}, []string{"docker"}},
		{"term in description", Query{Terms: []string{"idioms"}}, []string{"go-testing"}},
		{"term in body", Query{Terms: []string{"layer"}}, []string{"docker"}},
		{"term in tag", Query{Terms: []string{"ci"}}, []string{"docker"}},
		{"all terms required", Query{Terms: []string{"go", "wrapping"}}, []string{"go-errors"}},
		{"term no match", Query{Terms: []string{"go", "kubernetes"}}, []string{}},
		{"term case insensitive", Query{Terms: []string{"DOCKER"}}, []string{"docker"}},
		{"field case insensitive", Query{Terms: []string{"errorf"}}, []string{"go-errors"}},
		{"tags and terms", Query{Tags: []string{"go"}, Terms: []string{"t.run"}}, []string{"go-testing"}},
		{"tags and terms no match", Query{Tags: []string{"docker"}, Terms: []string{"t.run"}}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Search(notes, tt.q)
			if got == nil {
				t.Fatal("Search returned nil, want non-nil slice")
			}
			gotIDs := ids(got)
			slices.Sort(gotIDs)
			want := slices.Clone(tt.want)
			slices.Sort(want)
			if !reflect.DeepEqual(gotIDs, want) {
				t.Errorf("Search(%+v) = %v, want %v", tt.q, gotIDs, want)
			}
		})
	}
}

func TestSearchScoring(t *testing.T) {
	tests := []struct {
		name  string
		n     note.Note
		terms []string
		want  int
	}{
		{"title", mk("a", 0, "Redis cache", "", ""), []string{"redis"}, 5},
		{"tag exact", mk("a", 0, "", "", "", "redis"), []string{"redis"}, 4},
		{"tag substring", mk("a", 0, "", "", "", "redis-cluster"), []string{"redis"}, 4},
		{"description", mk("a", 0, "", "About redis", ""), []string{"redis"}, 3},
		{"body", mk("a", 0, "", "", "redis here"), []string{"redis"}, 1},
		{"all fields", mk("a", 0, "redis", "redis", "redis", "redis"), []string{"redis"}, 13},
		{"field counted once per term", mk("a", 0, "redis redis", "", "redis redis redis", "redis", "redis-x"), []string{"redis"}, 10},
		{"sum over terms", mk("a", 0, "redis cache", "", "eviction"), []string{"redis", "eviction"}, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Search([]note.Note{tt.n}, Query{Terms: tt.terms})
			if len(got) != 1 || got[0].Score != tt.want {
				t.Errorf("scores = %v, want [%d]", scores(got), tt.want)
			}
		})
	}
}

func TestSearchOrdering(t *testing.T) {
	tests := []struct {
		name  string
		notes []note.Note
		q     Query
		want  []string
	}{
		{
			"score desc",
			[]note.Note{
				mk("body", 0, "", "", "redis"),
				mk("title", 5, "redis", "", ""),
				mk("desc", 0, "", "redis", ""),
				mk("tag", 0, "", "", "", "redis"),
			},
			Query{Terms: []string{"redis"}},
			[]string{"title", "tag", "desc", "body"},
		},
		{
			"tie broken by updated desc",
			[]note.Note{
				mk("old", 5, "redis", "", ""),
				mk("new", 1, "redis", "", ""),
				mk("mid", 3, "redis", "", ""),
			},
			Query{Terms: []string{"redis"}},
			[]string{"new", "mid", "old"},
		},
		{
			"tie broken by id asc",
			[]note.Note{
				mk("c", 1, "redis", "", ""),
				mk("a", 1, "redis", "", ""),
				mk("b", 1, "redis", "", ""),
			},
			Query{Terms: []string{"redis"}},
			[]string{"a", "b", "c"},
		},
		{
			"empty query lists all newest first",
			[]note.Note{
				mk("b", 2, "x", "", ""),
				mk("c", 1, "y", "", ""),
				mk("a", 2, "z", "", ""),
				mk("d", 9, "w", "", ""),
			},
			Query{},
			[]string{"c", "a", "b", "d"},
		},
		{
			"tags only scores zero, newest first",
			[]note.Note{
				mk("old", 5, "", "", "", "go"),
				mk("new", 1, "", "", "", "go"),
			},
			Query{Tags: []string{"go"}},
			[]string{"new", "old"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ids(Search(tt.notes, tt.q)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("order = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchEmptyQueryScoresZero(t *testing.T) {
	got := Search([]note.Note{mk("a", 0, "t", "d", "b", "x")}, Query{})
	if !reflect.DeepEqual(scores(got), []int{0}) {
		t.Errorf("scores = %v, want [0]", scores(got))
	}
}

func TestSearchEmptyInput(t *testing.T) {
	for _, in := range [][]note.Note{nil, {}} {
		if got := Search(in, Query{}); got == nil || len(got) != 0 {
			t.Errorf("Search(%v) = %#v, want empty non-nil", in, got)
		}
	}
}

func TestSearchDoesNotMutateInput(t *testing.T) {
	notes := []note.Note{
		mk("b", 3, "redis", "", "", "Go"),
		mk("a", 1, "", "", "redis", "Go"),
		mk("c", 2, "", "redis", "", "Go"),
	}
	orig := make([]note.Note, len(notes))
	for i, n := range notes {
		orig[i] = n
		orig[i].Tags = slices.Clone(n.Tags)
	}
	q := Query{Tags: []string{" GO "}, Terms: []string{"Redis"}}
	if got := Search(notes, q); len(got) != 3 {
		t.Fatalf("got %d results, want 3", len(got))
	}
	if !reflect.DeepEqual(notes, orig) {
		t.Errorf("input mutated:\n got %+v\nwant %+v", notes, orig)
	}
	if !reflect.DeepEqual(q, Query{Tags: []string{" GO "}, Terms: []string{"Redis"}}) {
		t.Errorf("query mutated: %+v", q)
	}
}

func TestParseTerms(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"none", nil, []string{}},
		{"single", []string{"Redis"}, []string{"redis"}},
		{"split whitespace", []string{"go  error\twrapping\n"}, []string{"go", "error", "wrapping"}},
		{"multiple args", []string{"go", "Error Wrapping"}, []string{"go", "error", "wrapping"}},
		{"drop empties", []string{"", "   ", "go"}, []string{"go"}},
		{"dedupe keeps first order", []string{"b a", "B", "c a"}, []string{"b", "a", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseTerms(tt.in...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTerms(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
