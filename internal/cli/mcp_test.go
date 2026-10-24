package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var mcpRequests = []string{
	`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
	`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
	`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"notes_add","arguments":{"title":"From MCP","tags":["mcp"]}}}`,
}

type rpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// TestMCP drives the mcp command like a client: requests on stdin, one
// JSON-RPC message per stdout line, then stdin is closed.
func TestMCP(t *testing.T) {
	e := newEnv(t)
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	var stderr bytes.Buffer
	done := make(chan int)
	go func() {
		code := Run([]string{"mcp"}, Env{Stdin: stdinR, Stdout: stdoutW, Stderr: &stderr, Cwd: e.cwd, Getenv: func(string) string { return "" }})
		stdoutW.Close()
		done <- code
	}()

	lines := bufio.NewScanner(stdoutR)
	lines.Buffer(nil, 1<<20)
	responses := map[int]rpcResponse{}
	for _, req := range mcpRequests {
		if _, err := io.WriteString(stdinW, req+"\n"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(req, `"id"`) {
			continue
		}
		if !lines.Scan() {
			t.Fatalf("no response to %s: %v", req, lines.Err())
		}
		var r rpcResponse
		if err := json.Unmarshal(lines.Bytes(), &r); err != nil {
			t.Fatalf("stdout line is not JSON-RPC: %q", lines.Text())
		}
		responses[r.ID] = r
	}
	stdinW.Close()
	if code := <-done; code != 0 {
		t.Fatalf("mcp: exit %d, stderr %q", code, stderr.String())
	}
	if lines.Scan() {
		t.Errorf("unexpected stdout after responses: %q", lines.Text())
	}

	for id := 1; id <= 3; id++ {
		if r, ok := responses[id]; !ok || r.Error != nil {
			t.Errorf("response %d = %+v, want a result", id, r)
		}
	}
	if !strings.Contains(string(responses[2].Result), `"notes_search"`) {
		t.Errorf("tools/list = %s", responses[2].Result)
	}
	if _, err := os.Stat(filepath.Join(e.notesDir(), "from-mcp.md")); err != nil {
		t.Errorf("notes_add did not write to the resolved notes dir: %v", err)
	}
}

func TestMCPErrors(t *testing.T) {
	e := newEnv(t)
	args := []string{"mcp", "extra"}
	_, errOut, code := e.run(args...)
	assertCode(t, args, code, 2, errOut)
	if out := e.mustRun("help"); !strings.Contains(out, "mcp") {
		t.Errorf("help does not list mcp:\n%s", out)
	}
	if out := e.mustRun("help", "mcp"); !strings.Contains(out, "claude mcp add") {
		t.Errorf("mcp help does not explain setup:\n%s", out)
	}
}

func TestMCPCancel(t *testing.T) {
	e := newEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	stdinR, stdinW := io.Pipe()
	defer stdinW.Close()
	var stderr bytes.Buffer
	done := make(chan int)
	go func() {
		done <- Run([]string{"mcp"}, Env{Context: ctx, Stdin: stdinR, Stdout: io.Discard, Stderr: &stderr, Cwd: e.cwd, Getenv: func(string) string { return "" }})
	}()
	cancel()
	if code := <-done; code != 0 {
		t.Errorf("cancelled mcp: exit %d, stderr %q", code, stderr.String())
	}
}
