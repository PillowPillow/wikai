package main

import (
	"fmt"
	"os"
	"time"

	"github.com/PillowPillow/wikai/internal/cli"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(2)
	}
	env := cli.Env{
		Dir:    dir,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getenv: os.Getenv,
		Now:    time.Now,
	}
	os.Exit(cli.Execute(os.Args[1:], env))
}
