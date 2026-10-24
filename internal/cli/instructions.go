package cli

import (
	_ "embed"
	"fmt"
)

//go:embed instructions.md
var instructionsText string

var instructionsCmd = command{
	name:    "instructions",
	summary: "print agent instructions for CLAUDE.md or AGENTS.md",
	help: `Prints a Markdown section telling coding agents to search notes before
working and to record durable knowledge as notes. Append it to an agent
instructions file, for example:

  agent-notes instructions >> CLAUDE.md
`,
	run: runInstructions,
}

func runInstructions(r *runner, args []string) error {
	if _, err := r.parseN(r.flagSet("instructions"), args); err != nil {
		return err
	}
	_, err := fmt.Fprint(r.env.Stdout, instructionsText)
	return err
}
