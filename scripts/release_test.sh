#!/usr/bin/env bash
# Tests scripts/release.sh in a throwaway clone; the real repository is never touched.
# The release.sh and changelog.sh under test are the ones in this working tree, committed
# into the clone.
set -euo pipefail

repo=$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
work=$(mktemp -d "${TMPDIR:-/tmp}/release-test.XXXXXX")
trap 'rm -rf "$work"' EXIT

clone="$work/clone"
git clone -q "$repo" "$clone"
cd "$clone"
git config user.name "Release Test"
git config user.email "release-test@example.invalid"
git config commit.gpgsign false
git config tag.gpgsign false
mkdir -p scripts
cp "$repo/scripts/release.sh" "$repo/scripts/changelog.sh" scripts/
cp "$repo/.gitignore" .gitignore
chmod +x scripts/release.sh scripts/changelog.sh
printf '1.2.3\n' >VERSION
cat >CHANGELOG.md <<'EOF2'
# Changelog

Generated.

## [1.2.3] - 2026-01-01

- Previous release.
EOF2
git add -A
git commit -qm "chore: test fixture"
git tag | xargs -r git tag -d >/dev/null
git tag -a v1.2.3 -m "Release v1.2.3"
base=$(git rev-parse HEAD)

url=https://github.com/stormaaja/agent-notes
# entry SHA TEXT prints the changelog line for a commit.
entry() {
	echo "- $2 ([${1:0:7}]($url/commit/$1))"
}
# commit MESSAGE adds an empty commit and prints its hash.
commit() {
	git commit -q --allow-empty -m "$1"
	git rev-parse HEAD
}

c_feat=$(commit "feat(cli): add the thing")
c_fix=$(commit "fix: repair the other thing")
commit "docs: describe the thing" >/dev/null
fixture=$(git rev-parse HEAD)
fixture_notes="### Added

$(entry "$c_feat" "**cli:** Add the thing")

### Fixed

$(entry "$c_fix" "Repair the other thing")"

failures=0
fail() {
	echo "FAIL: $*" >&2
	failures=$((failures + 1))
}
pass() { echo "ok:   $*"; }

out=""
code=0
# run ARGS... runs release.sh from a subdirectory (to check it finds the repo root),
# capturing combined output in $out and the exit code in $code.
run() {
	code=0
	out=$(cd internal && ../scripts/release.sh "$@" 2>&1) || code=$?
}

expect_code() {
	local want=$1 what=$2
	if [[ $code -eq $want ]]; then
		pass "$what: exit $want"
	else
		fail "$what: exit $code, want $want; output:"$'\n'"$out"
	fi
}

expect_out() {
	local want=$1 what=$2
	if [[ $out == *"$want"* ]]; then
		pass "$what: output mentions '$want'"
	else
		fail "$what: output does not contain '$want':"$'\n'"$out"
	fi
}

# expect_unchanged WHAT [HEAD]: HEAD is the fixture commit (or HEAD), v1.2.3 is the only
# tag, VERSION is 1.2.3 and CHANGELOG.md is as committed.
expect_unchanged() {
	local what=$1
	[[ $(git rev-parse HEAD) == "${2:-$fixture}" ]] || fail "$what: HEAD moved"
	[[ $(git tag) == v1.2.3 ]] || fail "$what: tags: $(git tag | tr '\n' ' ')"
	[[ $(cat VERSION) == 1.2.3 ]] || fail "$what: VERSION changed to $(cat VERSION)"
	git diff --quiet HEAD -- CHANGELOG.md || fail "$what: CHANGELOG.md changed"
	pass "$what: no commit, tag, VERSION or CHANGELOG.md change"
}

# reset_fixture [COMMIT]: back to the fixture commit (or COMMIT) with only the v1.2.3 tag.
reset_fixture() {
	git reset -q --hard "${1:-$fixture}"
	git clean -qfd
	git tag | { grep -vx v1.2.3 || true; } | xargs -r git tag -d >/dev/null
}

# --- usage ---
for arg in patch bogus "--bump" Auto; do
	run "$arg"
	expect_code 2 "invalid arg $arg"
	expect_out "Usage:" "invalid arg $arg"
done
run build minor
expect_code 2 "two args"
for arg in -h --help; do
	run "$arg"
	expect_code 0 "$arg"
	expect_out "Usage:" "$arg"
done
expect_unchanged "usage"

# --- dirty tree ---
echo "// change" >>cmd/agent-notes/main.go
run build
expect_code 1 "modified file"
expect_out "main.go" "modified file"
git checkout -q -- cmd/agent-notes/main.go
expect_unchanged "modified file"

echo staged >staged.txt
git add staged.txt
run minor
expect_code 1 "staged file"
expect_out "staged.txt" "staged file"
git rm -q --cached staged.txt
rm staged.txt
expect_unchanged "staged file"

echo untracked >untracked.txt
run major
expect_code 1 "untracked file"
expect_out "untracked.txt" "untracked file"
rm untracked.txt
expect_unchanged "untracked file"

# --- malformed VERSION ---
printf '1.2\n' >VERSION
git commit -qam "Bad version"
run build
expect_code 1 "malformed VERSION"
expect_out "VERSION" "malformed VERSION"
[[ $(git tag) == v1.2.3 && $(git log -1 --format=%s) == "Bad version" ]] || fail "malformed VERSION: committed or tagged"
reset_fixture

# --- no releasable commits aborts before running tests ---
for args in "build docs:" "auto docs:" "minor tag:"; do
	read -r type kind <<<"$args"
	reset_fixture "$base"
	if [[ $kind == docs: ]]; then
		commit "docs: explain more" >/dev/null
		commit "chore(deps): tidy" >/dev/null
		commit "Not a conventional commit" >/dev/null
	fi
	head=$(git rev-parse HEAD)
	run "$type"
	expect_code 1 "no releasable commits ($type, $kind)"
	expect_out "no releasable commits" "no releasable commits ($type, $kind)"
	[[ $out != *"go test"* ]] || fail "no releasable commits ($type, $kind): ran tests before checking the commits"
	expect_unchanged "no releasable commits ($type, $kind)" "$head"
	[[ -z $(git status --porcelain) ]] || fail "no releasable commits ($type, $kind): tree dirty: $(git status --short)"
done
reset_fixture

# --- failing tests abort before anything changes ---
mkdir -p internal/broken
cat >internal/broken/broken_test.go <<'EOF'
package broken

import "testing"

func TestBroken(t *testing.T) { t.Fatal("broken on purpose") }
EOF
git add internal/broken
git commit -qm "test: add a broken test"
broken=$(git rev-parse HEAD)
run build
expect_code 1 "failing tests"
expect_out "go test failed" "failing tests"
expect_unchanged "failing tests" "$broken"
[[ -z $(git status --porcelain) ]] || fail "failing tests: tree dirty: $(git status --short)"
reset_fixture

# --- no argument: build only ---
rm -rf bin
run
expect_code 0 "no arg"
expect_unchanged "no arg"
[[ -z $(git status --porcelain) ]] || fail "no arg: tree dirty (bin/ not ignored?): $(git status --short)"
if [[ $(bin/agent-notes version 2>&1) == "agent-notes 1.2.3" ]]; then
	pass "no arg: binary reports 1.2.3"
else
	fail "no arg: binary version = $(bin/agent-notes version 2>&1)"
fi

# --- bumps ---
# check_release TYPE NEW NOTES [COMMIT...]: on top of the fixture (or on top of v1.2.3 with
# the given commit messages), release.sh TYPE releases NEW with changelog section NOTES.
check_release() {
	local type=$1 new=$2 notes=$3 start
	shift 3
	if [[ $# -gt 0 ]]; then
		reset_fixture "$base"
		local message
		for message in "$@"; do
			commit "$message" >/dev/null
		done
	else
		reset_fixture
	fi
	start=$(git rev-parse HEAD)
	# entry @HEAD@ ... stands for the last new commit.
	notes=${notes//\[@HEAD@\]/[${start:0:7}]}
	notes=${notes//@HEAD@/$start}
	run "$type"
	local what="$type ${*:-fixture}"
	expect_code 0 "$what"
	expect_out "Release v$new created" "$what"
	expect_out "$notes"$'\n\nPush with: git push --follow-tags' "$what"
	[[ $(cat VERSION) == "$new" ]] || fail "$what: VERSION = $(cat VERSION), want $new"
	[[ $(git rev-parse HEAD^) == "$start" ]] || fail "$what: expected exactly one new commit"
	[[ $(git log -1 --format=%B) == "chore(release): v$new" ]] || fail "$what: commit message = $(git log -1 --format=%B)"
	[[ $(git diff --name-only HEAD^ HEAD | tr '\n' ' ') == "CHANGELOG.md VERSION " ]] || fail "$what: commit changes $(git diff --name-only HEAD^ HEAD | tr '\n' ' ')"
	local want_changelog
	want_changelog="# Changelog

Generated.

## [$new] - $(date +%F)

$notes

## [1.2.3] - 2026-01-01

- Previous release."
	[[ $(<CHANGELOG.md) == "$want_changelog" ]] || fail "$what: CHANGELOG.md ="$'\n'"$(<CHANGELOG.md)"$'\n'"want:"$'\n'"$want_changelog"
	[[ $(git cat-file -t "v$new" 2>/dev/null) == tag ]] || fail "$what: v$new is not an annotated tag"
	[[ $(git rev-parse "v$new^{commit}" 2>/dev/null) == $(git rev-parse HEAD) ]] || fail "$what: v$new does not point at HEAD"
	[[ $(git tag -l --format='%(contents:subject)' "v$new") == "Release v$new" ]] || fail "$what: tag message"
	[[ -z $(git status --porcelain) ]] || fail "$what: tree dirty: $(git status --short)"
	local got
	got=$(bin/agent-notes version 2>&1) || true
	[[ $got == "agent-notes $new" ]] || fail "$what: binary version = $got, want agent-notes $new"
	# The release commit is not releasable itself.
	if scripts/changelog.sh bump-type >/dev/null 2>&1; then
		fail "$what: commits after the release are releasable"
	fi
	pass "$what: 1.2.3 -> $new committed, tagged and built"
}
check_release build 1.2.4 "$fixture_notes"
check_release minor 1.3.0 "$fixture_notes"
check_release major 2.0.0 "$fixture_notes"
check_release auto 1.3.0 "$fixture_notes"
check_release auto 1.2.4 "### Fixed

$(entry @HEAD@ "**store:** Handle a missing file")" "docs: note" "fix(store): handle a missing file"
check_release auto 1.2.4 "### Changed

$(entry @HEAD@ "Speed up search")" "perf: speed up search"
check_release auto 2.0.0 "### Breaking changes

$(entry @HEAD@ "**cli:** Drop the old flag")" "docs: note" "feat(cli)!: drop the old flag"
check_release auto 2.0.0 "### Breaking changes

$(entry @HEAD@ "Rework the store")" "refactor: rework the store

BREAKING CHANGE: notes move to a new directory."

# --- existing tag ---
reset_fixture
git tag v1.2.4 "$base"
run build
expect_code 1 "existing tag"
expect_out "v1.2.4" "existing tag"
[[ $(git rev-parse HEAD) == "$fixture" && $(cat VERSION) == 1.2.3 ]] || fail "existing tag: repo changed"
[[ $out != *"go test"* ]] || fail "existing tag: ran tests"
git tag -d v1.2.4 >/dev/null

if [[ $failures -gt 0 ]]; then
	echo "$failures failure(s)" >&2
	exit 1
fi
echo "PASS"
