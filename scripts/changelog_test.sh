#!/usr/bin/env bash
# Tests scripts/changelog.sh against a throwaway git repository with crafted Conventional
# Commits in a temporary directory; the real repository is never touched.
set -euo pipefail

src="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/changelog.sh"
work=$(mktemp -d "${TMPDIR:-/tmp}/changelog-test.XXXXXX")
trap 'rm -rf "$work"' EXIT

repo="$work/repo"
elsewhere="$work/elsewhere"
mkdir -p "$repo/scripts" "$elsewhere"
# The script finds the repository from its own location, so run a copy placed inside the
# fixture repository (left untracked).
script="$repo/scripts/changelog.sh"
cp "$src" "$script"
chmod +x "$script"
changelog="$repo/CHANGELOG.md"
url=https://github.com/stormaaja/agent-notes

git -C "$repo" init -q -b main
git -C "$repo" config user.name "Changelog Test"
git -C "$repo" config user.email "changelog-test@example.invalid"
git -C "$repo" config commit.gpgsign false
git -C "$repo" config tag.gpgsign false

# commit MESSAGE commits a change with MESSAGE (subject, blank line, body) at a fixed,
# increasing date so the log order is deterministic, and prints the full hash.
tick=0
commit() {
	tick=$((tick + 1))
	echo "$tick" >>"$repo/file.txt"
	git -C "$repo" add file.txt
	GIT_AUTHOR_DATE="2026-01-01T00:00:$(printf '%02d' "$tick")Z" \
		GIT_COMMITTER_DATE="2026-01-01T00:00:$(printf '%02d' "$tick")Z" \
		git -C "$repo" commit -q -m "$1"
	git -C "$repo" rev-parse HEAD
}

# entry SHA TEXT prints the changelog line for a commit.
entry() {
	echo "- $2 ([${1:0:7}]($url/commit/$1))"
}

c_init=$(commit "chore: initial commit")
c_first=$(commit "feat: add the first feature")
git -C "$repo" tag -a v0.1.0 -m "Release v0.1.0"
c_list=$(commit "feat(cli): add the list command

This is not a BREAKING CHANGE: the footer must start a line.")
c_missing=$(commit "fix(store): handle missing files")
c_perf=$(commit "perf: speed up search")
c_docs=$(commit "docs: describe search")
c_chore=$(commit "chore: tidy up")
c_test=$(commit "test(cli): cover list")
c_bang=$(commit "feat!: drop the old flag")
c_footer=$(commit "refactor(store): rename the loader

Moves things around.

BREAKING CHANGE: the loader API changed.")
c_upper=$(commit "Fix: handle an uppercase type")
c_plain=$(commit "Update readme")
c_revert=$(commit "revert: feat: add something")
c_style=$(commit "style: format")
git -C "$repo" checkout -q -b topic
c_typo=$(commit "fix: correct a typo in the output")
git -C "$repo" checkout -q main
c_mcp=$(commit "feat(mcp): Serve notes over stdio")
tick=$((tick + 1))
GIT_AUTHOR_DATE="2026-01-01T00:00:${tick}Z" GIT_COMMITTER_DATE="2026-01-01T00:00:${tick}Z" \
	git -C "$repo" merge -q --no-ff -m "Merge branch 'topic'" topic
c_ci=$(commit "ci: run tests

BREAKING-CHANGE: the workflow needs a token.")
: "$c_init $c_docs $c_chore $c_test $c_plain $c_revert $c_style"

failures=0
fail() {
	echo "FAIL: $*" >&2
	failures=$((failures + 1))
}
pass() { echo "ok:   $*"; }

out=""
err=""
code=0
# run ARGS... runs changelog.sh from an unrelated directory, capturing stdout in $out,
# stderr in $err and the exit code in $code.
run() {
	code=0
	out=$(cd "$elsewhere" && "$script" "$@" 2>"$work/stderr") || code=$?
	err=$(<"$work/stderr")
}

expect_code() {
	local want=$1 what=$2
	if [[ $code -eq $want ]]; then
		pass "$what: exit $want"
	else
		fail "$what: exit $code, want $want; stdout:"$'\n'"$out"$'\n'"stderr:"$'\n'"$err"
	fi
}

expect_out() {
	local want=$1 what=$2
	if [[ $out == "$want" ]]; then
		pass "$what: stdout"
	else
		fail "$what: stdout ="$'\n'"$out"$'\n'"want:"$'\n'"$want"
	fi
}

expect_err() {
	local want=$1 what=$2
	if [[ $err == *"$want"* ]]; then
		pass "$what: stderr mentions '$want'"
	else
		fail "$what: stderr does not contain '$want':"$'\n'"$err"
	fi
}

expect_file() {
	local want=$1 what=$2 got
	got=$(cat "${3:-$changelog}" 2>/dev/null) || got="<missing file>"
	if [[ $got == "$want" ]]; then
		pass "$what: CHANGELOG.md"
	else
		fail "$what: CHANGELOG.md ="$'\n'"$got"$'\n'"want:"$'\n'"$want"
	fi
}

expect_clean() {
	local what=$1 status
	status=$(git -C "$repo" status --porcelain --untracked-files=no)
	[[ -z $status ]] && pass "$what: no tracked changes" || fail "$what: tracked changes: $status"
}

header='# Changelog

All notable changes to this project are documented in this file. It is generated from
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) by `scripts/release.sh`;
do not edit it by hand.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).'

since_tag="### Breaking changes

$(entry "$c_bang" "Drop the old flag")
$(entry "$c_footer" "**store:** Rename the loader")
$(entry "$c_ci" "Run tests")

### Added

$(entry "$c_list" "**cli:** Add the list command")
$(entry "$c_mcp" "**mcp:** Serve notes over stdio")

### Fixed

$(entry "$c_missing" "**store:** Handle missing files")
$(entry "$c_upper" "Handle an uppercase type")
$(entry "$c_typo" "Correct a typo in the output")

### Changed

$(entry "$c_perf" "Speed up search")"

# --- usage ---
run
expect_code 2 "no args"
expect_err "Usage:" "no args"
run bogus
expect_code 2 "unknown command"
expect_err "Usage:" "unknown command"
run has-unreleased
expect_code 2 "has-unreleased is gone"
run extract
expect_code 2 "extract without version"
run notes
expect_code 2 "notes without version"
run release 1.2.0
expect_code 2 "release without date"
run release 1.2 2026-03-15
expect_code 2 "release bad version"
run release 1.2.0 15.3.2026
expect_code 2 "release bad date"
run generate a b c
expect_code 2 "generate with three args"
run bump-type a b c
expect_code 2 "bump-type with three args"
run -h
expect_code 0 "-h"
[[ $out == *"Usage:"* ]] && pass "-h: usage on stdout" || fail "-h: stdout = $out"

# --- generate ---
run generate
expect_code 0 "generate since latest tag"
expect_out "$since_tag" "generate since latest tag"

run generate v0.1.0 HEAD
expect_code 0 "generate explicit range"
expect_out "$since_tag" "generate explicit range"

run generate "$c_first" "$c_perf"
expect_code 0 "generate partial range"
expect_out "### Added

$(entry "$c_list" "**cli:** Add the list command")

### Fixed

$(entry "$c_missing" "**store:** Handle missing files")

### Changed

$(entry "$c_perf" "Speed up search")" "generate partial range"

run generate "$c_perf" "$c_test"
expect_code 0 "generate with nothing releasable"
expect_out "" "generate with nothing releasable"

run generate nosuchref
expect_code 1 "generate bad ref"
expect_err "nosuchref" "generate bad ref"

# Without any tag the whole history counts; the latest tag only bounds the default range.
git -C "$repo" tag -d v0.1.0 >/dev/null
run generate "" "$c_list"
expect_code 0 "generate without tags"
expect_out "### Added

$(entry "$c_first" "Add the first feature")
$(entry "$c_list" "**cli:** Add the list command")" "generate without tags"
git -C "$repo" tag -a v0.1.0 -m "Release v0.1.0" "$c_first"

# Tags not shaped like vX.Y.Z do not bound the range.
git -C "$repo" tag other "$c_missing"
run generate "" "$c_perf"
expect_code 0 "generate ignores non-version tags"
expect_out "### Added

$(entry "$c_list" "**cli:** Add the list command")

### Fixed

$(entry "$c_missing" "**store:** Handle missing files")

### Changed

$(entry "$c_perf" "Speed up search")" "generate ignores non-version tags"
git -C "$repo" tag -d other >/dev/null

# --- bump-type ---
run bump-type
expect_code 0 "bump-type breaking"
expect_out "major" "bump-type breaking"
run bump-type v0.1.0 "$c_test"
expect_code 0 "bump-type feat"
expect_out "minor" "bump-type feat"
run bump-type "$c_list" "$c_missing"
expect_code 0 "bump-type fix"
expect_out "patch" "bump-type fix"
run bump-type "$c_missing" "$c_perf"
expect_code 0 "bump-type perf"
expect_out "patch" "bump-type perf"
run bump-type "$c_perf" "$c_test"
expect_code 1 "bump-type nothing releasable"
expect_out "" "bump-type nothing releasable"
expect_err "no releasable commits" "bump-type nothing releasable"

# --- release ---
rm -f "$changelog"
run release 0.2.0 2026-03-15
expect_code 0 "release creates CHANGELOG.md"
expect_file "$header

## [0.2.0] - 2026-03-15

$since_tag" "release creates CHANGELOG.md"
expect_clean "release creates CHANGELOG.md"

run release 0.2.0 2026-03-16
expect_code 1 "release existing version"
expect_err "0.2.0" "release existing version"

printf '%s\n' "$header" "" "## [0.1.0] - 2026-01-01" "" "### Added" "" "- First." >"$changelog"
run release 0.2.0 2026-03-15 v0.1.0 "$c_missing"
expect_code 0 "release above older versions"
expect_file "$header

## [0.2.0] - 2026-03-15

### Added

$(entry "$c_list" "**cli:** Add the list command")

### Fixed

$(entry "$c_missing" "**store:** Handle missing files")

## [0.1.0] - 2026-01-01

### Added

- First." "release above older versions"
saved=$(<"$changelog")

run release 0.3.0 2026-03-15 "$c_perf" "$c_test"
expect_code 1 "release with nothing releasable"
expect_err "no releasable commits" "release with nothing releasable"
expect_file "$saved" "release with nothing releasable leaves file"

run release 0.1.0 2026-03-15
expect_code 1 "release existing older version"
expect_file "$saved" "release existing older version leaves file"

# CHANGELOG overrides the file.
other="$work/OTHER.md"
printf '%s\n' "# Other" "" "Intro." >"$other"
CHANGELOG="$other" run release 1.0.0 2026-04-01 "$c_perf" "$c_bang"
expect_code 0 "release with CHANGELOG override"
expect_file "# Other

Intro.

## [1.0.0] - 2026-04-01

### Breaking changes

$(entry "$c_bang" "Drop the old flag")" "release with CHANGELOG override" "$other"
expect_file "$saved" "release with CHANGELOG override leaves CHANGELOG.md"

# --- extract ---
run extract 0.2.0
expect_code 0 "extract"
expect_out "### Added

$(entry "$c_list" "**cli:** Add the list command")

### Fixed

$(entry "$c_missing" "**store:** Handle missing files")" "extract"
run extract 0.1.0
expect_code 0 "extract last section"
expect_out "### Added

- First." "extract last section"
run extract 0.1
expect_code 1 "extract prefix of a version"
run extract 9.9.9
expect_code 1 "extract missing"
expect_err "9.9.9" "extract missing"
printf '%s\n' "# Changelog" "" "## [2.0.0] - 2026-03-01" "" "" "## [1.0.0] - 2026-01-01" "" "- x" "" "[1.0.0]: https://example.invalid" >"$other"
CHANGELOG="$other" run extract 2.0.0
expect_code 1 "extract empty"
CHANGELOG="$other" run extract 1.0.0
expect_code 0 "extract before link footer"
expect_out "- x" "extract before link footer"
CHANGELOG="$work/missing.md" run extract 1.0.0
expect_code 1 "extract missing file"
expect_err "not found" "extract missing file"

# --- notes ---
printf '%s\n' "$saved" >"$changelog"
run notes 0.1.0
expect_code 0 "notes first release"
expect_out "### Added

- First.

**Full changelog**: $url/commits/v0.1.0" "notes first release"

git -C "$repo" tag -a v0.2.0 -m "Release v0.2.0" "$c_missing"
run notes 0.2.0
expect_code 0 "notes later release"
expect_out "### Added

$(entry "$c_list" "**cli:** Add the list command")

### Fixed

$(entry "$c_missing" "**store:** Handle missing files")

**Full changelog**: $url/compare/v0.1.0...v0.2.0" "notes later release"

run notes 0.3.0
expect_code 1 "notes without tag"
expect_err "v0.3.0" "notes without tag"
git -C "$repo" tag v0.3.0 "$c_perf"
run notes 0.3.0
expect_code 1 "notes without section"
expect_err "0.3.0" "notes without section"

# After v0.2.0 the default range starts there.
git -C "$repo" tag -d v0.3.0 >/dev/null
run generate "" "$c_perf"
expect_code 0 "generate after a later tag"
expect_out "### Changed

$(entry "$c_perf" "Speed up search")" "generate after a later tag"

if [[ $failures -gt 0 ]]; then
	echo "$failures failure(s)" >&2
	exit 1
fi
echo "PASS"
