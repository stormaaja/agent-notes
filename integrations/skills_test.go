// Package integrations holds agent integrations; its tests check the bundled skills.
package integrations

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"gopkg.in/yaml.v3"
)

// skillName follows the Agent Skills naming rules: lowercase letters, digits and single
// hyphens, not at the start or end.
var skillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestSkills(t *testing.T) {
	for _, tool := range []string{"claude-code"} {
		t.Run(tool, func(t *testing.T) {
			path := filepath.Join(tool, "agent-notes", "SKILL.md")
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing skill: %v", err)
			}
		})
	}

	paths, err := filepath.Glob(filepath.Join("*", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	misplaced, _ := filepath.Glob(filepath.Join("*", "SKILL.md"))
	for _, p := range misplaced {
		t.Errorf("%s: want <tool>/<skill-name>/SKILL.md", p)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			rest, ok := bytes.CutPrefix(data, []byte("---\n"))
			if !ok {
				t.Fatal("no frontmatter")
			}
			front, _, ok := bytes.Cut(rest, []byte("\n---\n"))
			if !ok {
				t.Fatal("unterminated frontmatter")
			}
			var fm struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
			}
			if err := yaml.Unmarshal(front, &fm); err != nil {
				t.Fatalf("frontmatter: %v", err)
			}
			if !skillName.MatchString(fm.Name) || len(fm.Name) > 64 {
				t.Errorf("name %q: want 1-64 lowercase letters, digits and single hyphens", fm.Name)
			}
			if dir := filepath.Base(filepath.Dir(path)); fm.Name != dir {
				t.Errorf("name %q does not match directory %q", fm.Name, dir)
			}
			if fm.Description == "" || len(fm.Description) > 1024 {
				t.Errorf("description: %d chars, want 1-1024", len(fm.Description))
			}
		})
	}
}
