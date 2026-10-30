# Development

## Prerequisites

- Go 1.26 or newer (`go.mod` pins `go 1.26.10`; the `go` command fetches that toolchain if
  needed).
- Bash and git for the scripts in `scripts/` and their tests.

## Repository layout

| Path | What it holds |
| --- | --- |
| `cmd/agent-notes` | `main`: wires `os` stdin/stdout/stderr, cwd, env and clock into `cli.Run` |
| `internal/cli` | Command line interface: argument parsing, commands, output, `instructions.md` |
| `internal/note` | Note model, Markdown-with-frontmatter file format, slug and tag helpers |
| `internal/store` | Notes as files in a notes directory, and resolving that directory |
| `internal/search` | Filtering and ranking notes by tags and keywords |
| `internal/notejson` | JSON shape of notes shared by the CLI and the MCP server |
| `internal/mcpserver` | Notes store exposed as Model Context Protocol tools |
| `internal/memory` | Reading Claude Code memory files for `import-memory` |
| `internal/version` | Version reporting (build-time ldflags, module version, `devel`) |
| `integrations` | Agent skills as `<tool>/agent-notes/SKILL.md`: [Claude Code](../integrations/claude-code/agent-notes/SKILL.md); `skills_test.go` checks their frontmatter |
| `scripts/` | `release.sh`, `changelog.sh`, `dist.sh` and their `*_test.sh` tests |
| `.github/workflows` | Release workflow for pushed `vX.Y.Z` tags (see [Releasing](releasing.md)) |
| `docs/` | These docs |

## Build and run

```sh
go run ./cmd/agent-notes help                # run from the checkout (version "devel")
go build -o bin/agent-notes ./cmd/agent-notes
scripts/release.sh                           # build bin/agent-notes at the version in VERSION
```

## Tests and checks

```sh
go test ./... -race
go vet ./...
gofmt -l .                # must print nothing
scripts/changelog_test.sh # tests the changelog script in a throwaway git repository
scripts/release_test.sh   # tests the release script in a throwaway clone
scripts/dist_test.sh      # tests the release archives in a throwaway clone
```

The shell tests create their throwaway repositories under `$TMPDIR` (default `/tmp`) and
remove them afterwards; set `TMPDIR` to a writable directory if `/tmp` is not usable, for
example in a sandbox.

## Test-driven development

All changes are made test first: red, green, refactor.

1. **Red.** Write a failing test for the behavior you want. Run it and see it fail, for the
   reason you expect.
2. **Green.** Write the minimal code that makes it pass.
3. **Refactor.** Clean up the code and the tests while everything stays green.

- Tests drive public APIs. CLI behavior is tested through `cli.Run` with a `t.TempDir()`
  working directory and an injected `cli.Env` (stdin, stdout, stderr, cwd, environment
  variables and a fake clock), so no test touches the real environment.
- Tests are table-driven where several cases share one shape.
- A bug fix starts with a regression test that reproduces the bug and fails.
- Shell scripts are tested by their `*_test.sh` script, which runs them in throwaway git
  repositories; the real repository is never touched.

## Commit messages

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):
`type(scope)!: description`, for example `feat(cli): add the export command` or
`fix(store): keep file permissions on update`. The changelog and the release version are
generated from them (see [Releasing](releasing.md#generated-changelog)), so pick the type that
describes the change for users.

| Type | Use for | In the changelog |
| --- | --- | --- |
| `feat` | New user-facing behavior | Added |
| `fix` | Bug fixes | Fixed |
| `perf` | Performance improvements | Changed |
| `refactor` | Code changes that keep behavior | — |
| `test` | Adding or changing tests only | — |
| `docs` | Documentation only | — |
| `build` | Build and release scripts, dependencies | — |
| `ci` | GitHub workflows | — |
| `style` | Formatting only | — |
| `chore` | Other maintenance, release commits | — |
| `revert` | Reverting an earlier commit | — |

- Scopes in use: `note`, `store`, `search`, `cli`, `mcp`, `memory`, `changelog`, `release`.
  The scope is optional.
- Breaking changes add `!` after the type or scope (`feat(cli)!: rename --tag to --label`) or
  a `BREAKING CHANGE:` footer. They land under "Breaking changes" whatever the type.
- The description is imperative, starts lowercase and has no trailing period.
- Keep commits small and focused: one logical change per commit, and every commit passes the
  tests.

## Dependencies

Prefer the standard library. The current direct dependencies are
[`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) for note, memory and skill frontmatter and the
[MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) for the MCP server. A new
dependency needs a reason the standard library cannot cover, stated in the commit message.
