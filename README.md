# agent-notes

Shared knowledge notes for people and coding agents, stored as plain Markdown in your
repository.

Coding agents such as Claude Code and pi learn things while they work: conventions, gotchas,
decisions, how to run things. Usually that knowledge goes into memory on one laptop, so your
teammates, your other machines and other agents never see it. `agent-notes` keeps it in a
`.notes/` folder that you commit with the code. Everyone who clones the repository, and every
agent working in it, can search and extend the same notes.

- Notes are readable Markdown files with YAML frontmatter, so they are easy to review in diffs.
- One small CLI with human-readable output and `--json` for agents.
- A built-in MCP server, so agents can use notes as native tools.
- Ready-made agent instructions, skills for Claude Code and pi, and an importer for existing
  Claude Code memories.

## Install

```sh
go install github.com/stormaaja/agent-notes/cmd/agent-notes@latest
```

Use `@vX.Y.Z` instead of `@latest` for a specific release.

This needs Go 1.26 or newer (the `go` command fetches a newer toolchain if needed) and puts `agent-notes` in `$(go env GOPATH)/bin`.

Without Go, download a prebuilt binary for Linux, macOS or Windows (amd64 or arm64) from the
[releases page](https://github.com/stormaaja/agent-notes/releases), for example:

```sh
version=X.Y.Z  # without the "v"
curl -fLO "https://github.com/stormaaja/agent-notes/releases/download/v$version/agent-notes_${version}_linux_amd64.tar.gz"
curl -fLO "https://github.com/stormaaja/agent-notes/releases/download/v$version/checksums.txt"
sha256sum --ignore-missing -c checksums.txt
tar -xzf "agent-notes_${version}_linux_amd64.tar.gz" agent-notes
install agent-notes ~/.local/bin/
```

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

Then tell your agents to use the notes: see [Using it with agents](docs/agent-integration.md).

## Documentation

- [Note format](docs/note-format.md): the note file format and where notes live.
- [Commands](docs/commands.md): command reference, examples, search ranking and JSON output.
- [Using it with agents](docs/agent-integration.md): agent instructions, the Claude Code and
  pi skills, importing Claude Code memories, other agents.
- [MCP server](docs/mcp.md): the MCP tools and how to set them up in Claude Code, pi and other
  clients.
- [Development](docs/development.md): repository layout, building, testing, TDD workflow,
  commit conventions and dependencies.
- [Releasing](docs/releasing.md): versioning, the release script, the generated changelog and
  release assets.
- [Changelog](CHANGELOG.md): changes in each release.

## Contributing

Read [AGENTS.md](AGENTS.md) for the rules every change follows (test-driven development,
Conventional Commits) and [docs/development.md](docs/development.md) for the details.

## License

See [LICENSE](LICENSE).
