package store

import (
	"os"
	"path/filepath"
)

// DirName is the notes directory name looked up in a repository.
const DirName = ".notes"

// ResolveDir picks the notes directory, first match wins: flagDir, envDir
// (both resolved against cwd when relative), the nearest existing .notes
// directory from cwd up to the git root (a directory containing a .git file
// or directory) or, outside a repository, up to the filesystem root,
// .notes at the git root, and finally cwd/.notes.
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
	gitRoot, inRepo := walkUp(cwd, "", func(d string) bool {
		_, err := os.Stat(filepath.Join(d, ".git"))
		return err == nil
	})
	if dir, ok := walkUp(cwd, gitRoot, func(d string) bool {
		fi, err := os.Stat(filepath.Join(d, DirName))
		return err == nil && fi.IsDir()
	}); ok {
		return filepath.Join(dir, DirName), nil
	}
	if inRepo {
		return filepath.Join(gitRoot, DirName), nil
	}
	return filepath.Join(cwd, DirName), nil
}

// walkUp returns the first directory from dir upwards that satisfies match,
// checking up to and including stop, or up to the filesystem root when stop is empty.
func walkUp(dir, stop string, match func(string) bool) (string, bool) {
	for {
		if match(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if dir == stop || parent == dir {
			return "", false
		}
		dir = parent
	}
}
