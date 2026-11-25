package cli

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/write"
	"github.com/spf13/cobra"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if !datePattern.MatchString(s) || err != nil {
		return time.Time{}, exit.Usage("--date must be YYYY-MM-DD, found %q", s)
	}
	return t, nil
}

func newNew(env Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a page or a log entry",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(newNewPage(env))
	return cmd
}

func newNewPage(env Env) *cobra.Command {
	var summary, slug, date string
	var sets []string
	cmd := &cobra.Command{
		Use:   "page <kind> <title>",
		Short: "Create a page from its skeleton and add its index line",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			day := env.Now()
			if date != "" {
				var err error
				if day, err = parseDate(date); err != nil {
					return err
				}
			}
			set := map[string]string{}
			for _, kv := range sets {
				k, v, ok := strings.Cut(kv, "=")
				if !ok || k == "" {
					return exit.Usage("--set expects key=value, found %q", kv)
				}
				set[k] = v
			}
			b, err := loadBase(env, cmd)
			if err != nil {
				return err
			}
			p, err := write.NewPage(b, write.PageOpts{Kind: args[0], Title: args[1], Summary: summary, Slug: slug, Date: day, Set: set})
			if err != nil {
				return err
			}
			if write.Indexed(b, p) {
				fmt.Fprintf(env.Stdout, "created %s and its line in %s\n", p, b.IndexPath())
			} else {
				fmt.Fprintf(env.Stdout, "created %s\n", p)
			}
			fmt.Fprintln(env.Stdout, "next: fill the sections, link the page from another page, add the log entry (wikai new log)")
			return nil
		},
	}
	cmd.Flags().StringVar(&summary, "summary", "", "one-line summary for the index line")
	cmd.Flags().StringVar(&slug, "slug", "", "slug used in the file name (default: from the title)")
	cmd.Flags().StringVar(&date, "date", "", "date YYYY-MM-DD (default: today)")
	cmd.Flags().StringArrayVar(&sets, "set", nil, "fill {{key}} in the skeleton, as key=value")
	_ = cmd.MarkFlagRequired("summary")
	return cmd
}
