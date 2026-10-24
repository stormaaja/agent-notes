package store

import (
	"os"
	"path/filepath"
)

// DirName is the notes directory name looked up in a repository.
const DirName = ".notes"

// ResolveDir picks the notes directory, first match wins: flagDir, envDir
// (both resolved against cwd when relative), the nearest existing .notes
// directory at or above cwd, .notes at the nearest git root (a directory
// containing a .git file or directory), and finally cwd/.notes.
func ResolveDir(cwd, flagDir, envDir string) (string, error) {
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	for _, d := range []string{flagDir, envDir} {
		if d == "" {
			continue
		}
		if !filepath.IsAbs(d) {
			d = filepath.Join(cwd, d)
		}
		return filepath.Clean(d), nil
	}
	if dir, ok := walkUp(cwd, func(d string) bool {
		fi, err := os.Stat(filepath.Join(d, DirName))
		return err == nil && fi.IsDir()
	}); ok {
		return filepath.Join(dir, DirName), nil
	}
	if dir, ok := walkUp(cwd, func(d string) bool {
		_, err := os.Stat(filepath.Join(d, ".git"))
		return err == nil
	}); ok {
		return filepath.Join(dir, DirName), nil
	}
	return filepath.Join(cwd, DirName), nil
}

// walkUp returns the first directory from dir up to the filesystem root that satisfies match.
func walkUp(dir string, match func(string) bool) (string, bool) {
	for {
		if match(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
