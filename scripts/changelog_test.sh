#!/usr/bin/env bash
# Tests scripts/changelog.sh against fixture changelogs in a temporary directory.
set -euo pipefail

script="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/changelog.sh"
work=$(mktemp -d "${TMPDIR:-/tmp}/changelog-test.XXXXXX")
trap 'rm -rf "$work"' EXIT
export CHANGELOG="$work/CHANGELOG.md"

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
	out=$(cd "$work" && "$script" "$@" 2>"$work/stderr") || code=$?
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
	got=$(<"$CHANGELOG")
	if [[ $got == "$want" ]]; then
		pass "$what: CHANGELOG.md"
	else
		fail "$what: CHANGELOG.md ="$'\n'"$got"$'\n'"want:"$'\n'"$want"
	fi
}

header='# Changelog

All notable changes to this project are documented in this file.'

full="$header

## [Unreleased]

### Added

- Upcoming feature.

### Fixed

## [1.1.0] - 2026-02-01

### Added

- Feature one.
- Feature two.

### Fixed

- Bug fix.

## [1.0.0] - 2026-01-01

### Added

- First release.

[1.0.0]: https://example.invalid/releases/tag/v1.0.0"

empty_unreleased="$header

## [Unreleased]

### Added

### Fixed

## [1.0.0] - 2026-01-01

- First release."

# --- usage ---
run
expect_code 2 "no args"
expect_err "Usage:" "no args"
run bogus
expect_code 2 "unknown command"
expect_err "Usage:" "unknown command"
run extract
expect_code 2 "extract without version"
run release 1.2.0
expect_code 2 "release without date"
run -h
expect_code 0 "-h"
[[ $out == *"Usage:"* ]] && pass "-h: usage on stdout" || fail "-h: stdout = $out"

# --- missing file ---
rm -f "$CHANGELOG"
run extract 1.0.0
expect_code 1 "missing file"
expect_err "not found" "missing file"

# --- extract ---
printf '%s\n' "$full" >"$CHANGELOG"
run extract 1.1.0
expect_code 0 "extract middle"
expect_out "### Added

- Feature one.
- Feature two.

### Fixed

- Bug fix." "extract middle"

run extract 1.0.0
expect_code 0 "extract last before link footer"
expect_out "### Added

- First release." "extract last before link footer"

run extract 1.0
expect_code 1 "extract prefix of a version"
expect_err "1.0" "extract prefix of a version"


run extract 9.9.9
expect_code 1 "extract missing"
expect_err "9.9.9" "extract missing"

printf '%s\n' "$header" "" "## [Unreleased]" "" "## [2.0.0] - 2026-03-01" "" "" "## [1.0.0] - 2026-01-01" "" "- x" >"$CHANGELOG"
run extract 2.0.0
expect_code 1 "extract empty"
expect_err "2.0.0" "extract empty"

printf '%s\n' "$header" "" "## [1.0.0] - 2026-01-01" "" "- Last section, no footer." "" >"$CHANGELOG"
run extract 1.0.0
expect_code 0 "extract at end of file"
expect_out "- Last section, no footer." "extract at end of file"

# --- has-unreleased ---
printf '%s\n' "$full" >"$CHANGELOG"
run has-unreleased
expect_code 0 "has-unreleased with notes"

printf '%s\n' "$empty_unreleased" >"$CHANGELOG"
run has-unreleased
expect_code 1 "has-unreleased with only empty subheadings"
expect_err "Unreleased" "has-unreleased with only empty subheadings"

printf '%s\n' "$header" "" "## [Unreleased]" >"$CHANGELOG"
run has-unreleased
expect_code 1 "has-unreleased with bare heading at end of file"

printf '%s\n' "$header" "" "## [1.0.0] - 2026-01-01" "" "- x" >"$CHANGELOG"
run has-unreleased
expect_code 1 "has-unreleased without section"
expect_err "Unreleased" "has-unreleased without section"

# --- release ---
printf '%s\n' "$full" >"$CHANGELOG"
run release 1.2.0 2026-03-15
expect_code 0 "release"
expect_file "$header

## [Unreleased]

## [1.2.0] - 2026-03-15

### Added

- Upcoming feature.

## [1.1.0] - 2026-02-01

### Added

- Feature one.
- Feature two.

### Fixed

- Bug fix.

## [1.0.0] - 2026-01-01

### Added

- First release.

[1.0.0]: https://example.invalid/releases/tag/v1.0.0" "release drops empty subheadings, adds fresh Unreleased"
run extract 1.2.0
expect_code 0 "extract after release"
expect_out "### Added

- Upcoming feature." "extract after release"
run has-unreleased
expect_code 1 "Unreleased empty after release"

printf '%s\n' "$full" >"$CHANGELOG"
run release 1.1.0 2026-03-15
expect_code 1 "release existing version"
expect_err "1.1.0" "release existing version"
expect_file "$full" "release existing version leaves file"

printf '%s\n' "$empty_unreleased" >"$CHANGELOG"
run release 1.1.0 2026-03-15
expect_code 1 "release with empty Unreleased"
expect_err "Unreleased" "release with empty Unreleased"
expect_file "$empty_unreleased" "release with empty Unreleased leaves file"

printf '%s\n' "$header" "" "## [1.0.0] - 2026-01-01" "" "- x" >"$CHANGELOG"
run release 1.1.0 2026-03-15
expect_code 1 "release without Unreleased"
expect_err "Unreleased" "release without Unreleased"

printf '%s\n' "$full" >"$CHANGELOG"
run release 1.2 2026-03-15
expect_code 2 "release bad version"
run release 1.2.0 15.3.2026
expect_code 2 "release bad date"
expect_file "$full" "release bad args leave file"

# Unreleased as the last section, no subheadings.
printf '%s\n' "$header" "" "## [Unreleased]" "" "- Only change." >"$CHANGELOG"
run release 0.1.0 2026-04-01
expect_code 0 "release last section"
expect_file "$header

## [Unreleased]

## [0.1.0] - 2026-04-01

- Only change." "release last section"

if [[ $failures -gt 0 ]]; then
	echo "$failures failure(s)" >&2
	exit 1
fi
echo "PASS"
