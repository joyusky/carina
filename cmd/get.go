package cmd

import (
	"github.com/getcarina/carina/console"
	"github.com/spf13/cobra"
)

func newGetCommand() *cobra.Command {
	var options struct {
		name string
		wait bool
	}

	var cmd = &cobra.Command{
		Use:               "get <cluster-name>",
		Short:             "Show information about a cluster",
		Long:              "Show information about a cluster",
		PersistentPreRunE: authenticatedPreRunE,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return bindClusterNameArg(args, &options.name)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cluster, err := cxt.Client.GetCluster(cxt.Account, options.name, options.wait)
			if err != nil {
				return err
			}

			console.WriteCluster(cluster)

			return nil
		},
	}

	cmd.ValidArgs = []string{"cluster-name"}
	cmd.Flags().BoolVar(&options.wait, "wait", false, "Wait for the cluster to become active")
	cmd.SetUsageTemplate(cmd.UsageTemplate())

	return cmd
}
