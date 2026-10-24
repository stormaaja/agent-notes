// Command agent-notes manages knowledge notes stored as Markdown in a repository.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stormaaja/agent-notes/internal/cli"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cli.Run(os.Args[1:], cli.Env{
		Context: ctx,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Cwd:     cwd,
		Getenv:  os.Getenv,
		Now:     time.Now,
	})
	stop()
	os.Exit(code)
}
