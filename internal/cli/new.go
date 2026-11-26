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
	cmd.AddCommand(newNewLog(env))
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

func newNewLog(env Env) *cobra.Command {
	var slug, date string
	var created, modified, notes []string
	cmd := &cobra.Command{
		Use:   "log <operation> <title>",
		Short: "Write a log entry",
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
			resolve := func(args []string) ([]string, error) {
				var out []string
				for _, a := range args {
					p, err := b.Rel(env.Dir, a)
					if err != nil {
						return nil, err
					}
					out = append(out, p)
				}
				return out, nil
			}
			c, err := resolve(created)
			if err != nil {
				return err
			}
			m, err := resolve(modified)
			if err != nil {
				return err
			}
			p, err := write.NewLog(b, write.LogOpts{Operation: args[0], Title: args[1], Slug: slug, Created: c, Modified: m, Notes: notes, Date: day})
			if err != nil {
				return err
			}
			if b.Cfg.Log.Mode == "file" {
				fmt.Fprintf(env.Stdout, "appended to %s\n", p)
			} else {
				fmt.Fprintf(env.Stdout, "created %s\n", p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "slug used in the file name (default: from the title)")
	cmd.Flags().StringVar(&date, "date", "", "date YYYY-MM-DD (default: today)")
	cmd.Flags().StringArrayVar(&created, "created", nil, "path of a created file")
	cmd.Flags().StringArrayVar(&modified, "modified", nil, "path of a modified file")
	cmd.Flags().StringArrayVar(&notes, "note", nil, "free note")
	return cmd
}
