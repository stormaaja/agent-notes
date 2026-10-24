## Shared notes (agent-notes)

This repository keeps shared knowledge in `.notes/`, managed with the `agent-notes` CLI.
The notes are committed to git, so they reach teammates, other machines and other agents.
Treat them as the project's memory.

**Before starting a task**, look for relevant notes and read the ones that apply:

    agent-notes search <keywords> --json    # all keywords must match, best match first
    agent-notes search --tag <tag> --json   # filter by tag; repeat --tag to require several
    agent-notes show <id>                   # read a note in full

**When you learn something durable and non-obvious** (a convention, gotcha, decision or
how-to), record it as a note instead of saving it to private or local memory:

- Search first; prefer updating an existing note over creating a duplicate.
- Keep notes concise: a clear title, a one-line description, a few lowercase tags.
- Never store secrets, credentials, tokens or personal data.
- Skip what the code, git history or README already make obvious.
- Commit `.notes/` changes together with the related work.

Commands:

    agent-notes list [--tag <tag>] --json
    agent-notes add --title "Title" --description "One line" --tags a,b --body "Markdown"
    agent-notes add --title "Title" --body - < body.md         # body from stdin
    agent-notes update <id> --body "..." --add-tag t --description "..."
    agent-notes delete <id>
