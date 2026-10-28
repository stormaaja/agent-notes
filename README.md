# agent-notes

Shared knowledge notes for people and coding agents, stored as plain Markdown in your
repository.

Coding agents such as Claude Code learn things while they work: conventions, gotchas,
decisions, how to run things. Usually that knowledge goes into memory on one laptop, so your
teammates, your other machines and other agents never see it. `agent-notes` keeps it in a
`.notes/` folder that you commit with the code. Everyone who clones the repository, and every
agent working in it, can search and extend the same notes.

- Notes are readable Markdown files with YAML frontmatter, so they are easy to review in diffs.
- One small CLI with human-readable output and `--json` for agents.
- A built-in MCP server, so agents can use notes as native tools.
- Ready-made agent instructions, a Claude Code skill, and an importer for existing Claude Code
  memories.

## Install

```sh
go install github.com/stormaaja/agent-notes/cmd/agent-notes@latest
```

Use `@vX.Y.Z` instead of `@latest` for a specific release.

This needs Go 1.26 or newer (the `go` command fetches a newer toolchain if needed) and puts `agent-notes` in `$(go env GOPATH)/bin`.

## Quick start

```sh
cd your-repo
agent-notes init
agent-notes add --title "Go error wrapping" \
  --description "Always wrap errors with %w and context" \
  --tags go,errors \
  --body 'Use fmt.Errorf("read config: %w", err) so callers can errors.Is.'
agent-notes search errors
git add .notes && git commit -m "Add note on error wrapping"
```

## Note format

Each note is a file `.notes/<id>.md`:

```markdown
---
id: go-error-wrapping
title: Go error wrapping
description: Always wrap errors with %w and context
tags: [errors, go]
created: 2026-10-24T07:05:47Z
updated: 2026-10-24T07:05:47Z
---

Use fmt.Errorf("read config: %w", err) so callers can errors.Is.
```

- The **id** is the slug of the title (lowercase ASCII letters, digits and dashes, at most 60
  characters). If it is taken, `-2`, `-3`, … is appended. The id never changes, even when you
  change the title later. The file name is authoritative.
- **Tags** are lowercased, trimmed, deduplicated and sorted.
- **Timestamps** are RFC 3339 in UTC. `updated` changes on every update.
- The **body** is free Markdown.

You can edit note files by hand. Files that fail to parse are skipped with a warning.

## Where notes live

The notes directory is resolved as follows. The first match wins:

1. `--dir PATH`
2. the `AGENT_NOTES_DIR` environment variable
3. the nearest existing `.notes/` directory, looking from the current directory upwards and
   stopping at the git root (the directory containing `.git`), or at the filesystem root
   outside a repository
4. `.notes/` at the git root
5. `.notes/` in the current directory

Relative paths in `--dir` and `AGENT_NOTES_DIR` are resolved against the current directory.
The directory is created on the first write, or explicitly with `agent-notes init`. In
practice, any command run anywhere inside a repository uses `<repo>/.notes`.

## Commands

```
agent-notes [--dir PATH] [--json] [--version] <command> [flags] [args]
```

`--dir` and `--json` work before or after the command. Run `agent-notes help <command>` for
details.

| Command | What it does |
| --- | --- |
| `init` | Create the notes directory and print its path |
| `add --title T [--description D] [--tags a,b] [--body TEXT \| --body - \| --body-file PATH]` | Create a note |
| `show <id>` | Print a note |
| `list [--tag t]... [--full]` | List notes, most recently updated first |
| `search [--tag t]... [--full] [keywords...]` | Find notes, best match first |
| `update <id> [--title] [--description] [--tags] [--add-tag] [--remove-tag] [--body \| --body-file]` | Change only the given fields |
| `delete <id>` | Delete a note |
| `import-memory <dir> [--tag t]... [--dry-run]` | Import Claude Code memory files |
| `instructions` | Print agent instructions for `CLAUDE.md` or `AGENTS.md` |
| `mcp` | Run an MCP server on stdin and stdout (see [MCP server](#mcp-server)) |
| `version` | Print the version (also `agent-notes --version`; see [Releasing](#releasing)) |

Exit codes: `0` success, `1` error, `2` usage error, `3` note not found.

### Examples

```console
$ agent-notes add --title "Redis cache TTL" --description "Keep cache TTLs under an hour" \
    --tags redis,cache --body - <<'EOF'
Long TTLs hide stale data after deploys.
EOF
created redis-cache-ttl (/path/to/repo/.notes/redis-cache-ttl.md)

$ agent-notes list
go-error-wrapping  Go error wrapping  [errors, go]
    Always wrap errors with %w and context
redis-cache-ttl  Redis cache TTL  [cache, redis]
    Keep cache TTLs under an hour

$ agent-notes search ttl
redis-cache-ttl  Redis cache TTL  [cache, redis]
    Keep cache TTLs under an hour

$ agent-notes show redis-cache-ttl
---
id: redis-cache-ttl
title: Redis cache TTL
description: Keep cache TTLs under an hour
tags: [cache, redis]
created: 2026-10-24T07:05:47Z
updated: 2026-10-24T07:05:47Z
path: /path/to/repo/.notes/redis-cache-ttl.md
---

Long TTLs hide stale data after deploys.

$ agent-notes update redis-cache-ttl --add-tag ops --description "Keep cache TTLs under 30 minutes"
updated redis-cache-ttl (/path/to/repo/.notes/redis-cache-ttl.md)

$ agent-notes update redis-cache-ttl --title "Valkey cache TTL"   # the id stays redis-cache-ttl
updated redis-cache-ttl (/path/to/repo/.notes/redis-cache-ttl.md)

$ agent-notes delete redis-cache-ttl
deleted redis-cache-ttl
```

Body input for `add` and `update`: `--body TEXT`, `--body -` to read stdin, or
`--body-file PATH`. With `update`, `--description ""` clears the description and `--tags ""`
clears all tags. `--tag`, `--tags`, `--add-tag` and `--remove-tag` accept comma-separated
values and can be repeated.

### Search

- `--tag` filters by whole tag, case-insensitively. Several tags must all be present.
- Every keyword must appear, case-insensitively, as a substring of the title, description,
  a tag or the body.
- Results are ranked by score. Each keyword adds 5 for a title match, 4 for a tag, 3 for the
  description and 1 for the body. Ties go to the most recently updated note.
- `search` with neither keywords nor tags lists all notes, newest first, like `list`.

### JSON output for agents

With `--json`, every command except `instructions`, `mcp` and `help` prints JSON:

- `show`, `add` and `update` print the full note, including `body` and `path`.
- `list` and `search` print an array of notes without `body` (add `--full` to include it).
  `search` adds a `score`.
- `delete` prints `{"id": "...", "deleted": true}`. `init` prints `{"dir": "..."}`.
- `import-memory` prints one result per file (see below).

```console
$ agent-notes search --tag cache redis --json
[
  {
    "id": "redis-cache-ttl",
    "title": "Redis cache TTL",
    "description": "Keep cache TTLs under an hour",
    "tags": [
      "cache",
      "redis"
    ],
    "created": "2026-10-24T07:05:47Z",
    "updated": "2026-10-24T07:05:47Z",
    "path": "/path/to/repo/.notes/redis-cache-ttl.md",
    "score": 9
  }
]
```

Errors go to stderr as `error: ...` with the exit codes listed above.

## Using it with agents

The point of `agent-notes` is that agents read notes before they work and write durable
knowledge back into them. Agents need to be told to do this.

### Claude Code

1. **Add instructions to `CLAUDE.md`.** `agent-notes instructions` prints a short Markdown
   section that tells the agent to search notes before starting a task, record conventions,
   gotchas and decisions as notes instead of private memory, avoid duplicates and never store
   secrets. It also includes a command cheat sheet.

   ```sh
   agent-notes instructions >> CLAUDE.md
   ```

2. **Install the skill (optional).** [`integrations/claude-code/SKILL.md`](integrations/claude-code/SKILL.md)
   is a Claude Code skill with the same workflow. It triggers when you work in a repository
   with `.notes/`, when you ask Claude to remember or note something, and when you look up
   project conventions. Install it for yourself or for one repository:

   ```sh
   # for all your projects
   mkdir -p ~/.claude/skills/agent-notes
   cp integrations/claude-code/SKILL.md ~/.claude/skills/agent-notes/SKILL.md

   # or for one repository, committed so teammates get it too
   mkdir -p .claude/skills/agent-notes
   cp integrations/claude-code/SKILL.md .claude/skills/agent-notes/SKILL.md
   ```

3. **Migrate existing memories.** Claude Code keeps per-project memories in
   `~/.claude/projects/<project>/memory/`, where `<project>` is the project path with `/`
   replaced by `-`. Import them into the repository's notes, review the result, then commit:

   ```console
   $ agent-notes import-memory ~/.claude/projects/-home-me-src-myapp/memory --dry-run
   created no-mocks-in-integration-tests from no-mocks.md
   created use-pnpm from project_pnpm.md
   created user-role from user_role.md
   3 created, 0 updated, 0 unchanged, 0 skipped (dry run, nothing written)

   $ agent-notes import-memory ~/.claude/projects/-home-me-src-myapp/memory
   ...
   $ git diff --stat; git add .notes
   ```

   How memories are imported:

   - Every `*.md` file is imported except `MEMORY.md`, the index file.
   - Both frontmatter layouts are supported: `name`, `description` and `type` at the top
     level, or `type` nested under `metadata`. Other fields are ignored. A file without
     frontmatter is imported with its file name as the name and its whole content as body.
   - The note title is the memory `name`. Lowercase slugs such as `use-pnpm` or `user_role`
     become `Use pnpm` and `User role`. The id is the slug of the name.
   - Tags are `imported-memory` plus the memory type (`user`, `feedback`, `project`,
     `reference`), plus any `--tag` values.
   - Importing again is safe. Notes whose memory changed are `updated`, the others are
     `unchanged`. Tags you added to an imported note are kept. A note with the same id that
     was not imported (it lacks the `imported-memory` tag) is never overwritten. That file is
     `skipped`, as are files with broken frontmatter and files whose name maps to an id
     already imported in the same run.

   Memories often hold personal preferences (`user` type) that do not belong in a shared
   repository. Use `--dry-run`, review the notes, and delete what should stay private before
   committing.

### Other agents

Any agent that can run shell commands can use `agent-notes`. Put the instructions where the
agent reads them, for example:

```sh
agent-notes instructions >> AGENTS.md   # Codex, Cursor and other agents that read AGENTS.md
```

For agents with their own rules files, paste the same output there.

Tell the agent to use `--json` for `search`, `list` and `show`, and to check exit codes
(`3` means the note does not exist).

## MCP server

`agent-notes mcp` serves the notes over the [Model Context Protocol](https://modelcontextprotocol.io)
on stdin and stdout. Agents then search and edit notes through tools instead of shell commands.
The notes directory is resolved once at startup, the same way as for every other command, so
start the server from inside the repository or pass `--dir`. Logs go to stderr.

| Tool | Arguments | Returns |
| --- | --- | --- |
| `notes_search` | `query?`, `tags?`, `limit?` (default 20), `full?` | `{"results": [...], "total": n}`, best match first, each with a `score`; bodies only with `full` |
| `notes_get` | `id` | the full note, including `body` and `path` |
| `notes_add` | `title`, `description?`, `tags?`, `body?` | the created note |
| `notes_update` | `id`, `title?`, `description?`, `tags?` (replaces), `add_tags?`, `remove_tags?`, `body?` | the updated note; omitted fields are left unchanged |
| `notes_delete` | `id` | `{"id": "...", "deleted": true}` |

Notes have the same JSON shape as `--json` output, and search follows the same rules as
`agent-notes search`: every keyword and every tag must match. Problems such as an unknown id,
an empty title or an update without any field come back as tool errors the agent can read and
correct. The server also sends short instructions telling the agent to search before working,
prefer updating over duplicating, and never store secrets.

### Claude Code

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

### Other MCP clients

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

## Development

```sh
go test ./... -race
go vet ./...
scripts/release_test.sh   # tests the release script in a throwaway clone
```

### Releasing

The current version is in the `VERSION` file (`X.Y.Z`); releases are tagged `vX.Y.Z`.

```sh
scripts/release.sh          # build bin/agent-notes at the current version
scripts/release.sh build    # patch release: 1.2.3 -> 1.2.4
scripts/release.sh minor    # 1.2.3 -> 1.3.0
scripts/release.sh major    # 1.2.3 -> 2.0.0
```

With a bump type, the script refuses to run if the working tree has any staged, unstaged or
untracked changes, if `VERSION` is malformed, or if the new tag already exists. It then runs
`go vet ./...` and `go test ./...`, writes `VERSION`, builds `bin/agent-notes`, commits
`Release vX.Y.Z` and creates the annotated tag `vX.Y.Z`. It never pushes; publish with:

```sh
git push --follow-tags
```

After that, `go install github.com/stormaaja/agent-notes/cmd/agent-notes@vX.Y.Z` installs the
release.

`agent-notes version` (or `--version`, `--json` for `{"version": "..."}`) prints the version,
which the MCP server also reports. It is resolved as:

1. the value set at build time with
   `-ldflags "-X github.com/stormaaja/agent-notes/internal/version.version=X.Y.Z"`
   (what `scripts/release.sh` does),
2. else the module version recorded by `go install …@vX.Y.Z`, without the `v`,
3. else `devel` (for example `go build` or `go run` from a checkout).

## License

See [LICENSE](LICENSE).
