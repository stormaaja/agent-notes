# Releasing

The current version is in the `VERSION` file (`X.Y.Z`); releases are tagged `vX.Y.Z`.
[CHANGELOG.md](../CHANGELOG.md) ([Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format)
and the GitHub release notes are generated from the commit messages (see
[Commit messages](development.md#commit-messages)), so there is nothing to write by hand:

```sh
scripts/release.sh auto       # cut the release the commits call for
git push --follow-tags        # publish it
```

## Generated changelog

Commit types and where they land in the changelog:

| Commit | Changelog section | `auto` bump |
| --- | --- | --- |
| `feat!:`, `fix(cli)!:`, … or a `BREAKING CHANGE:` footer (any type) | Breaking changes | major |
| `feat:` | Added | minor |
| `fix:` | Fixed | patch |
| `perf:` | Changed | patch |
| `docs`, `test`, `chore`, `ci`, `build`, `style`, `refactor`, `revert` | — | — |

Each entry is the commit description with a link to the commit, prefixed with the scope if
there is one (`- **cli:** Add the export command (abc1234)`). Merge commits and messages that are
not Conventional Commits are left out. The changelog covers the commits since the previous
`vX.Y.Z` tag; if none of them is a feat, fix, perf or breaking change, there is nothing to
release and the script stops.

`scripts/changelog.sh` does the changelog work: `generate` and `bump-type` preview the notes
and the bump for the commits since the last tag, `release X.Y.Z YYYY-MM-DD` adds the version
section, `extract X.Y.Z` prints it and `notes X.Y.Z` prints the GitHub release notes (the
section plus a link to the full list of changes since the previous tag).

## release.sh

```sh
scripts/release.sh          # build bin/agent-notes at the current version
scripts/release.sh auto     # bump by the commits: major, minor or patch
scripts/release.sh build    # patch release: 1.2.3 -> 1.2.4
scripts/release.sh minor    # 1.2.3 -> 1.3.0
scripts/release.sh major    # 1.2.3 -> 2.0.0
```

With a bump type, the script refuses to run if the working tree has any staged, unstaged or
untracked changes, if `VERSION` is malformed, if the new tag already exists, or if there are
no releasable commits since the last tag. It then runs `go vet ./...` and `go test ./...`,
writes `VERSION`, builds `bin/agent-notes`, adds a `## [X.Y.Z] - YYYY-MM-DD` section (dated
today) generated from the commits to `CHANGELOG.md`, commits `VERSION` and `CHANGELOG.md` as
`chore(release): vX.Y.Z`, creates the annotated tag `vX.Y.Z` and prints the new changelog
section. It never pushes.

## GitHub workflow and release assets

When the `vX.Y.Z` tag is pushed, the workflow
([`.github/workflows/release.yml`](../.github/workflows/release.yml)) checks that it matches
`VERSION`, runs `go vet` and `go test -race`, and publishes a release whose notes are the
version's changelog section plus a "Full changelog" link to the changes since the previous tag,
with these assets, built by `scripts/dist.sh X.Y.Z` (run it locally to inspect the output in
`dist/`):

- `agent-notes_X.Y.Z_<os>_<arch>.tar.gz` for `linux` and `darwin`, and
  `agent-notes_X.Y.Z_windows_<arch>.zip` (binary `agent-notes.exe`), each for `amd64` and
  `arm64`, containing the binary, `README.md`, `LICENSE`, `CHANGELOG.md` and the `docs/`
  pages (in a `docs/` directory next to the binary);
- `checksums.txt` with the SHA-256 sums of the archives.

After that, `go install github.com/stormaaja/agent-notes/cmd/agent-notes@vX.Y.Z` also installs
the release.

## Version command

`agent-notes version` (or `--version`, `--json` for `{"version": "..."}`) prints the version,
which the MCP server also reports. It is resolved as:

1. the value set at build time with
   `-ldflags "-X github.com/stormaaja/agent-notes/internal/version.version=X.Y.Z"`
   (what `scripts/release.sh` does),
2. else the module version recorded by `go install …@vX.Y.Z`, without the `v`,
3. else `devel` (for example `go build` or `go run` from a checkout).
