# Note format

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

See [Commands](commands.md) for creating, searching and editing notes.
