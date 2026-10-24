// Package search filters and ranks notes by tags and keywords.
package search

import (
	"cmp"
	"slices"
	"strings"

	"github.com/stormaaja/agent-notes/internal/note"
)

// Field weights added per matching term.
const (
	titleWeight       = 5
	tagWeight         = 4
	descriptionWeight = 3
	bodyWeight        = 1
)

// Query selects notes carrying all Tags and containing every term in Terms.
type Query struct {
	Tags  []string
	Terms []string
}

// Result is a matching note with its relevance score.
type Result struct {
	Note  note.Note
	Score int
}

// ParseTerms splits keyword arguments on whitespace into lowercase, deduplicated terms.
func ParseTerms(s ...string) []string {
	terms := []string{}
	for _, arg := range s {
		for _, f := range strings.Fields(strings.ToLower(arg)) {
			if !slices.Contains(terms, f) {
				terms = append(terms, f)
			}
		}
	}
	return terms
}

// Search returns the notes matching q, best first. Notes must carry every query
// tag and contain every term (case-insensitively) in their title, description,
// tags or body. Ties are broken by most recently updated, then by ID.
// The input slice is not modified.
func Search(notes []note.Note, q Query) []Result {
	tags := note.NormalizeTags(q.Tags)
	terms := ParseTerms(q.Terms...)
	results := []Result{}
	for _, n := range notes {
		noteTags := note.NormalizeTags(n.Tags)
		if !hasAll(noteTags, tags) {
			continue
		}
		if score, ok := scoreNote(n, noteTags, terms); ok {
			results = append(results, Result{Note: n, Score: score})
		}
	}
	slices.SortFunc(results, func(a, b Result) int {
		return cmp.Or(
			cmp.Compare(b.Score, a.Score),
			b.Note.Updated.Compare(a.Note.Updated),
			cmp.Compare(a.Note.ID, b.Note.ID),
		)
	})
	return results
}

func hasAll(have, want []string) bool {
	for _, t := range want {
		if _, found := slices.BinarySearch(have, t); !found {
			return false
		}
	}
	return true
}

// scoreNote reports the note's score and whether every term matched some field.
func scoreNote(n note.Note, tags, terms []string) (int, bool) {
	title := strings.ToLower(n.Title)
	desc := strings.ToLower(n.Description)
	body := strings.ToLower(n.Body)
	total := 0
	for _, term := range terms {
		score := 0
		if strings.Contains(title, term) {
			score += titleWeight
		}
		if slices.ContainsFunc(tags, func(t string) bool { return strings.Contains(t, term) }) {
			score += tagWeight
		}
		if strings.Contains(desc, term) {
			score += descriptionWeight
		}
		if strings.Contains(body, term) {
			score += bodyWeight
		}
		if score == 0 {
			return 0, false
		}
		total += score
	}
	return total, true
}
