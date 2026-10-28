# Commands

```
agent-notes [--dir PATH] [--json] [--version] <command> [flags] [args]
```

`--dir` and `--json` work before or after the command. Run `agent-notes help <command>` for
details. `--dir` overrides where notes live (see [Where notes live](note-format.md#where-notes-live)).

| Command | What it does |
| --- | --- |
| `init` | Create the notes directory and print its path |
| `add --title T [--description D] [--tags a,b] [--body TEXT \| --body - \| --body-file PATH]` | Create a note |
| `show <id>` | Print a note |
| `list [--tag t]... [--full]` | List notes, most recently updated first |
| `search [--tag t]... [--full] [keywords...]` | Find notes, best match first |
| `update <id> [--title] [--description] [--tags] [--add-tag] [--remove-tag] [--body \| --body-file]` | Change only the given fields |
| `delete <id>` | Delete a note |
| `import-memory <dir> [--tag t]... [--dry-run]` | Import Claude Code memory files (see [Migrate existing memories](agent-integration.md#migrate-existing-memories)) |
| `instructions` | Print agent instructions for `CLAUDE.md` or `AGENTS.md` (see [Using it with agents](agent-integration.md)) |
| `mcp` | Run an MCP server on stdin and stdout (see [MCP server](mcp.md)) |
| `version` | Print the version (also `agent-notes --version`; see [Releasing](releasing.md#version-command)) |

Exit codes: `0` success, `1` error, `2` usage error, `3` note not found.

## Examples

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

## Search

- `--tag` filters by whole tag, case-insensitively. Several tags must all be present.
- Every keyword must appear, case-insensitively, as a substring of the title, description,
  a tag or the body.
- Results are ranked by score. Each keyword adds 5 for a title match, 4 for a tag, 3 for the
  description and 1 for the body. Ties go to the most recently updated note.
- `search` with neither keywords nor tags lists all notes, newest first, like `list`.

## JSON output for agents

With `--json`, every command except `instructions`, `mcp` and `help` prints JSON:

- `show`, `add` and `update` print the full note, including `body` and `path`.
- `list` and `search` print an array of notes without `body` (add `--full` to include it).
  `search` adds a `score`.
- `delete` prints `{"id": "...", "deleted": true}`. `init` prints `{"dir": "..."}`.
- `import-memory` prints one result per file (see
  [Migrate existing memories](agent-integration.md#migrate-existing-memories)).

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
