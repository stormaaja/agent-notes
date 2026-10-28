#!/usr/bin/env bash
# Tests scripts/dist.sh in a throwaway clone; the real repository is never touched.
# The dist.sh and docs under test are the ones in this working tree, committed into the clone.
set -euo pipefail

repo=$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
work=$(mktemp -d "${TMPDIR:-/tmp}/dist-test.XXXXXX")
trap 'rm -rf "$work"' EXIT

clone="$work/clone"
git clone -q "$repo" "$clone"
cd "$clone"
git config user.name "Dist Test"
git config user.email "dist-test@example.invalid"
git config commit.gpgsign false
mkdir -p scripts
cp "$repo/scripts/dist.sh" scripts/
chmod +x scripts/dist.sh
rm -rf docs
cp -R "$repo/docs" docs
cp "$repo/README.md" "$repo/LICENSE" "$repo/CHANGELOG.md" .
git add -A
git commit -qm "chore: test fixture" --allow-empty
fixture=$(git rev-parse HEAD)

failures=0
fail() {
	echo "FAIL: $*" >&2
	failures=$((failures + 1))
}
pass() { echo "ok:   $*"; }

out=""
code=0
# run ARGS... runs dist.sh from a subdirectory (to check it finds the repo root),
# capturing combined output in $out and the exit code in $code.
run() {
	code=0
	out=$(cd internal && ../scripts/dist.sh "$@" 2>&1) || code=$?
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

# --- usage ---
run
expect_code 2 "no arg"
expect_out "Usage:" "no arg"
run 1.2.3 1.2.4
expect_code 2 "two args"
for arg in 1.2 v1.2.3 1.2.x bogus; do
	run "$arg"
	expect_code 2 "bad version $arg"
	expect_out "Usage:" "bad version $arg"
done
for arg in -h --help; do
	run "$arg"
	expect_code 0 "$arg"
	expect_out "Usage:" "$arg"
done
[[ ! -e dist ]] || fail "usage: dist/ created"

# --- missing docs ---
git rm -rq docs
git commit -qm "docs: remove docs"
run 1.2.3
expect_code 1 "missing docs"
expect_out "docs" "missing docs"
[[ ! -e dist ]] || fail "missing docs: dist/ created"
git reset -q --hard "$fixture"

# --- archives ---
version=1.2.3-test
run "$version"
expect_code 0 "build $version"

want_archives="agent-notes_${version}_darwin_amd64.tar.gz
agent-notes_${version}_darwin_arm64.tar.gz
agent-notes_${version}_linux_amd64.tar.gz
agent-notes_${version}_linux_arm64.tar.gz
agent-notes_${version}_windows_amd64.zip
agent-notes_${version}_windows_arm64.zip
checksums.txt"
got_archives=$(ls -1 dist 2>/dev/null || true)
if [[ $got_archives == "$want_archives" ]]; then
	pass "dist/ holds the 6 archives and checksums.txt"
else
	fail "dist/ ="$'\n'"$got_archives"$'\n'"want:"$'\n'"$want_archives"
fi

if [[ $(grep -c . dist/checksums.txt 2>/dev/null) == 6 ]] &&
	(cd dist && if command -v sha256sum >/dev/null; then sha256sum -c --quiet checksums.txt; else shasum -a 256 -c --quiet checksums.txt; fi) >/dev/null 2>&1; then
	pass "checksums.txt verifies 6 archives"
else
	fail "checksums.txt does not verify:"$'\n'"$(cat dist/checksums.txt 2>&1)"
fi

docs=$(printf '%s\n' README.md LICENSE CHANGELOG.md docs/*.md)
for archive in dist/*.tar.gz dist/*.zip; do
	[[ -e $archive ]] || continue
	binary=agent-notes
	if [[ $archive == *.zip ]]; then
		binary=agent-notes.exe
		got=$(unzip -Z1 "$archive")
	else
		got=$(tar -tzf "$archive")
	fi
	want=$(printf '%s\n%s\n' "$binary" "$docs" | sort)
	got=$(sort <<<"$got")
	if [[ $got == "$want" ]]; then
		pass "$(basename "$archive"): binary, README.md, LICENSE, CHANGELOG.md and docs/*.md"
	else
		fail "$(basename "$archive") contents ="$'\n'"$got"$'\n'"want:"$'\n'"$want"
	fi
done

# The archive for the host platform runs and reports the version.
host="$(go env GOOS)_$(go env GOARCH)"
host_archive="dist/agent-notes_${version}_$host.tar.gz"
if [[ -f $host_archive ]]; then
	mkdir -p "$work/host"
	tar -xzf "$host_archive" -C "$work/host" agent-notes
	got=$("$work/host/agent-notes" version 2>&1) || true
	if [[ $got == "agent-notes $version" ]]; then
		pass "$host binary reports $version"
	else
		fail "$host binary version = $got, want agent-notes $version"
	fi
else
	echo "skip: no tar.gz archive for host $host"
fi

[[ -z $(git status --porcelain) ]] || fail "tree dirty (dist/ not ignored?): $(git status --short)"

if [[ $failures -gt 0 ]]; then
	echo "$failures failure(s)" >&2
	exit 1
fi
echo "PASS"
