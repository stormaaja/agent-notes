// Package note defines the note model and its Markdown-with-frontmatter file format.
package note

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const delimiter = "---"

// Note is a single knowledge note. Body holds the Markdown after the frontmatter.
type Note struct {
	ID          string    `yaml:"id"`
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags,flow"`
	Created     time.Time `yaml:"created"`
	Updated     time.Time `yaml:"updated"`
	Body        string    `yaml:"-"`
}

var (
	ErrNoFrontmatter = errors.New("note: missing frontmatter")
	ErrUnterminated  = errors.New("note: unterminated frontmatter")
	ErrMissingID     = errors.New("note: missing id")
)

// Parse decodes a note file. CRLF line endings are accepted. Leading and
// trailing blank lines around the body are dropped. Timestamps are returned in UTC.
func Parse(data []byte) (Note, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	rest, ok := strings.CutPrefix(text, delimiter+"\n")
	if !ok {
		return Note{}, ErrNoFrontmatter
	}
	front, body, ok := cutAtDelimiterLine(rest)
	if !ok {
		return Note{}, ErrUnterminated
	}
	var n Note
	if err := yaml.Unmarshal([]byte(front), &n); err != nil {
		return Note{}, fmt.Errorf("note: invalid frontmatter: %w", err)
	}
	if n.ID == "" {
		return Note{}, ErrMissingID
	}
	if n.Tags == nil {
		n.Tags = []string{}
	}
	n.Created = n.Created.UTC()
	n.Updated = n.Updated.UTC()
	n.Body = strings.Trim(body, "\n")
	return n, nil
}

// cutAtDelimiterLine splits s around the first line that is exactly the delimiter.
func cutAtDelimiterLine(s string) (before, after string, found bool) {
	for i := 0; i <= len(s); {
		line, _, _ := strings.Cut(s[i:], "\n")
		if line == delimiter {
			return s[:i], s[i+len(line):], true
		}
		i += len(line) + 1
	}
	return "", "", false
}

// Marshal encodes the note as frontmatter, a blank line and the body.
// Timestamps are written as RFC3339 in UTC with second precision; tags use flow style.
func (n Note) Marshal() ([]byte, error) {
	n.Created = n.Created.UTC().Truncate(time.Second)
	n.Updated = n.Updated.UTC().Truncate(time.Second)
	if n.Tags == nil {
		n.Tags = []string{}
	}
	var buf bytes.Buffer
	buf.WriteString(delimiter + "\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return nil, fmt.Errorf("note: encode frontmatter: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString(delimiter + "\n")
	if body := strings.Trim(n.Body, "\n"); body != "" {
		buf.WriteString("\n" + body + "\n")
	}
	return buf.Bytes(), nil
}
