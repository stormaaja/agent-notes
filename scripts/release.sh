#!/usr/bin/env bash
# Builds agent-notes and optionally cuts a release. See usage below.
set -euo pipefail

# Go package variable that holds the version; set with -ldflags "-X $version_var=X.Y.Z".
version_var=github.com/stormaaja/agent-notes/internal/version.version

usage() {
	cat <<'EOF'
Usage: scripts/release.sh [build|minor|major]

Without an argument, builds bin/agent-notes at the version in VERSION.

With a bump type, cuts a release:
  build   patch release, X.Y.Z -> X.Y.(Z+1)
  minor   X.Y.Z -> X.(Y+1).0
  major   X.Y.Z -> (X+1).0.0

The working tree must be clean (no staged, unstaged or untracked changes) and
CHANGELOG.md must have release notes under "## [Unreleased]". Runs go vet and
go test, writes VERSION, builds bin/agent-notes, turns the Unreleased notes into
"## [X.Y.Z] - <today>", commits VERSION and CHANGELOG.md as "Release vX.Y.Z"
and creates the annotated tag vX.Y.Z. Never pushes.
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

bump=""
case $# in
0) ;;
1)
	case $1 in
	-h | --help)
		usage
		exit 0
		;;
	build | minor | major) bump=$1 ;;
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
case $bump in
build) new="$major.$minor.$((patch + 1))" ;;
minor) new="$major.$((minor + 1)).0" ;;
major) new="$((major + 1)).0.0" ;;
esac
tag="v$new"

if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
	die "tag $tag already exists"
fi

if ! scripts/changelog.sh has-unreleased; then
	die "add release notes under ## [Unreleased] in CHANGELOG.md first"
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
	git checkout -- VERSION CHANGELOG.md
	die "updating CHANGELOG.md failed; VERSION and CHANGELOG.md restored"
fi
git add VERSION CHANGELOG.md
git commit -q -m "Release $tag"
git tag -a "$tag" -m "Release $tag"

echo "Release $tag created. Push with: git push --follow-tags"
