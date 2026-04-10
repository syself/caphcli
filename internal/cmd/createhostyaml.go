package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/syself/caphcli/internal/createhostyaml"
	"github.com/syself/caphcli/internal/provisioncheck"
)

func newCreateHostYAMLCommand() *cobra.Command {
	cfg := createhostyaml.DefaultConfig()
	cfg.Input = os.Stdin
	cfg.LogOutput = os.Stderr

	cmd := &cobra.Command{
		Use:   "create-host-yaml SERVER_ID OUTPUT_FILE",
		Short: "Generate a HetznerBareMetalHost YAML file for one Robot server",
		Long: `Generate a HetznerBareMetalHost YAML file for one Hetzner Robot server.

The command talks directly to Hetzner Robot, ensures rescue SSH access, reboots
the target server into rescue once, inspects the available disks, and writes a
YAML file to the requested output path. Progress and confirmation prompts go to stderr.`,
		Example: `  caphcli create-host-yaml 1751550 host.yaml
  caphcli create-host-yaml --force --name bm-e2e-1751550 1751550 host.yaml`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			serverID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("parse SERVER_ID %q: %w", args[0], err)
			}
			cfg.ServerID = serverID
			outputFile := args[1]

			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("create output file %q: %w", outputFile, err)
			}
			defer func() {
				if f != nil {
					_ = f.Close()
				}
			}()
			cfg.Output = f

			if err := createhostyaml.Run(context.Background(), cfg); err != nil {
				return fmt.Errorf("caphcli create-host-yaml failed for server %d: %w", cfg.ServerID, err)
			}

			if err := f.Close(); err != nil {
				return fmt.Errorf("close output file %q: %w", outputFile, err)
			}
			f = nil
			_, _ = fmt.Fprintf(cfg.LogOutput, "✓ created %s\n", outputFile)
			_, _ = fmt.Fprintf(cfg.LogOutput, "Hint: run `caphcli check-bm-server %s` next.\n", outputFile)

			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&cfg.Force, "force", false, "Skip the reboot confirmation prompt")
	flags.StringVar(&cfg.Name, "name", "", "metadata.name for the generated HetznerBareMetalHost (default: bm-SERVER_ID)")
	flags.DurationVar(&cfg.PollInterval, "poll-interval", provisioncheck.DefaultPollInterval, "Polling interval while waiting for rescue SSH")
	flags.DurationVar(&cfg.Timeouts.LoadInput, "timeout-load-input", provisioncheck.DefaultLoadInputTimeout, "Timeout for env loading + initial validation")
	flags.DurationVar(&cfg.Timeouts.EnsureSSHKey, "timeout-ensure-ssh-key", provisioncheck.DefaultEnsureSSHKeyTimeout, "Timeout for ensuring SSH key in Robot")
	flags.DurationVar(&cfg.Timeouts.FetchServerDetails, "timeout-fetch-server", provisioncheck.DefaultFetchServerDetailsTimeout, "Timeout for fetching server details from Robot")
	flags.DurationVar(&cfg.Timeouts.ActivateRescue, "timeout-activate-rescue", provisioncheck.DefaultActivateRescueTimeout, "Timeout for activating rescue boot")
	flags.DurationVar(&cfg.Timeouts.RebootToRescue, "timeout-reboot-rescue", provisioncheck.DefaultRebootToRescueTimeout, "Timeout for requesting reboot to rescue")
	flags.DurationVar(&cfg.Timeouts.WaitForRescue, "timeout-wait-rescue", provisioncheck.DefaultWaitForRescueTimeout, "Timeout for waiting until rescue SSH is reachable")

	return cmd
}
