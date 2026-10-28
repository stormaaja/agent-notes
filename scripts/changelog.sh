#!/usr/bin/env bash
# Generates CHANGELOG.md (Keep a Changelog format) from Conventional Commits and reads
# release notes from it. See usage below.
set -euo pipefail

repo_url=https://github.com/stormaaja/agent-notes

usage() {
	cat <<'EOF'
Usage: scripts/changelog.sh COMMAND [ARGS]

Commands:
  generate [FROM] [TO]      print the changelog body for the commits in FROM..TO
  bump-type [FROM] [TO]     print major, minor or patch for the commits in FROM..TO
                            (exit 1 if none of them is releasable)
  release X.Y.Z YYYY-MM-DD [FROM] [TO]
                            add "## [X.Y.Z] - YYYY-MM-DD" with the generated body
                            above the existing versions (exit 1 if nothing is
                            releasable or the version already exists)
  extract X.Y.Z             print the body of the "## [X.Y.Z]" section
                            (exit 1 if the section is missing or empty)
  notes X.Y.Z               print the GitHub release notes for tag vX.Y.Z: the
                            version's section and a "Full changelog" link

FROM defaults to the latest vX.Y.Z tag reachable from TO (all history if there is
none) and TO to HEAD.

Commit subjects must follow Conventional Commits, "type(scope)!: description":
  feat -> Added, fix -> Fixed, perf -> Changed; "!" after the type or scope, or a
  "BREAKING CHANGE:" / "BREAKING-CHANGE:" footer, -> Breaking changes (any type).
Other types (docs, test, chore, ci, build, style, refactor, revert, ...), merge
commits and subjects that are not Conventional Commits are left out.
Breaking changes make a major bump, features a minor one, fixes and perf a patch.

Works on CHANGELOG.md at the repository root, or on $CHANGELOG if set.
EOF
}

die() {
	echo "error: $*" >&2
	exit 1
}

usage_error() {
	echo "error: $*" >&2
	usage >&2
	exit 2
}

# Written when release creates CHANGELOG.md; keep in sync with CHANGELOG.md.
header='# Changelog

All notable changes to this project are documented in this file. It is generated from
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) by `scripts/release.sh`;
do not edit it by hand.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).'

git_() {
	git -C "$root" "$@"
}

# section NAME prints the lines of the "## [NAME]..." section, without its heading and
# with surrounding blank lines trimmed. The section ends at the next "## " heading or at
# a link reference definition ("[x]: url"). Exits 1 if there is no such section.
section() {
	awk -v want="$1" '
		/^## / || /^\[[^]]+\]: / {
			if (inside) exit
			if (substr($0, 1, 4) == "## [") {
				name = substr($0, 5)
				sub(/\].*/, "", name)
				if (name == want) { inside = 1; found = 1; next }
			}
		}
		inside { lines[++n] = $0 }
		END {
			if (!found) exit 1
			first = 1
			while (first <= n && lines[first] ~ /^[ \t]*$/) first++
			while (n >= first && lines[n] ~ /^[ \t]*$/) n--
			for (i = first; i <= n; i++) print lines[i]
		}
	' "$changelog"
}

# Entries of the classified commits, filled by collect.
breaking=()
added=()
fixed=()
changed=()
range_desc=""

# collect [FROM] [TO] classifies the non-merge commits in FROM..TO, oldest first, into
# the entry arrays above.
collect() {
	local from=${1:-} to=${2:-HEAD} range
	git_ rev-parse -q --verify "$to^{commit}" >/dev/null || die "unknown revision '$to'"
	if [[ -z $from ]]; then
		from=$(git_ describe --tags --abbrev=0 --match 'v[0-9]*' "$to" 2>/dev/null) || from=""
	else
		git_ rev-parse -q --verify "$from^{commit}" >/dev/null || die "unknown revision '$from'"
	fi
	if [[ -n $from ]]; then
		range=("$from..$to")
		range_desc="$from..$to"
	else
		range=("$to")
		range_desc="the history of $to"
	fi

	local re='^([A-Za-z]+)(\(([^()]+)\))?(!?): (.+)$'
	local record sha subject body type scope bang desc text item
	while IFS= read -r -d $'\x1e' record; do
		record=${record#$'\n'}
		sha=${record%%$'\x1f'*}
		record=${record#*$'\x1f'}
		subject=${record%%$'\x1f'*}
		body=${record#*$'\x1f'}
		[[ $subject =~ $re ]] || continue
		type=$(tr '[:upper:]' '[:lower:]' <<<"${BASH_REMATCH[1]}")
		scope=${BASH_REMATCH[3]}
		bang=${BASH_REMATCH[4]}
		desc=${BASH_REMATCH[5]}
		text="$(tr '[:lower:]' '[:upper:]' <<<"${desc:0:1}")${desc:1}"
		[[ -n $scope ]] && text="**$scope:** $text"
		item="- $text ([${sha:0:7}]($repo_url/commit/$sha))"
		if [[ -n $bang ]] || grep -qE '^BREAKING[ -]CHANGE:' <<<"$body"; then
			breaking+=("$item")
			continue
		fi
		case $type in
		feat) added+=("$item") ;;
		fix) fixed+=("$item") ;;
		perf) changed+=("$item") ;;
		esac
	done < <(git_ log --reverse --no-merges --format='%H%x1f%s%x1f%b%x1e' "${range[@]}" --)
}

releasable() {
	((${#breaking[@]} + ${#added[@]} + ${#fixed[@]} + ${#changed[@]} > 0))
}

# print_section TITLE ENTRY... prints a "### TITLE" subsection, preceded by a blank line
# unless it is the first one.
printed=0
print_section() {
	local title=$1
	shift
	[[ $# -gt 0 ]] || return 0
	[[ $printed -eq 0 ]] || echo
	printed=1
	printf '### %s\n\n' "$title"
	printf '%s\n' "$@"
}

# print_body prints the subsections for the collected entries.
print_body() {
	print_section "Breaking changes" ${breaking[@]+"${breaking[@]}"}
	print_section "Added" ${added[@]+"${added[@]}"}
	print_section "Fixed" ${fixed[@]+"${fixed[@]}"}
	print_section "Changed" ${changed[@]+"${changed[@]}"}
}

cmd_generate() {
	collect "$@"
	print_body
}

no_releasable() {
	die "no releasable commits (feat, fix, perf or breaking changes) in $range_desc"
}

cmd_bump_type() {
	collect "$@"
	if ((${#breaking[@]} > 0)); then
		echo major
	elif ((${#added[@]} > 0)); then
		echo minor
	elif releasable; then
		echo patch
	else
		no_releasable
	fi
}

cmd_release() {
	local version=$1 date=$2
	shift 2
	[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || usage_error "version must be X.Y.Z, got '$version'"
	[[ $date =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || usage_error "date must be YYYY-MM-DD, got '$date'"
	if [[ -f $changelog ]] && section "$version" >/dev/null; then
		die "$changelog already has a '## [$version]' section"
	fi
	collect "$@"
	releasable || no_releasable
	local body
	body=$(print_body)

	local current="$header" tmp
	[[ -f $changelog ]] && current=$(<"$changelog")
	tmp=$(mktemp "$changelog.XXXXXX")
	# Insert the new section before the first version heading (or link reference
	# definition), or at the end, with one blank line around it.
	if ! printf '%s\n' "$current" | HEADING="## [$version] - $date" BODY="$body" awk '
		function insert() {
			while (n > 0 && pre[n] ~ /^[ \t]*$/) n--
			for (i = 1; i <= n; i++) print pre[i]
			print ""
			print ENVIRON["HEADING"]
			print ""
			print ENVIRON["BODY"]
			done = 1
		}
		!done && (/^## / || /^\[[^]]+\]: /) { insert(); print ""; print; next }
		done { print; next }
		{ pre[++n] = $0 }
		END { if (!done) insert() }
	' >"$tmp"; then
		rm -f "$tmp"
		die "failed to update $changelog"
	fi
	# Overwrite in place (rather than mv) to keep the file's permissions.
	cat "$tmp" >"$changelog"
	rm -f "$tmp"
}

cmd_extract() {
	local version=$1 body
	[[ -f $changelog ]] || die "$changelog not found"
	body=$(section "$version") || die "no '## [$version]' section in $changelog"
	[[ -n $body ]] || die "the '## [$version]' section in $changelog is empty"
	printf '%s\n' "$body"
}

cmd_notes() {
	local version=$1 tag="v$1" prev body
	git_ rev-parse -q --verify "refs/tags/$tag" >/dev/null || die "tag $tag not found"
	body=$(cmd_extract "$version")
	printf '%s\n\n' "$body"
	if prev=$(git_ describe --tags --abbrev=0 --match 'v[0-9]*' "$tag^" 2>/dev/null); then
		echo "**Full changelog**: $repo_url/compare/$prev...$tag"
	else
		echo "**Full changelog**: $repo_url/commits/$tag"
	fi
}

if [[ $# -eq 0 ]]; then
	usage_error "missing command"
fi
command=$1
shift
case $command in
-h | --help)
	usage
	exit 0
	;;
generate) [[ $# -le 2 ]] || usage_error "generate takes at most two arguments: [FROM] [TO]" ;;
bump-type) [[ $# -le 2 ]] || usage_error "bump-type takes at most two arguments: [FROM] [TO]" ;;
release)
	[[ $# -ge 2 && $# -le 4 ]] || usage_error "release takes two to four arguments: X.Y.Z YYYY-MM-DD [FROM] [TO]"
	;;
extract) [[ $# -eq 1 ]] || usage_error "extract takes one argument: X.Y.Z" ;;
notes) [[ $# -eq 1 ]] || usage_error "notes takes one argument: X.Y.Z" ;;
*) usage_error "unknown command '$command'" ;;
esac

root=$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
changelog=${CHANGELOG:-$root/CHANGELOG.md}

case $command in
generate) cmd_generate "$@" ;;
bump-type) cmd_bump_type "$@" ;;
release) cmd_release "$@" ;;
extract) cmd_extract "$@" ;;
notes) cmd_notes "$@" ;;
esac
