package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	statsFrom string
	statsTo   string
)

var statsCmd = &cobra.Command{
	Use:   "stats [platform]",
	Short: "Fetch and merge cross-platform stats (ios | android | all)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := "all"
		if len(args) == 1 {
			platform = args[0]
		}
		_ = platform
		fmt.Println("stats: providers are scaffolded but the report pipelines are not wired yet.")
		fmt.Println("Next step: run `statra ping ios` / `ping android` with real credentials,")
		fmt.Println("then we implement analyticsReportRequests (Apple) and metric-set queries (Google).")
		return nil
	},
}

func init() {
	statsCmd.Flags().StringVar(&statsFrom, "from", "", "start date YYYY-MM-DD")
	statsCmd.Flags().StringVar(&statsTo, "to", "", "end date YYYY-MM-DD")
	rootCmd.AddCommand(statsCmd)
}
