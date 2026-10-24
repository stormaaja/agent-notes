// Package memory reads Claude Code memory files: Markdown with optional YAML
// frontmatter holding name, description and a type either at the top level
// or under metadata.
package memory

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/stormaaja/agent-notes/internal/note"
)

// IndexFile is the memory index Claude Code keeps next to the memory files.
const IndexFile = "MEMORY.md"

var ErrUnterminated = errors.New("memory: unterminated frontmatter")

// Memory is one memory file. Name falls back to the file name without extension.
type Memory struct {
	Name        string
	Description string
	Type        string
	Body        string
}

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Metadata    struct {
		Type string `yaml:"type"`
	} `yaml:"metadata"`
}

// Parse decodes the memory file named file. A file without frontmatter is
// all body. CRLF line endings are accepted; the body is trimmed of
// surrounding blank lines.
func Parse(file string, data []byte) (Memory, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	m := Memory{}
	body := text
	if rest, ok := strings.CutPrefix(text, "---\n"); ok {
		front, after, ok := cutAtDelimiterLine(rest)
		if !ok {
			return Memory{}, ErrUnterminated
		}
		var fm frontmatter
		if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
			return Memory{}, fmt.Errorf("memory: invalid frontmatter: %w", err)
		}
		m.Name = strings.TrimSpace(fm.Name)
		m.Description = strings.TrimSpace(fm.Description)
		m.Type = strings.TrimSpace(fm.Type)
		if m.Type == "" {
			m.Type = strings.TrimSpace(fm.Metadata.Type)
		}
		body = after
	}
	if m.Name == "" {
		m.Name = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	}
	m.Body = strings.Trim(body, "\n")
	return m, nil
}

func cutAtDelimiterLine(s string) (before, after string, found bool) {
	for i := 0; i <= len(s); {
		line, _, _ := strings.Cut(s[i:], "\n")
		if line == "---" {
			return s[:i], s[i+len(line):], true
		}
		i += len(line) + 1
	}
	return "", "", false
}

var slugName = regexp.MustCompile(`^[a-z0-9]+([-_][a-z0-9]+)*$`)

// Title is the name, or for a lowercase kebab or snake case slug, the slug
// with separators turned into spaces and the first letter capitalized.
func (m Memory) Title() string {
	if !slugName.MatchString(m.Name) {
		return m.Name
	}
	t := strings.NewReplacer("-", " ", "_", " ").Replace(m.Name)
	return strings.ToUpper(t[:1]) + t[1:]
}

// ID is the note id the memory is imported as.
func (m Memory) ID() string { return note.Slugify(m.Name) }
