// Package store keeps notes as Markdown files in a notes directory.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stormaaja/agent-notes/internal/note"
)

const ext = ".md"

var (
	ErrNotFound   = errors.New("note not found")
	ErrExists     = errors.New("note already exists")
	ErrInvalidID  = errors.New("invalid note id")
	ErrEmptyTitle = errors.New("note title is empty")
)

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Store reads and writes notes as <dir>/<id>.md. The file name is the
// authoritative id; a differing id in the frontmatter is ignored.
type Store struct {
	dir string
	now func() time.Time
}

// New returns a store for dir. A nil now uses time.Now.
func New(dir string, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{dir: dir, now: now}
}

func (s *Store) clock() time.Time { return s.now().UTC().Truncate(time.Second) }

func (s *Store) path(id string) string { return filepath.Join(s.dir, id+ext) }

func validateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return nil
}

// Create stores a new note with normalized tags and Created = Updated = now.
// Without an ID the id is the slug of the title, suffixed -2, -3, … until
// unused. A caller-provided ID is used as is and fails with ErrExists when taken.
func (s *Store) Create(n note.Note) (note.Note, error) {
	n.Title = strings.TrimSpace(n.Title)
	if n.Title == "" {
		return note.Note{}, ErrEmptyTitle
	}
	explicit := n.ID != ""
	if explicit {
		if err := validateID(n.ID); err != nil {
			return note.Note{}, err
		}
	}
	n.Tags = note.NormalizeTags(n.Tags)
	n.Created = s.clock()
	n.Updated = n.Created
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return note.Note{}, err
	}
	base := n.ID
	if !explicit {
		base = note.Slugify(n.Title)
	}
	for i := 1; ; i++ {
		n.ID = base
		if i > 1 {
			n.ID += "-" + strconv.Itoa(i)
		}
		err := s.write(n, false)
		if err == nil {
			return n, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return note.Note{}, err
		}
		if explicit {
			return note.Note{}, fmt.Errorf("%w: %q", ErrExists, n.ID)
		}
	}
}

// Get reads the note with the given id.
func (s *Store) Get(id string) (note.Note, error) {
	if err := validateID(id); err != nil {
		return note.Note{}, err
	}
	return s.read(s.path(id))
}

// List returns all notes, most recently updated first, then by id. A missing
// directory yields no notes. Files that cannot be read or parsed, or whose
// name is not a valid id, are skipped and reported as warnings. Hidden files
// and subdirectories are ignored.
func (s *Store) List() (notes []note.Note, warnings []error, err error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []note.Note{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	notes = []note.Note{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || filepath.Ext(name) != ext {
			continue
		}
		path := filepath.Join(s.dir, name)
		if err := validateID(strings.TrimSuffix(name, ext)); err != nil {
			warnings = append(warnings, fmt.Errorf("%s: %w", path, err))
			continue
		}
		n, err := s.read(path)
		if err != nil {
			warnings = append(warnings, err)
			continue
		}
		notes = append(notes, n)
	}
	slices.SortFunc(notes, func(a, b note.Note) int {
		if c := b.Updated.Compare(a.Updated); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return notes, warnings, nil
}

func (s *Store) read(path string) (note.Note, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		id := strings.TrimSuffix(filepath.Base(path), ext)
		return note.Note{}, fmt.Errorf("%w: %q", ErrNotFound, id)
	}
	if err != nil {
		return note.Note{}, err
	}
	n, err := note.Parse(data)
	if err != nil {
		return note.Note{}, fmt.Errorf("%s: %w", path, err)
	}
	n.ID = strings.TrimSuffix(filepath.Base(path), ext)
	return n, nil
}

// write atomically stores n via a temp file in the notes dir. With replace
// false it fails with fs.ErrExist instead of overwriting an existing note.
func (s *Store) write(n note.Note, replace bool) error {
	data, err := n.Marshal()
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".tmp-*"+ext)
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	_, err = f.Write(data)
	if err == nil {
		err = f.Chmod(0o644)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if replace {
		return os.Rename(tmp, s.path(n.ID))
	}
	return os.Link(tmp, s.path(n.ID))
}
