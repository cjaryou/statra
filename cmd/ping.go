package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cjaryou/statra/internal/providers"
)

var pingCmd = &cobra.Command{
	Use:       "ping <platform>",
	Short:     "Verify credentials by reaching the store API",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"ios", "android"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "ios":
			p, err := providers.NewAppStore()
			if err != nil {
				return err
			}
			name, err := p.Ping()
			if err != nil {
				return err
			}
			fmt.Printf("✅ App Store Connect OK — app: %s\n", name)
		case "android":
			p, err := providers.NewGooglePlay()
			if err != nil {
				return err
			}
			pkg, err := p.Ping()
			if err != nil {
				return err
			}
			fmt.Printf("✅ Google Play OK — package: %s\n", pkg)
		default:
			return fmt.Errorf("platform must be 'ios' or 'android'")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
