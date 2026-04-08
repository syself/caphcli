package cmd

import (
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/syself/caphcli/internal/data"
)

func NewRootCommand() *cobra.Command {
	ctrlLog.SetLogger(logr.Discard())
	data.RegisterEmbeddedInstallImageTGZ()

	rootCmd := &cobra.Command{
		Use:               "caphcli",
		Short:             "CAPH developer and operations CLI",
		DisableAutoGenTag: true,
		SilenceUsage:      true,
		SilenceErrors:     false,
	}

	rootCmd.AddCommand(newCheckBMServersCommand())
	rootCmd.AddCommand(newCreateHostTemplateCommand())

	return rootCmd
}
