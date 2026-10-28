#!/usr/bin/env bash
# Reads and updates CHANGELOG.md (Keep a Changelog format). See usage below.
set -euo pipefail

usage() {
	cat <<'EOF'
Usage: scripts/changelog.sh COMMAND [ARGS]

Commands:
  extract X.Y.Z             print the body of the "## [X.Y.Z]" section
                            (exit 1 if the section is missing or empty)
  has-unreleased            exit 0 if "## [Unreleased]" has release notes,
                            1 if it is missing or has only blank lines and
                            empty "###" subheadings
  release X.Y.Z YYYY-MM-DD  turn the Unreleased notes into "## [X.Y.Z] - YYYY-MM-DD"
                            (dropping empty subheadings) under a fresh, empty
                            "## [Unreleased]"

Operates on CHANGELOG.md at the repository root, or on $CHANGELOG if set.
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

# has_notes reads section lines on stdin and succeeds if any line is neither blank nor
# a "###"-or-deeper subheading.
has_notes() {
	grep -qvE '^[[:space:]]*$|^###+ '
}

check_unreleased() {
	local body
	body=$(section Unreleased) || die "no '## [Unreleased]' section in $changelog"
	has_notes <<<"$body" || die "no release notes under '## [Unreleased]' in $changelog"
}

cmd_extract() {
	local version=$1 body
	body=$(section "$version") || die "no '## [$version]' section in $changelog"
	[[ -n $body ]] || die "the '## [$version]' section in $changelog is empty"
	printf '%s\n' "$body"
}

cmd_release() {
	local version=$1 date=$2
	[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || usage_error "version must be X.Y.Z, got '$version'"
	[[ $date =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || usage_error "date must be YYYY-MM-DD, got '$date'"
	check_unreleased
	if section "$version" >/dev/null; then
		die "$changelog already has a '## [$version]' section"
	fi

	local tmp
	tmp=$(mktemp "$changelog.XXXXXX")
	# Copy the file, replacing the Unreleased section by an empty one followed by the new
	# version section holding the Unreleased notes. Subsections ("### ...") of the notes
	# without any content are dropped.
	if ! awk -v heading="## [$version] - $date" '
		function flush_group() {
			if (!group_is_sub || group_has_notes)
				for (i = 1; i <= group_n; i++) print group[i]
			group_n = 0
			group_has_notes = 0
		}
		/^## / || /^\[[^]]+\]: / {
			if (inside) { flush_group(); inside = 0 }
			if ($0 ~ /^## \[Unreleased\]/) {
				print
				print ""
				print heading
				inside = 1
				group_is_sub = 0
				next
			}
		}
		!inside { print; next }
		/^###+ / { flush_group(); group_is_sub = 1 }
		{
			group[++group_n] = $0
			if ($0 !~ /^[ \t]*$/ && $0 !~ /^###+ /) group_has_notes = 1
		}
		END { if (inside) flush_group() }
	' "$changelog" >"$tmp"; then
		rm -f "$tmp"
		die "failed to update $changelog"
	fi
	# Overwrite in place (rather than mv) to keep the file's permissions.
	cat "$tmp" >"$changelog"
	rm -f "$tmp"
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
extract)
	[[ $# -eq 1 ]] || usage_error "extract takes one argument: X.Y.Z"
	;;
has-unreleased)
	[[ $# -eq 0 ]] || usage_error "has-unreleased takes no arguments"
	;;
release)
	[[ $# -eq 2 ]] || usage_error "release takes two arguments: X.Y.Z YYYY-MM-DD"
	;;
*) usage_error "unknown command '$command'" ;;
esac

if [[ -n ${CHANGELOG:-} ]]; then
	changelog=$CHANGELOG
else
	changelog="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)/CHANGELOG.md"
fi
[[ -f $changelog ]] || die "$changelog not found"

case $command in
extract) cmd_extract "$1" ;;
has-unreleased) check_unreleased ;;
release) cmd_release "$1" "$2" ;;
esac
