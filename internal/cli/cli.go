// Package cli implements the agent-notes command line interface.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/stormaaja/agent-notes/internal/store"
)

// Exit codes returned by Run.
const (
	exitOK       = 0
	exitError    = 1
	exitUsage    = 2
	exitNotFound = 3
)

const envDir = "AGENT_NOTES_DIR"

// Env is the process environment Run operates in.
type Env struct {
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	Cwd            string
	Getenv         func(string) string
	Now            func() time.Time
}

type command struct {
	name    string
	usage   string // arguments after the command name
	summary string
	help    string // flag and argument details
	run     func(r *runner, args []string) error
}

var commands = []command{
	initCmd,
}

func lookup(name string) (command, bool) {
	for _, c := range commands {
		if c.name == name {
			return c, true
		}
	}
	return command{}, false
}

const globalHelp = `Global flags (accepted before or after the command):
  --dir PATH   notes directory (default: $AGENT_NOTES_DIR, the nearest .notes
               up to the git root, <git root>/.notes, or ./.notes)
  --json       print machine-readable JSON
`

func usage() string {
	var b strings.Builder
	b.WriteString("Usage: agent-notes [--dir PATH] [--json] <command> [flags] [args]\n\nCommands:\n")
	for _, c := range commands {
		fmt.Fprintf(&b, "  %-8s %s\n", c.name, c.summary)
	}
	fmt.Fprintf(&b, "  %-8s %s\n", "help", "show help for a command")
	b.WriteString("\n" + globalHelp)
	b.WriteString("\nRun 'agent-notes help <command>' for command details.\n")
	return b.String()
}

func (c command) usageLine() string {
	return strings.TrimSpace("agent-notes " + c.name + " " + c.usage)
}

func (c command) fullHelp() string {
	s := "Usage: " + c.usageLine() + "\n\n" + c.summary + ".\n"
	if c.help != "" {
		s += "\n" + c.help
	}
	return s + "\n" + globalHelp
}

// usageError is a command line mistake; Run reports it with exit code 2.
type usageError struct {
	cmd string // command whose usage line is printed, if any
	msg string
}

func (e *usageError) Error() string { return e.msg }

func usagef(cmd, format string, a ...any) error {
	return &usageError{cmd: cmd, msg: fmt.Sprintf(format, a...)}
}

// errHelp is returned when help was requested for a command.
type errHelp struct{ cmd string }

func (e *errHelp) Error() string { return "help requested" }

type runner struct {
	env  Env
	dir  string
	json bool
}

// Run executes the command line args (without the program name) and returns the exit code.
func Run(args []string, env Env) int {
	if env.Stdin == nil {
		env.Stdin = strings.NewReader("")
	}
	if env.Getenv == nil {
		env.Getenv = os.Getenv
	}
	if env.Now == nil {
		env.Now = time.Now
	}
	r := &runner{env: env}
	return r.exit(r.dispatch(args))
}

func (r *runner) dispatch(args []string) error {
	fs := r.flagSet("")
	if err := fs.Parse(args); err != nil {
		return r.flagError("", err)
	}
	args = fs.Args()
	if len(args) == 0 {
		return usagef("", "missing command")
	}
	name, args := args[0], args[1:]
	if name == "help" {
		return r.help(args)
	}
	c, ok := lookup(name)
	if !ok {
		return usagef("", "unknown command %q", name)
	}
	return c.run(r, args)
}

func (r *runner) help(args []string) error {
	switch len(args) {
	case 0:
		fmt.Fprint(r.env.Stdout, usage())
		return nil
	case 1:
		c, ok := lookup(args[0])
		if !ok {
			return usagef("", "unknown command %q", args[0])
		}
		fmt.Fprint(r.env.Stdout, c.fullHelp())
		return nil
	}
	return usagef("", "help takes at most one command")
}

func (r *runner) exit(err error) int {
	var uerr *usageError
	var herr *errHelp
	switch {
	case err == nil:
		return exitOK
	case errors.As(err, &herr):
		if c, ok := lookup(herr.cmd); ok {
			fmt.Fprint(r.env.Stdout, c.fullHelp())
		} else {
			fmt.Fprint(r.env.Stdout, usage())
		}
		return exitOK
	case errors.As(err, &uerr):
		fmt.Fprintf(r.env.Stderr, "error: %s\n", uerr.msg)
		if c, ok := lookup(uerr.cmd); ok {
			fmt.Fprintf(r.env.Stderr, "Usage: %s\nRun 'agent-notes help %s' for details.\n", c.usageLine(), c.name)
		} else {
			fmt.Fprint(r.env.Stderr, usage())
		}
		return exitUsage
	case errors.Is(err, store.ErrEmptyTitle):
		fmt.Fprintf(r.env.Stderr, "error: %v\n", err)
		return exitUsage
	case errors.Is(err, store.ErrNotFound):
		fmt.Fprintf(r.env.Stderr, "error: %v\n", err)
		return exitNotFound
	default:
		fmt.Fprintf(r.env.Stderr, "error: %v\n", err)
		return exitError
	}
}

// flagSet returns a flag set for cmd with the global flags registered.
// Values set earlier on the command line are kept as defaults.
func (r *runner) flagSet(cmd string) *flag.FlagSet {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	fs.StringVar(&r.dir, "dir", r.dir, "")
	fs.BoolVar(&r.json, "json", r.json, "")
	return fs
}

func (r *runner) flagError(cmd string, err error) error {
	if errors.Is(err, flag.ErrHelp) {
		return &errHelp{cmd: cmd}
	}
	return usagef(cmd, "%v", err)
}

// parse parses flags interspersed with positional arguments, which it
// returns. Everything after a "--" argument is positional.
func (r *runner) parse(fs *flag.FlagSet, args []string) ([]string, error) {
	pos := []string{}
	for {
		if err := fs.Parse(args); err != nil {
			return nil, r.flagError(fs.Name(), err)
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		if n := len(args) - len(rest); n > 0 && args[n-1] == "--" {
			return append(pos, rest...), nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

// parseN is parse requiring exactly the named positional arguments.
func (r *runner) parseN(fs *flag.FlagSet, args []string, names ...string) ([]string, error) {
	pos, err := r.parse(fs, args)
	if err != nil {
		return nil, err
	}
	if len(pos) < len(names) {
		return nil, usagef(fs.Name(), "missing %s", names[len(pos)])
	}
	if len(pos) > len(names) {
		return nil, usagef(fs.Name(), "unexpected argument %q", pos[len(names)])
	}
	return pos, nil
}

func (r *runner) notesDir() (string, error) {
	return store.ResolveDir(r.env.Cwd, r.dir, r.env.Getenv(envDir))
}

func (r *runner) store() (*store.Store, error) {
	dir, err := r.notesDir()
	if err != nil {
		return nil, err
	}
	return store.New(dir, r.env.Now), nil
}

func (r *runner) printJSON(v any) error {
	enc := json.NewEncoder(r.env.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
