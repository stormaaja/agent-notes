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
cat >CHANGELOG.md <<'EOF'
# Changelog

## [Unreleased]

### Added

- Test change.

### Fixed

## [1.2.3] - 2026-01-01

- Previous release.
EOF
git add -A
git commit -qm "Test fixture"
git tag | xargs -r git tag -d >/dev/null
fixture=$(git rev-parse HEAD)

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

# expect_unchanged WHAT: HEAD is the fixture commit, there are no tags, VERSION is 1.2.3
# and CHANGELOG.md is as committed.
expect_unchanged() {
	local what=$1
	[[ $(git rev-parse HEAD) == "$fixture" ]] || fail "$what: HEAD moved"
	[[ -z $(git tag) ]] || fail "$what: tags created: $(git tag | tr '\n' ' ')"
	[[ $(cat VERSION) == 1.2.3 ]] || fail "$what: VERSION changed to $(cat VERSION)"
	git diff --quiet HEAD -- CHANGELOG.md || fail "$what: CHANGELOG.md changed"
	pass "$what: no commit, tag, VERSION or CHANGELOG.md change"
}

reset_fixture() {
	git reset -q --hard "$fixture"
	git clean -qfd
}

# --- usage ---
for arg in patch bogus "--bump"; do
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
[[ -z $(git tag) && $(git log -1 --format=%s) == "Bad version" ]] || fail "malformed VERSION: committed or tagged"
reset_fixture

# --- empty Unreleased section aborts before running tests ---
cat >CHANGELOG.md <<'EOF'
# Changelog

## [Unreleased]

### Added

## [1.2.3] - 2026-01-01

- Previous release.
EOF
git commit -qam "Empty Unreleased"
empty=$(git rev-parse HEAD)
run minor
expect_code 1 "empty Unreleased"
expect_out "add release notes under ## [Unreleased] in CHANGELOG.md first" "empty Unreleased"
[[ $out != *"go test"* ]] || fail "empty Unreleased: ran tests before checking the changelog"
[[ $(git rev-parse HEAD) == "$empty" && -z $(git tag) && $(cat VERSION) == 1.2.3 ]] || fail "empty Unreleased: repo changed"
[[ -z $(git status --porcelain) ]] || fail "empty Unreleased: tree dirty: $(git status --short)"
reset_fixture

# --- failing tests abort before anything changes ---
mkdir -p internal/broken
cat >internal/broken/broken_test.go <<'EOF'
package broken

import "testing"

func TestBroken(t *testing.T) { t.Fatal("broken on purpose") }
EOF
git add internal/broken
git commit -qm "Broken test"
broken=$(git rev-parse HEAD)
run build
expect_code 1 "failing tests"
expect_out "go test failed" "failing tests"
[[ $(git rev-parse HEAD) == "$broken" && -z $(git tag) && $(cat VERSION) == 1.2.3 ]] || fail "failing tests: repo changed"
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
# check_release TYPE NEW: release.sh TYPE turns the 1.2.3 fixture into release NEW.
check_release() {
	local type=$1 new=$2
	reset_fixture
	run "$type"
	expect_code 0 "$type"
	expect_out "Release v$new created" "$type"
	expect_out "git push" "$type"
	[[ $(cat VERSION) == "$new" ]] || fail "$type: VERSION = $(cat VERSION), want $new"
	[[ $(git rev-parse HEAD^) == "$fixture" ]] || fail "$type: expected exactly one new commit"
	[[ $(git log -1 --format=%B) == "Release v$new" ]] || fail "$type: commit message = $(git log -1 --format=%B)"
	[[ $(git diff --name-only HEAD^ HEAD | tr '\n' ' ') == "CHANGELOG.md VERSION " ]] || fail "$type: commit changes $(git diff --name-only HEAD^ HEAD | tr '\n' ' ')"
	grep -qxF "## [$new] - $(date +%F)" CHANGELOG.md || fail "$type: CHANGELOG.md has no '## [$new] - $(date +%F)' heading"
	if scripts/changelog.sh has-unreleased 2>/dev/null; then
		fail "$type: Unreleased section is not empty after the release"
	fi
	local notes
	notes=$(scripts/changelog.sh extract "$new" 2>&1) || fail "$type: changelog.sh extract $new failed: $notes"
	[[ $notes == $'### Added\n\n- Test change.' ]] || fail "$type: release notes ="$'\n'"$notes"
	[[ $(git cat-file -t "v$new" 2>/dev/null) == tag ]] || fail "$type: v$new is not an annotated tag"
	[[ $(git rev-parse "v$new^{commit}" 2>/dev/null) == $(git rev-parse HEAD) ]] || fail "$type: v$new does not point at HEAD"
	[[ $(git tag -l --format='%(contents:subject)' "v$new") == "Release v$new" ]] || fail "$type: tag message"
	[[ -z $(git status --porcelain) ]] || fail "$type: tree dirty: $(git status --short)"
	local got
	got=$(bin/agent-notes version 2>&1) || true
	[[ $got == "agent-notes $new" ]] || fail "$type: binary version = $got, want agent-notes $new"
	pass "$type: 1.2.3 -> $new committed, tagged and built"
}
check_release build 1.2.4
check_release minor 1.3.0
check_release major 2.0.0

# --- existing tag ---
reset_fixture
run build # v1.2.4 exists from the build release above
expect_code 1 "existing tag"
expect_out "v1.2.4" "existing tag"
[[ $(git rev-parse HEAD) == "$fixture" && $(cat VERSION) == 1.2.3 ]] || fail "existing tag: repo changed"

if [[ $failures -gt 0 ]]; then
	echo "$failures failure(s)" >&2
	exit 1
fi
echo "PASS"
