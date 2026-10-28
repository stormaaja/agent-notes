package cli

import (
	"io"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stormaaja/agent-notes/internal/mcpserver"
	"github.com/stormaaja/agent-notes/internal/version"
)

var mcpCmd = command{
	name:    "mcp",
	summary: "run an MCP server on stdin and stdout",
	help: `Serves the notes over the Model Context Protocol until stdin is closed.
The notes directory is resolved once at startup. Logs go to stderr.
Tools: notes_search, notes_get, notes_add, notes_update, notes_delete.

Register it with Claude Code, for example:

  claude mcp add agent-notes -- agent-notes mcp
`,
	run: runMCP,
}

func runMCP(r *runner, args []string) error {
	if _, err := r.parseN(r.flagSet("mcp"), args); err != nil {
		return err
	}
	s, err := r.store()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(r.env.Stderr, nil))
	server := mcpserver.New(s, version.String(), logger)
	err = server.Run(r.env.Context, &mcp.IOTransport{
		Reader: io.NopCloser(r.env.Stdin),
		Writer: nopWriteCloser{r.env.Stdout},
	})
	if r.env.Context.Err() != nil {
		return nil
	}
	return err
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
