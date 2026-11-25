package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/spf13/cobra"
)

var Version = "dev"

type Env struct {
	Dir            string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	Getenv         func(string) string
	Now            func() time.Time
}

func newRoot(env Env) *cobra.Command {
	root := &cobra.Command{
		Use:           "wikai",
		Short:         "Check, write and guard LLM wikis",
		Version:       Version,
		Args:          cobra.NoArgs,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.SetIn(env.Stdin)
	root.SetOut(env.Stdout)
	root.SetErr(env.Stderr)
	root.PersistentFlags().String("config", "", "path to the configuration file")
	root.PersistentFlags().String("dir", "", "wiki folder (default: current folder)")
	return root
}

func Execute(args []string, env Env) int {
	root := newRoot(env)
	root.SetArgs(args)
	err := root.Execute()
	code := exit.Code(err)
	if err != nil {
		prefix := "error"
		if code == 1 {
			prefix = "refused"
		}
		fmt.Fprintf(env.Stderr, "%s: %s\n", prefix, err)
	}
	return code
}
