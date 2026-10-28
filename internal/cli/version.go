package cli

import (
	"fmt"

	"github.com/stormaaja/agent-notes/internal/version"
)

var versionCmd = command{
	name:    "version",
	summary: "print the agent-notes version",
	help: `Prints "agent-notes <version>", or {"version": "..."} with --json. The version
is the one set at build time (scripts/release.sh), else the module version
from 'go install …@vX.Y.Z', else "devel".
`,
	run: runVersion,
}

func runVersion(r *runner, args []string) error {
	if _, err := r.parseN(r.flagSet("version"), args); err != nil {
		return err
	}
	return r.printVersion()
}

func (r *runner) printVersion() error {
	v := version.String()
	if r.json {
		return r.printJSON(struct {
			Version string `json:"version"`
		}{v})
	}
	_, err := fmt.Fprintf(r.env.Stdout, "agent-notes %s\n", v)
	return err
}
