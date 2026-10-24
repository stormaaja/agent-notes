// Command agent-notes manages knowledge notes stored as Markdown in a repository.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/stormaaja/agent-notes/internal/cli"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Cwd:    cwd,
		Getenv: os.Getenv,
		Now:    time.Now,
	}))
}
