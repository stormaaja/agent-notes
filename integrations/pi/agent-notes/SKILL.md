---
name: agent-notes
description: Shared, git-committed project knowledge in a repository's .notes/ folder, managed with the agent-notes CLI or its MCP tools. Use when working in a repository that has a .notes/ directory or mentions agent-notes, when the user asks to remember, record, note down or document something for later, and when looking up project conventions, decisions, gotchas or how-tos.
---

# agent-notes

`agent-notes` keeps knowledge notes as Markdown files in the repository's `.notes/` folder.
They are committed to git and shared with teammates and other agents, so use them for
knowledge about the repository instead of private memory.

If this skill was invoked as `/skill:agent-notes <arguments>`, treat the arguments as search
keywords: search, then summarize the matching notes for the user.

## Tools

Run the CLI with the bash tool and add `--json` to `search`, `list` and `show`. If the
`agent-notes` MCP server is configured in pi (its `notes_search`, `notes_get`, `notes_add`,
`notes_update` and `notes_delete` tools are available), prefer those tools over the CLI.
Don't edit files in `.notes/` directly.

## Workflow

1. **Search before working.** Every keyword must match, so start with one or two specific
   words, then broaden. `agent-notes list --json` shows everything.
2. **Read the relevant notes** with `agent-notes show <id> --json` and follow them.
3. **Do the task.**
4. **Record durable knowledge**: conventions, gotchas, decisions and how-tos that are not
   obvious from the code, git history or README, and anything the user asks you to remember
   about the project. Search first and update an existing note rather than adding a
   duplicate (adding an existing title creates a second note with a `-2` id).
5. **Tell the user** which notes you added or updated (ids and a one-line summary).

## Writing notes

- A clear title, a one-line description and a few lowercase tags.
- A short, specific body that gives the reason as well as the rule.
- Personal preferences that are not about the repository don't belong in `.notes/`.

## Safety

pi runs tools without asking for permission, so:

- Never store secrets, credentials, tokens, private URLs or personal data in a note.
- Never delete a note unless the user asked for it. Prefer updating outdated notes.
- Don't commit or push on your own. When the user commits the related work, include the
  `.notes/` changes in it; otherwise remind them that `.notes/` has changes to commit.

## Commands

```sh
agent-notes search [--tag t]... [keywords...] --json   # best match first, with a score
agent-notes list [--tag t]... --json                   # all notes, newest first
agent-notes show <id> --json                           # full note including body
agent-notes add --title "Title" --description "One line" --tags a,b --body "Markdown"
agent-notes add --title "Title" --tags a,b --body - <<'EOF'
Body read from stdin.
EOF
agent-notes update <id> [--title T] [--description D] [--tags a,b] [--add-tag t] \
    [--remove-tag t] [--body TEXT | --body - | --body-file PATH]
agent-notes delete <id>                                 # only when the user asks
agent-notes init                                        # create .notes/ if missing
```

`list` and `search` leave bodies out of `--json` output unless `--full` is given.
`update` changes only the given fields; `--tags` replaces all tags. Notes live in the
nearest `.notes/` up to the git root; `--dir PATH` overrides it. Exit codes: 0 ok,
1 error, 2 usage error, 3 note not found.
