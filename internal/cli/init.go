package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/preset"
	"github.com/spf13/cobra"
)

func newInit(env Env) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "init [dir] --preset " + strings.Join(preset.Names(), "|"),
		Short: "Write a base from a preset",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return exit.Usage("--preset is required: one of %s", strings.Join(preset.Names(), ", "))
			}
			dir := env.Dir
			if len(args) == 1 {
				dir = args[0]
				if !filepath.IsAbs(dir) {
					dir = filepath.Join(env.Dir, dir)
				}
			}
			created, err := preset.Write(name, dir, env.Now())
			for _, p := range created {
				fmt.Fprintf(env.Stdout, "created %s\n", p)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&name, "preset", "", "preset to write: "+strings.Join(preset.Names(), " or "))
	return cmd
}
