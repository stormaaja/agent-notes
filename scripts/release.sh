#!/usr/bin/env bash
# Builds agent-notes and optionally cuts a release. See usage below.
set -euo pipefail

# Go package variable that holds the version; set with -ldflags "-X $version_var=X.Y.Z".
version_var=github.com/stormaaja/agent-notes/internal/version.version

usage() {
	cat <<'EOF'
Usage: scripts/release.sh [build|minor|major|auto]

Without an argument, builds bin/agent-notes at the version in VERSION.

With a bump type, cuts a release:
  build   patch release, X.Y.Z -> X.Y.(Z+1)
  minor   X.Y.Z -> X.(Y+1).0
  major   X.Y.Z -> (X+1).0.0
  auto    from the Conventional Commits since the last vX.Y.Z tag: major if
          any is a breaking change, else minor if any is a feat, else patch

The working tree must be clean (no staged, unstaged or untracked changes) and
there must be releasable commits (feat, fix, perf or breaking changes) since the
last vX.Y.Z tag. Runs go vet and go test, writes VERSION, builds bin/agent-notes,
adds "## [X.Y.Z] - <today>" generated from the commits to CHANGELOG.md, commits
VERSION and CHANGELOG.md as "chore(release): vX.Y.Z" and creates the annotated
tag vX.Y.Z. Never pushes.
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

build() {
	local version=$1
	mkdir -p bin
	go build -ldflags "-X $version_var=$version" -o bin/agent-notes ./cmd/agent-notes
}

# restore_changelog puts CHANGELOG.md back as committed (or removes it if it is new).
restore_changelog() {
	if git ls-files --error-unmatch CHANGELOG.md >/dev/null 2>&1; then
		git checkout -- CHANGELOG.md
	else
		rm -f CHANGELOG.md
	fi
}

bump=""
case $# in
0) ;;
1)
	case $1 in
	-h | --help)
		usage
		exit 0
		;;
	build | minor | major | auto) bump=$1 ;;
	*) usage_error "unknown argument '$1'" ;;
	esac
	;;
*) usage_error "expected at most one argument" ;;
esac

cd "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"

if [[ -z $bump ]]; then
	[[ -f VERSION ]] || die "VERSION file not found"
	current=$(<VERSION)
	build "$current"
	echo "Built bin/agent-notes $current"
	exit 0
fi

if [[ -n $(git status --porcelain) ]]; then
	echo "error: the working tree has uncommitted changes; commit or stash them first:" >&2
	git status --short >&2
	exit 1
fi

[[ -f VERSION ]] || die "VERSION file not found"
current=$(<VERSION)
if [[ ! $current =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
	die "VERSION must contain X.Y.Z, got '$current'"
fi
major=${BASH_REMATCH[1]}
minor=${BASH_REMATCH[2]}
patch=${BASH_REMATCH[3]}
# changelog.sh bump-type fails (with a message) when there are no releasable commits since
# the last tag; that is checked before the slow vet and test steps.
checked=""
if [[ $bump == auto ]]; then
	checked=1
	type=$(scripts/changelog.sh bump-type) || die "nothing to release"
	case $type in
	major | minor) bump=$type ;;
	patch) bump=build ;;
	*) die "unexpected bump type '$type'" ;;
	esac
	echo "Commits since the last release call for a $type release"
fi
case $bump in
build) new="$major.$minor.$((patch + 1))" ;;
minor) new="$major.$((minor + 1)).0" ;;
major) new="$((major + 1)).0.0" ;;
esac
tag="v$new"

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
	die "tag $tag already exists"
fi

if [[ -z $checked ]]; then
	scripts/changelog.sh bump-type >/dev/null || die "nothing to release"
fi

echo "Releasing $tag (was $current)"
echo "Running go vet ./..."
go vet ./... || die "go vet failed; nothing was changed"
echo "Running go test ./..."
go test ./... || die "go test failed; nothing was changed"

printf '%s\n' "$new" >VERSION
if ! build "$new"; then
	git checkout -- VERSION
	die "build failed; VERSION restored"
fi
if ! scripts/changelog.sh release "$new" "$(date +%F)"; then
	git checkout -- VERSION
	restore_changelog
	die "updating CHANGELOG.md failed; VERSION and CHANGELOG.md restored"
fi
git add VERSION CHANGELOG.md
if ! git commit -q -m "chore(release): $tag"; then
	git reset -q -- VERSION CHANGELOG.md
	git checkout -- VERSION
	restore_changelog
	die "committing the release failed; VERSION and CHANGELOG.md restored"
fi
git tag -a "$tag" -m "Release $tag"

echo "Release $tag created with these changelog notes:"
echo
scripts/changelog.sh extract "$new"
echo
echo "Push with: git push --follow-tags"
