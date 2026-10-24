package memory

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		name, file, data string
		want             Memory
	}{
		{"top-level type", "feedback_git.md", "---\nname: Git amend workflow\ndescription: When to amend\ntype: feedback\n---\n\nAmend while iterating.\n",
			Memory{Name: "Git amend workflow", Description: "When to amend", Type: "feedback", Body: "Amend while iterating."}},
		{"metadata type", "x.md", "---\nname: commit-type-is-feat\ndescription: \"Removal is `feat`\"\nmetadata:\n  node_type: memory\n  type: project\n  modified: 2026-10-21T08:16:20.933Z\n---\nBody\n\n---\n\nMore\n",
			Memory{Name: "commit-type-is-feat", Description: "Removal is `feat`", Type: "project", Body: "Body\n\n---\n\nMore"}},
		{"crlf", "x.md", "---\r\nname: N\r\ntype: user\r\n---\r\nB\r\n",
			Memory{Name: "N", Type: "user", Body: "B"}},
		{"no frontmatter", "user_profile.md", "\nJust text.\n",
			Memory{Name: "user_profile", Body: "Just text."}},
		{"no name", "prefers-light.md", "---\ndescription: D\n---\nB",
			Memory{Name: "prefers-light", Description: "D", Body: "B"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.file, []byte(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("Parse =\n%#v\nwant\n%#v", got, tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse("x.md", []byte("---\nname: x\n")); !errors.Is(err, ErrUnterminated) {
		t.Errorf("unterminated: err = %v", err)
	}
	if _, err := Parse("x.md", []byte("---\nname: [x\n---\n")); err == nil {
		t.Error("invalid yaml: want error")
	}
}

func TestTitleAndID(t *testing.T) {
	for _, tc := range []struct{ name, title, id string }{
		{"Git amend workflow", "Git amend workflow", "git-amend-workflow"},
		{"removal-commit-type-is-feat", "Removal commit type is feat", "removal-commit-type-is-feat"},
		{"user_profile", "User profile", "user-profile"},
		{"notes", "Notes", "notes"},
		{"API v2 notes", "API v2 notes", "api-v2-notes"},
		{"Mixed-Case-Slug", "Mixed-Case-Slug", "mixed-case-slug"},
	} {
		m := Memory{Name: tc.name}
		if got := m.Title(); got != tc.title {
			t.Errorf("Title(%q) = %q, want %q", tc.name, got, tc.title)
		}
		if got := m.ID(); got != tc.id {
			t.Errorf("ID(%q) = %q, want %q", tc.name, got, tc.id)
		}
	}
}
