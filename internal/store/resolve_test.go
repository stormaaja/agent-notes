package store

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdirs(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveDir(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	deep := filepath.Join(repo, "a", "b")
	mkdirs(t, deep, filepath.Join(repo, ".git"))

	worktree := filepath.Join(root, "worktree")
	wtDeep := filepath.Join(worktree, "sub")
	mkdirs(t, wtDeep)
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	withNotes := filepath.Join(root, "withnotes")
	withNotesDeep := filepath.Join(withNotes, "x", "y")
	mkdirs(t, withNotesDeep, filepath.Join(withNotes, ".notes"), filepath.Join(withNotes, "x", ".notes"), filepath.Join(withNotes, ".git"))

	notesFile := filepath.Join(root, "notesfile")
	mkdirs(t, filepath.Join(notesFile, ".git"))
	if err := os.WriteFile(filepath.Join(notesFile, ".notes"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	home := filepath.Join(root, "home")
	homeRepo := filepath.Join(home, "repo")
	homeRepoDeep := filepath.Join(homeRepo, "pkg")
	homeNoGit := filepath.Join(home, "scratch")
	mkdirs(t, filepath.Join(home, ".notes"), filepath.Join(homeRepo, ".git"), homeRepoDeep, homeNoGit)

	plain := filepath.Join(root, "plain")
	mkdirs(t, plain)

	tests := []struct {
		name, cwd, flag, env, want string
	}{
		{"flag absolute", deep, "/abs/flag", "/abs/env", "/abs/flag"},
		{"flag relative to cwd", deep, "../notes", "/abs/env", filepath.Join(repo, "a", "notes")},
		{"env absolute", deep, "", "/abs/env", "/abs/env"},
		{"env relative to cwd", deep, "", "envnotes", filepath.Join(deep, "envnotes")},
		{"nested .notes below git root wins", withNotesDeep, "", "", filepath.Join(withNotes, "x", ".notes")},
		{"existing .notes in cwd", withNotes, "", "", filepath.Join(withNotes, ".notes")},
		{".notes file is not a notes dir", notesFile, "", "", filepath.Join(notesFile, ".notes")},
		{".notes above git root is ignored", homeRepoDeep, "", "", filepath.Join(homeRepo, ".notes")},
		{".notes above cwd used outside git", homeNoGit, "", "", filepath.Join(home, ".notes")},
		{"git dir root", deep, "", "", filepath.Join(repo, ".notes")},
		{"git file root (worktree)", wtDeep, "", "", filepath.Join(worktree, ".notes")},
		{"fallback to cwd", plain, "", "", filepath.Join(plain, ".notes")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveDir(tt.cwd, tt.flag, tt.env)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("ResolveDir = %q, want %q", got, tt.want)
			}
		})
	}
}
