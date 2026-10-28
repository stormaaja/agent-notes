package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stormaaja/agent-notes/internal/version"
)

func TestVersion(t *testing.T) {
	e := newEnv(t)
	want := "agent-notes " + version.String() + "\n"
	for _, args := range [][]string{{"version"}, {"--version"}, {"-version"}} {
		out, errOut, code := e.run(args...)
		assertCode(t, args, code, 0, errOut)
		if out != want {
			t.Errorf("%q: stdout = %q, want %q", args, out, want)
		}
	}
	for _, args := range [][]string{{"--json", "version"}, {"version", "--json"}, {"--json", "--version"}, {"--version", "--json"}} {
		out, errOut, code := e.run(args...)
		assertCode(t, args, code, 0, errOut)
		var got map[string]string
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("%q: stdout is not JSON: %q", args, out)
		}
		if len(got) != 1 || got["version"] != version.String() {
			t.Errorf("%q: JSON = %v, want {version: %q}", args, got, version.String())
		}
	}
}

func TestVersionHelpAndErrors(t *testing.T) {
	e := newEnv(t)
	if out := e.mustRun("help"); !strings.Contains(out, "version") || !strings.Contains(out, "--version") {
		t.Errorf("help does not list version:\n%s", out)
	}
	if out := e.mustRun("help", "version"); !strings.Contains(out, "agent-notes version") {
		t.Errorf("version help = %q", out)
	}
	args := []string{"version", "extra"}
	_, errOut, code := e.run(args...)
	assertCode(t, args, code, 2, errOut)
}
