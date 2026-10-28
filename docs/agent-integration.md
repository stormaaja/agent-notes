# Using it with agents

The point of `agent-notes` is that agents read notes before they work and write durable
knowledge back into them. Agents need to be told to do this.

Agents that support the Model Context Protocol can also use notes as native tools; see
[MCP server](mcp.md).

## Claude Code

### Add instructions to `CLAUDE.md`

`agent-notes instructions` prints a short Markdown section that tells the agent to search
notes before starting a task, record conventions, gotchas and decisions as notes instead of
private memory, avoid duplicates and never store secrets. It also includes a command cheat
sheet.

```sh
agent-notes instructions >> CLAUDE.md
```

### Install the skill (optional)

[`integrations/claude-code/SKILL.md`](../integrations/claude-code/SKILL.md) is a Claude Code
skill with the same workflow. It triggers when you work in a repository with `.notes/`, when
you ask Claude to remember or note something, and when you look up project conventions.
Install it for yourself or for one repository:

```sh
# for all your projects
mkdir -p ~/.claude/skills/agent-notes
cp integrations/claude-code/SKILL.md ~/.claude/skills/agent-notes/SKILL.md

# or for one repository, committed so teammates get it too
mkdir -p .claude/skills/agent-notes
cp integrations/claude-code/SKILL.md .claude/skills/agent-notes/SKILL.md
```

### Migrate existing memories

Claude Code keeps per-project memories in `~/.claude/projects/<project>/memory/`, where
`<project>` is the project path with `/` replaced by `-`. Import them into the repository's
notes, review the result, then commit:

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

## Other agents

Any agent that can run shell commands can use `agent-notes`. Put the instructions where the
agent reads them, for example:

```sh
agent-notes instructions >> AGENTS.md   # Codex, Cursor and other agents that read AGENTS.md
```

For agents with their own rules files, paste the same output there.

Tell the agent to use `--json` for `search`, `list` and `show`, and to check exit codes
(`3` means the note does not exist). See [JSON output for agents](commands.md#json-output-for-agents).
