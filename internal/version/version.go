// Package version reports the agent-notes version.
package version

import (
	"runtime/debug"
	"strings"
)

// version is set at build time with
//
//	-ldflags "-X github.com/stormaaja/agent-notes/internal/version.version=1.2.3"
var version = ""

// String returns the version: the value set with -ldflags, else the main
// module version recorded by the go command (go install …@vX.Y.Z) without
// its "v" prefix, else "devel". Both forms match the VERSION file.
func String() string { return Resolve(version, debug.ReadBuildInfo) }

// Resolve implements String with the injected version and build info source.
func Resolve(injected string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if injected != "" {
		return injected
	}
	if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "devel"
}
