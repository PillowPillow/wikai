package cli

import (
	"encoding/json"
	"fmt"

	"github.com/PillowPillow/wikai/internal/check"
	"github.com/spf13/cobra"
)

func newCheck(env Env) *cobra.Command {
	var base string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check the base and report findings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			b, err := loadBase(env, cmd)
			if err != nil {
				return err
			}
			findings, warnings, err := check.Run(b, base)
			for _, w := range warnings {
				fmt.Fprintf(env.Stderr, "warning: %s\n", w)
			}
			if err != nil {
				return err
			}
			if asJSON {
				data, err := json.MarshalIndent(findings, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintf(env.Stdout, "%s\n", data)
			} else {
				for _, f := range findings {
					fmt.Fprintf(env.Stdout, "%s: %s: %s\n", f.Check, f.Path, f.Message)
				}
				if len(findings) == 0 {
					fmt.Fprintf(env.Stdout, "OK: %d pages\n", len(b.Pages))
				} else {
					fmt.Fprintf(env.Stdout, "FAIL: %d finding(s)\n", len(findings))
				}
			}
			if len(findings) > 0 {
				return errFindings
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&base, "base", "", "git ref the frozen check compares against")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the findings as JSON")
	return cmd
}
