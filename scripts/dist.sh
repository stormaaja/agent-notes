#!/usr/bin/env bash
# Builds release archives of agent-notes for all supported platforms into dist/.
# Used by .github/workflows/release.yml; can be run locally to check the output.
set -euo pipefail

# Go package variable that holds the version; set with -ldflags "-X $version_var=X.Y.Z".
version_var=github.com/stormaaja/agent-notes/internal/version.version
platforms=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64)
# Files packed next to the binary, relative to the repository root; docs/*.md is added below.
docs=(README.md LICENSE CHANGELOG.md)

usage() {
	cat <<'EOF'
Usage: scripts/dist.sh VERSION

Cross-builds agent-notes VERSION (X.Y.Z, without the "v") for linux, darwin and
windows on amd64 and arm64, and writes to dist/ (replacing it):

  agent-notes_VERSION_<os>_<arch>.tar.gz   (.zip on windows) with the binary,
                                           README.md, LICENSE, CHANGELOG.md and
                                           docs/*.md
  checksums.txt                            SHA-256 sums of the archives
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

case ${1:-} in
-h | --help)
	usage
	exit 0
	;;
esac
if [[ $# -ne 1 ]]; then
	usage >&2
	exit 2
fi
version=$1
[[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$ ]] || usage_error "version must be X.Y.Z, got '$version'"

cd "$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
shopt -s nullglob
pages=(docs/*.md)
shopt -u nullglob
[[ ${#pages[@]} -gt 0 ]] || die "docs/*.md not found"
docs+=("${pages[@]}")
for doc in "${docs[@]}"; do
	[[ -f $doc ]] || die "$doc not found"
done

rm -rf dist
mkdir -p dist
for platform in "${platforms[@]}"; do
	os=${platform%/*}
	arch=${platform#*/}
	name="agent-notes_${version}_${os}_${arch}"
	binary=agent-notes
	[[ $os == windows ]] && binary=agent-notes.exe
	stage="dist/$name"

	echo "Building $name"
	mkdir -p "$stage"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X $version_var=$version" -o "$stage/$binary" ./cmd/agent-notes
	for doc in "${docs[@]}"; do
		mkdir -p "$stage/$(dirname "$doc")"
		cp "$doc" "$stage/$doc"
	done

	if [[ $os == windows ]]; then
		(cd "$stage" && zip -q -X "../$name.zip" "$binary" "${docs[@]}")
	else
		tar -czf "dist/$name.tar.gz" -C "$stage" "$binary" "${docs[@]}"
	fi
	rm -rf "$stage"
done

(
	cd dist
	if command -v sha256sum >/dev/null; then
		sha256sum -- *.tar.gz *.zip
	else
		shasum -a 256 -- *.tar.gz *.zip
	fi
) >dist/checksums.txt

echo "Wrote dist/:"
ls -1 dist
