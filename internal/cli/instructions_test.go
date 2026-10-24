package cli

import (
	"strings"
	"testing"
)

func TestInstructions(t *testing.T) {
	e := newEnv(t)
	out := e.mustRun("instructions")
	for _, want := range []string{
		".notes/",
		"agent-notes search <keywords> --json",
		"agent-notes search --tag",
		"agent-notes show <id>",
		"agent-notes list",
		"agent-notes add --title",
		"agent-notes update <id>",
		"agent-notes delete <id>",
		"secrets",
		"memory",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("instructions missing %q:\n%s", want, out)
		}
	}
	if !strings.HasPrefix(out, "## ") || !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
		t.Errorf("instructions should be a Markdown section ending in one newline:\n%q", out)
	}
	if n := strings.Count(out, "\n"); n > 40 {
		t.Errorf("instructions are %d lines, want them short", n)
	}
}

func TestInstructionsErrors(t *testing.T) {
	e := newEnv(t)
	args := []string{"instructions", "extra"}
	_, errOut, code := e.run(args...)
	assertCode(t, args, code, 2, errOut)
	if out := e.mustRun("help"); !strings.Contains(out, "instructions") {
		t.Errorf("help does not list instructions:\n%s", out)
	}
}
