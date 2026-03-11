package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func cmdProfile() *cobra.Command {
	var profileName string

	cmd := &cobra.Command{
		Use:   "profile <node-repo>",
		Short: "Show the node profile that would be used for a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath := args[0]
			prof, err := resolveProfile(profileName, repoPath)
			if err != nil {
				return err
			}

			source := profileName
			if source == "" {
				source = filepath.Join(repoPath, ".node-bench.toml")
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "name:         %s\n", prof.Name)
			fmt.Fprintf(w, "build_cmd:    %s\n", strings.Join(prof.BuildCmd, " "))
			fmt.Fprintf(w, "binary_path:  %s\n", prof.BinaryPath)
			networkDisplay := prof.Flags.Network
			if prof.Flags.NetworkFlag != "" {
				networkDisplay = prof.Flags.NetworkFlag + " (boolean flag)"
			}
			if networkDisplay == "" {
				networkDisplay = "(none)"
			}
			fmt.Fprintf(w, "flags:\n")
			fmt.Fprintf(w, "  network:  %s\n", networkDisplay)
			fmt.Fprintf(w, "  datadir:  %s\n", prof.Flags.Datadir)
			fmt.Fprintf(w, "  connect:  %s\n", prof.Flags.Connect)
			if len(prof.ExtraArgs) > 0 {
				fmt.Fprintf(w, "extra_args:   %s\n", strings.Join(prof.ExtraArgs, " "))
			}
			fmt.Fprintf(w, "log patterns:\n")
			fmt.Fprintf(w, "  connected_to_peer:  %s\n", prof.Logs.ConnectedToPeer)
			fmt.Fprintf(w, "  update_tip:         %s\n", prof.Logs.UpdateTip)
			fmt.Fprintf(w, "  timestamp_layout:   %s\n", prof.Logs.TimestampLayout)
			tsPattern := prof.Logs.TimestampPattern
			if tsPattern == "" {
				tsPattern = "(default)"
			}
			fmt.Fprintf(w, "  timestamp_pattern:  %s\n", tsPattern)
			hPattern := prof.Logs.HeightPattern
			if hPattern == "" {
				hPattern = "(default)"
			}
			fmt.Fprintf(w, "  height_pattern:     %s\n", hPattern)
			cpStrs := make([]string, len(prof.Logs.Checkpoints))
			for i, h := range prof.Logs.Checkpoints {
				cpStrs[i] = strconv.Itoa(h)
			}
			cpDisplay := strings.Join(cpStrs, ", ")
			if cpDisplay == "" {
				cpDisplay = "(default)"
			}
			fmt.Fprintf(w, "  checkpoints:        %s\n", cpDisplay)
			peersDisplay := strings.Join(prof.DefaultPeers, ", ")
			if peersDisplay == "" {
				peersDisplay = "(none — --connect required)"
			}
			fmt.Fprintf(w, "default_peers: %s\n", peersDisplay)
			fmt.Fprintf(w, "source:       %s\n", source)
			return nil
		},
	}

	cmd.Flags().StringVar(&profileName, "profile", "", "node profile name, directory, or .toml path (default: auto-detect from repo)")
	return cmd
}
