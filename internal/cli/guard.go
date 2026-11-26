package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/PillowPillow/wikai/internal/config"
	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/guard"
	"github.com/spf13/cobra"
)

func newGuard(env Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "Refuse agent writes and commits that break the wiki rules",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(newGuardHook(env), newGuardCommit(env))
	return cmd
}

func newGuardHook(env Env) *cobra.Command {
	return &cobra.Command{
		Use:   "hook",
		Short: "Claude Code PreToolUse hook: deny a write to a guarded file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ev, err := readEvent(env.Stdin)
			if err != nil {
				return err
			}
			start := env.Getenv("CLAUDE_PROJECT_DIR")
			if start == "" {
				start = ev.Cwd
			}
			if start == "" {
				start = env.Dir
			}
			if ev.Cwd == "" {
				ev.Cwd = start
			}
			file, err := config.Find(start)
			if err != nil {
				return nil
			}
			cfg, err := config.Load(file)
			if err != nil {
				return err
			}
			if deny := guard.Hook(cfg, filepath.Dir(file), ev); deny != "" {
				fmt.Fprintln(env.Stderr, deny)
				return errDenied
			}
			return nil
		},
	}
}

func readEvent(r io.Reader) (guard.Event, error) {
	var ev guard.Event
	data, err := io.ReadAll(r)
	if err != nil {
		return ev, exit.Usage("reading the hook event: %v", err)
	}
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return ev, exit.Usage("the hook event is not a JSON object")
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return ev, exit.Usage("the hook event is not valid: %v", err)
	}
	return ev, nil
}

func newGuardCommit(env Env) *cobra.Command {
	var staged, agent bool
	cmd := &cobra.Command{
		Use:   "commit (--staged | <base> <head>)",
		Short: "Refuse staged or committed changes that break the wiki rules",
		Args: func(_ *cobra.Command, args []string) error {
			if staged && len(args) != 0 || !staged && len(args) != 2 {
				return exit.Usage("guard commit takes --staged or <base> <head>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, boundary, err := config.Locate(env.Dir, cmd.Flag("config").Value.String(), cmd.Flag("dir").Value.String())
			if err != nil {
				return err
			}
			var base, head string
			if !staged {
				base, head = args[0], args[1]
			}
			violations, err := guard.Commit(cfg, boundary, staged, base, head, agent)
			if err != nil {
				return err
			}
			if len(violations) == 0 {
				fmt.Fprintln(env.Stdout, "guard: OK")
				return nil
			}
			fmt.Fprintln(env.Stderr, "guard: changes refused")
			for _, v := range violations {
				fmt.Fprintf(env.Stderr, "%s %s: %s\n", v.Status, v.Path, v.Reason)
			}
			return errFindings
		},
	}
	cmd.Flags().BoolVar(&staged, "staged", false, "check the staged changes against HEAD")
	cmd.Flags().BoolVar(&agent, "agent", false, "also refuse changes to protected paths")
	return cmd
}
