package cli

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// listFlag collects comma-separated values from repeated flags.
type listFlag []string

func (l *listFlag) String() string { return strings.Join(*l, ",") }

func (l *listFlag) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			*l = append(*l, s)
		}
	}
	return nil
}

func setFlags(fs *flag.FlagSet) map[string]bool {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

type bodyFlags struct{ text, file string }

func (b *bodyFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&b.text, "body", "", "")
	fs.StringVar(&b.file, "body-file", "", "")
}

// readBody returns the body selected by --body (with "-" reading stdin) or
// --body-file, and whether either flag was given.
func (r *runner) readBody(fs *flag.FlagSet, b bodyFlags) (string, bool, error) {
	set := setFlags(fs)
	switch {
	case set["body"] && set["body-file"]:
		return "", false, usagef(fs.Name(), "--body and --body-file are mutually exclusive")
	case set["body"] && b.text == "-":
		data, err := io.ReadAll(r.env.Stdin)
		return string(data), true, err
	case set["body"]:
		return b.text, true, nil
	case set["body-file"]:
		path := b.file
		if !filepath.IsAbs(path) {
			path = filepath.Join(r.env.Cwd, path)
		}
		data, err := os.ReadFile(path)
		return string(data), true, err
	}
	return "", false, nil
}
