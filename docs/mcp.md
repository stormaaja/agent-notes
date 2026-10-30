# MCP server

`agent-notes mcp` serves the notes over the [Model Context Protocol](https://modelcontextprotocol.io)
on stdin and stdout. Agents then search and edit notes through tools instead of shell commands.
The notes directory is resolved once at startup, the same way as for every other command (see
[Where notes live](note-format.md#where-notes-live)), so start the server from inside the
repository or pass `--dir`. Logs go to stderr.

| Tool | Arguments | Returns |
| --- | --- | --- |
| `notes_search` | `query?`, `tags?`, `limit?` (default 20), `full?` | `{"results": [...], "total": n}`, best match first, each with a `score`; bodies only with `full` |
| `notes_get` | `id` | the full note, including `body` and `path` |
| `notes_add` | `title`, `description?`, `tags?`, `body?` | the created note |
| `notes_update` | `id`, `title?`, `description?`, `tags?` (replaces), `add_tags?`, `remove_tags?`, `body?` | the updated note; omitted fields are left unchanged |
| `notes_delete` | `id` | `{"id": "...", "deleted": true}` |

Notes have the same JSON shape as [`--json` output](commands.md#json-output-for-agents), and
search follows the same rules as [`agent-notes search`](commands.md#search): every keyword and
every tag must match. Problems such as an unknown id, an empty title or an update without any
field come back as tool errors the agent can read and correct. The server also sends short
instructions telling the agent to search before working, prefer updating over duplicating, and
never store secrets.

## Claude Code

Claude Code starts stdio servers in the directory you launched it from, normally the
repository, so that repository's notes are used:

```sh
claude mcp add agent-notes -- agent-notes mcp                   # just you, this project
claude mcp add --scope user agent-notes -- agent-notes mcp      # just you, all projects
claude mcp add --scope project agent-notes -- agent-notes mcp   # everyone, via .mcp.json
```

The project scope writes a `.mcp.json` file to commit with the repository:

```json
{
  "mcpServers": {
    "agent-notes": {
      "command": "agent-notes",
      "args": ["mcp"]
    }
  }
}
```

## pi

[pi](https://pi.dev) has built-in MCP support. Add the server to `~/.pi/agent/mcp.json` for
all your projects, or to `.pi/mcp.json` in the repository to share it with teammates (pi uses
a project file only after you trust the project):

```json
{
  "mcpServers": {
    "agent-notes": {
      "command": "agent-notes",
      "args": ["mcp"]
    }
  }
}
```

You can also add servers with `pi mcp add`; see pi's documentation for its arguments. Check
the connection with `pi mcp list` or `/mcp` inside pi. The
[pi skill](agent-integration.md#pi) uses the `notes_*` tools when they are available.

## Other MCP clients

Any client that launches stdio servers works. The usual configuration is a command, its
arguments and optionally environment variables. Use an absolute `--dir` (or
`AGENT_NOTES_DIR`) when the client does not start the server in the repository:

```json
{
  "mcpServers": {
    "agent-notes": {
      "command": "agent-notes",
      "args": ["mcp", "--dir", "/path/to/repo/.notes"]
    }
  }
}
```

If the client cannot find `agent-notes`, use the full path to the binary, for example
`$(go env GOPATH)/bin/agent-notes`.
