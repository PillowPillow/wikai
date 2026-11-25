package cli

import (
	"fmt"

	"github.com/PillowPillow/wikai/internal/write"
	"github.com/spf13/cobra"
)

func newMove(env Env) *cobra.Command {
	var date string
	var noLog bool
	var notes []string
	cmd := &cobra.Command{
		Use:   "move <old> <new>",
		Short: "Rename or archive a page and rewrite every link to it",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			day := env.Now()
			if date != "" {
				var err error
				if day, err = parseDate(date); err != nil {
					return err
				}
			}
			b, err := loadBase(env, cmd)
			if err != nil {
				return err
			}
			old, err := b.Rel(env.Dir, args[0])
			if err != nil {
				return err
			}
			dest, err := b.Rel(env.Dir, args[1])
			if err != nil {
				return err
			}
			res, err := write.Move(b, write.MoveOpts{Old: old, New: dest, NoLog: noLog, Notes: notes, Date: day})
			if err != nil {
				return err
			}
			line := fmt.Sprintf("moved %s to %s; links rewritten in %d page(s)", old, dest, len(res.Modified))
			if b.Cfg.Index != nil {
				line += "; index updated"
			}
			fmt.Fprintln(env.Stdout, line)
			switch {
			case res.Log == "":
			case b.Cfg.Log.Mode == "file":
				fmt.Fprintf(env.Stdout, "appended to %s\n", res.Log)
			default:
				fmt.Fprintf(env.Stdout, "created %s\n", res.Log)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&noLog, "no-log", false, "write no log entry")
	cmd.Flags().StringArrayVar(&notes, "note", nil, "free note for the log entry")
	cmd.Flags().StringVar(&date, "date", "", "date YYYY-MM-DD of the log entry (default: today)")
	return cmd
}
