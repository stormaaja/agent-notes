package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)

// testEnv is a sandboxed environment: cwd is a temp git repository root,
// the clock advances one minute per call and stdin is empty unless set.
type testEnv struct {
	t     *testing.T
	cwd   string
	vars  map[string]string
	stdin string
	calls int
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	cwd := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwd, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &testEnv{t: t, cwd: cwd, vars: map[string]string{}}
}

// notesDir is where notes land without --dir or AGENT_NOTES_DIR.
func (e *testEnv) notesDir() string { return filepath.Join(e.cwd, ".notes") }

func (e *testEnv) run(args ...string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	code = Run(args, Env{
		Stdin:  strings.NewReader(e.stdin),
		Stdout: &out,
		Stderr: &errOut,
		Cwd:    e.cwd,
		Getenv: func(k string) string { return e.vars[k] },
		Now: func() time.Time {
			tm := t0.Add(time.Duration(e.calls) * time.Minute)
			e.calls++
			return tm
		},
	})
	return out.String(), errOut.String(), code
}

// mustRun runs args and fails the test unless the exit code is 0.
func (e *testEnv) mustRun(args ...string) string {
	e.t.Helper()
	out, errOut, code := e.run(args...)
	if code != 0 {
		e.t.Fatalf("%q: exit %d, stderr %q", args, code, errOut)
	}
	return out
}

func assertCode(t *testing.T, args []string, code, want int, stderr string) {
	t.Helper()
	if code != want {
		t.Errorf("%q: exit %d, want %d (stderr %q)", args, code, want, stderr)
	}
}

func TestHelp(t *testing.T) {
	e := newEnv(t)
	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		out, errOut, code := e.run(args...)
		assertCode(t, args, code, 0, errOut)
		for _, want := range []string{"Usage:", "init", "add", "show", "list", "search", "help", "--dir", "--json"} {
			if !strings.Contains(out, want) {
				t.Errorf("%q: stdout missing %q:\n%s", args, want, out)
			}
		}
	}
}

func TestCommandHelp(t *testing.T) {
	e := newEnv(t)
	for _, args := range [][]string{{"help", "init"}, {"init", "-h"}, {"init", "--help"}, {"--json", "init", "--help"}} {
		out, errOut, code := e.run(args...)
		assertCode(t, args, code, 0, errOut)
		if !strings.Contains(out, "agent-notes init") {
			t.Errorf("%q: stdout = %q, want init usage", args, out)
		}
	}
	if _, err := os.Stat(e.notesDir()); !os.IsNotExist(err) {
		t.Errorf("help created notes dir: %v", err)
	}
}

func TestUsageErrors(t *testing.T) {
	e := newEnv(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "Usage:"},
		{[]string{"bogus"}, `unknown command "bogus"`},
		{[]string{"help", "bogus"}, `unknown command "bogus"`},
		{[]string{"--bogus", "init"}, "-bogus"},
		{[]string{"init", "--bogus"}, "-bogus"},
		{[]string{"init", "extra"}, "unexpected argument"},
		{[]string{"--dir"}, "-dir"},
	} {
		out, errOut, code := e.run(tc.args...)
		assertCode(t, tc.args, code, 2, errOut)
		if !strings.Contains(errOut, tc.want) {
			t.Errorf("%q: stderr = %q, want it to contain %q", tc.args, errOut, tc.want)
		}
		if out != "" {
			t.Errorf("%q: stdout = %q, want empty", tc.args, out)
		}
	}
}

func TestInit(t *testing.T) {
	e := newEnv(t)
	for range 2 {
		out := e.mustRun("init")
		if out != e.notesDir()+"\n" {
			t.Errorf("init stdout = %q, want %q", out, e.notesDir())
		}
		if fi, err := os.Stat(e.notesDir()); err != nil || !fi.IsDir() {
			t.Fatalf("notes dir not created: %v", err)
		}
	}
}

func TestDirResolution(t *testing.T) {
	e := newEnv(t)
	e.vars["AGENT_NOTES_DIR"] = "from-env"
	envDir := filepath.Join(e.cwd, "from-env")
	flagDir := filepath.Join(e.cwd, "from-flag")

	if out := e.mustRun("init"); out != envDir+"\n" {
		t.Errorf("env: init stdout = %q, want %q", out, envDir)
	}
	for _, args := range [][]string{{"--dir", "from-flag", "init"}, {"init", "--dir", "from-flag"}, {"init", "--dir=" + flagDir}} {
		if out := e.mustRun(args...); out != flagDir+"\n" {
			t.Errorf("%q: stdout = %q, want %q", args, out, flagDir)
		}
	}
	if _, err := os.Stat(e.notesDir()); !os.IsNotExist(err) {
		t.Errorf("default notes dir created despite overrides: %v", err)
	}
}

func TestInitJSON(t *testing.T) {
	e := newEnv(t)
	var got struct{ Dir string }
	decodeJSON(t, e.mustRun("init", "--json"), &got)
	if got.Dir != e.notesDir() {
		t.Errorf("dir = %q, want %q", got.Dir, e.notesDir())
	}
}

func decodeJSON(t *testing.T, s string, v any) {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
}
