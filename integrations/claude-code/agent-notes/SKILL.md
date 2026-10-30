---
name: agent-notes
description: Shared, git-committed project knowledge kept in a repository's .notes/ folder and managed with the agent-notes CLI. Use when starting work in a repository that has a .notes/ directory or mentions agent-notes, when the user asks to remember, record, note down or document something for later, and when looking up project conventions, gotchas, decisions or how-tos.
---

# agent-notes

`agent-notes` stores knowledge notes as Markdown files in the repository's `.notes/`
folder. The notes are committed to git, so they are shared with teammates, other
machines and other agents. For knowledge about a repository, use these notes
instead of private or local memory.

## Before starting work

1. Search for notes related to the task. Every keyword must match; results are
   ranked best first.

   ```sh
   agent-notes search <keywords> --json
   agent-notes search --tag <tag> --json
   ```

2. Read the notes that look relevant and follow them:

   ```sh
   agent-notes show <id>
   ```

If nothing matches, try fewer or broader keywords, or `agent-notes list --json`.

## Recording knowledge

When you learn something durable and non-obvious (a convention, gotcha, decision,
how-to), or the user asks you to remember something about the project:

1. Search first. If a note already covers the topic, update it rather than adding a
   duplicate.
2. Otherwise add a note with a clear title, a one-line description and a few
   lowercase tags.
3. Keep the body short and specific. Explain the reason as well as the rule.
4. Never store secrets, credentials, tokens or personal data.
5. Don't record what the code, git history or README already make obvious.
6. Remind the user to commit `.notes/` changes, or commit them with the related work.

Personal preferences that are not about the repository can stay in local memory.

## Commands

```sh
agent-notes search [--tag t]... [keywords...] --json   # find notes, best match first
agent-notes list [--tag t]... --json                   # all notes, newest first
agent-notes show <id> [--json]                         # full note including body
agent-notes add --title "Title" --description "One line" --tags a,b --body "Markdown"
agent-notes add --title "Title" --body - <<'EOF'       # body from stdin
...
EOF
agent-notes update <id> [--title T] [--description D] [--tags a,b] [--add-tag t] [--remove-tag t] [--body TEXT | --body -]
agent-notes delete <id>
```

`list` and `search` leave the body out of `--json` output unless `--full` is given.
Exit codes: 0 ok, 1 error, 2 usage error, 3 note not found.
