package version

import (
	"runtime/debug"
	"testing"
)

func buildInfo(v string, ok bool) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) {
		if !ok {
			return nil, false
		}
		return &debug.BuildInfo{Main: debug.Module{Version: v}}, true
	}
}

func TestResolve(t *testing.T) {
	for _, tc := range []struct {
		name     string
		injected string
		info     func() (*debug.BuildInfo, bool)
		want     string
	}{
		{"ldflags wins", "1.2.3", buildInfo("v0.9.0", true), "1.2.3"},
		{"module version", "", buildInfo("v0.9.0", true), "0.9.0"},
		{"pseudo-version", "", buildInfo("v0.0.0-20261028120000-abcdef123456", true), "0.0.0-20261028120000-abcdef123456"},
		{"devel module", "", buildInfo("(devel)", true), "devel"},
		{"empty module", "", buildInfo("", true), "devel"},
		{"no build info", "", buildInfo("", false), "devel"},
	} {
		if got := Resolve(tc.injected, tc.info); got != tc.want {
			t.Errorf("%s: Resolve = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestStringUsesInjectedVersion(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	version = "4.5.6"
	if got := String(); got != "4.5.6" {
		t.Errorf("String() = %q, want 4.5.6", got)
	}
}
