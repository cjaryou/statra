// Package cmd wires the statra CLI commands.
package cmd

import (
	"github.com/spf13/cobra"
)

// JSONOutput is set by the global --json flag. When true, commands emit
// machine-readable JSON to stdout (for AI agents, scripts and pipelines)
// instead of human-formatted tables. Diagnostics always go to stderr.
var JSONOutput bool

var rootCmd = &cobra.Command{
	Use:   "statra",
	Short: "One CLI for App Store Connect + Google Play stats",
	Long: `statra — the cross-platform CLI for App Store Connect + Google Play Console stats.

Pull downloads, revenue, crashes, ANR and ratings for iOS and Android in one
command. An open-source alternative to app-store-connect, fastlane and gpc.`,
	Version:       "0.1.0",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "machine-readable JSON output (for agents/scripts)")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
