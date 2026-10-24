package cli

import (
	"fmt"
	"os"
)

var initCmd = command{
	name:    "init",
	summary: "create the notes directory and print its path",
	run:     runInit,
}

func runInit(r *runner, args []string) error {
	fs := r.flagSet("init")
	if _, err := r.parseN(fs, args); err != nil {
		return err
	}
	dir, err := r.notesDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if r.json {
		return r.printJSON(struct {
			Dir string `json:"dir"`
		}{dir})
	}
	_, err = fmt.Fprintln(r.env.Stdout, dir)
	return err
}
