package cli

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/PillowPillow/wikai/internal/exit"
	"github.com/PillowPillow/wikai/internal/manifest"
	"github.com/PillowPillow/wikai/internal/wiki"
	"github.com/spf13/cobra"
)

type manifestTarget struct {
	b    *wiki.Base
	file string
	raw  string
	disk string
}

func manifestSetup(env Env, cmd *cobra.Command, arg string) (manifestTarget, error) {
	b, err := loadBase(env, cmd)
	if err != nil {
		return manifestTarget{}, err
	}
	if b.Cfg.Manifest == nil {
		return manifestTarget{}, exit.Usage("the configuration has no manifest block")
	}
	raw, err := b.Rel(env.Dir, arg)
	if err != nil {
		return manifestTarget{}, err
	}
	return manifestTarget{
		b:    b,
		file: filepath.Join(b.Boundary, filepath.FromSlash(b.Cfg.RootPath(b.Cfg.Manifest.File))),
		raw:  raw,
		disk: filepath.Join(b.Boundary, filepath.FromSlash(raw)),
	}, nil
}

func newManifest(env Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Hash, look up and record source notes in the manifest",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "hash <note>",
		Short: "Print the normalized digest of a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := manifestSetup(env, cmd, args[0])
			if err != nil {
				return err
			}
			digest, err := manifest.Hash(t.disk)
			if err != nil {
				return err
			}
			fmt.Fprintln(env.Stdout, digest)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "status <note>",
		Short: "Print new, unchanged <status> <raw> or modified",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := manifestSetup(env, cmd, args[0])
			if err != nil {
				return err
			}
			digest, err := manifest.Hash(t.disk)
			if err != nil {
				return err
			}
			out, err := manifest.Status(t.file, t.raw, digest)
			if err != nil {
				return err
			}
			fmt.Fprintln(env.Stdout, out)
			return nil
		},
	})
	var status, note string
	var pages []string
	appendCmd := &cobra.Command{
		Use:   "append <note> --status S [--pages P,...] [--note T]",
		Short: "Record a note in the manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := manifestSetup(env, cmd, args[0])
			if err != nil {
				return err
			}
			if status == "" {
				return exit.Usage("--status is required")
			}
			if !slices.Contains(t.b.Cfg.Manifest.Statuses, status) {
				return exit.Usage("status %q is not one of: %s", status, strings.Join(t.b.Cfg.Manifest.Statuses, ", "))
			}
			digest, err := manifest.Hash(t.disk)
			if err != nil {
				return err
			}
			line, err := manifest.Append(t.file, manifest.Record{
				TS:       env.Now().UTC().Format("2006-01-02T15:04:05Z"),
				Raw:      t.raw,
				SHA256:   digest,
				Status:   status,
				Pipeline: t.b.Cfg.Manifest.Pipeline,
				Pages:    pages,
				Note:     note,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(env.Stdout, line)
			return nil
		},
	}
	appendCmd.Flags().StringVar(&status, "status", "", "status of the note")
	appendCmd.Flags().StringSliceVar(&pages, "pages", nil, "pages written from the note, repeated or comma-separated")
	appendCmd.Flags().StringVar(&note, "note", "", "free note")
	cmd.AddCommand(appendCmd)
	return cmd
}
