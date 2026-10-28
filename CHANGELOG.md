# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Notes stored as Markdown files with YAML frontmatter in a `.notes/` directory.
- CLI commands `init`, `add`, `show`, `list`, `update`, `delete` and `search` (tag and
  keyword search, best match first), with `--json` output for agents.
- Notes directory resolution: `--dir`, `AGENT_NOTES_DIR`, the nearest `.notes/` up to the
  git root, then `.notes/` at the git root or the current directory.
- `import-memory` command to import Claude Code memory files as notes.
- `instructions` command printing agent instructions for `CLAUDE.md` or `AGENTS.md`.
- Claude Code skill for agent-notes.
- `mcp` command serving notes as tools over stdio (Model Context Protocol).
- `version` command and `--version` flag.
- `scripts/release.sh` to bump `VERSION`, build, commit and tag releases.
- GitHub Actions release workflow publishing binaries for Linux, macOS and Windows with
  release notes from this changelog.
