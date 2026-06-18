// Package cmd wires the statra CLI commands.
package cmd

import (
	"github.com/spf13/cobra"
)

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

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
