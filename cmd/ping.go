package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
			return pingIOS()
		case "android":
			return pingAndroid()
		default:
			return fmt.Errorf("platform must be 'ios' or 'android'")
		}
	},
}

func pingIOS() error {
	p, err := providers.NewAppStore()
	if err != nil {
		return err
	}
	apps, err := p.Apps()
	if err != nil {
		return err
	}
	if JSONOutput {
		return emitPingJSON(map[string]any{"platform": "ios", "ok": true, "apps": apps})
	}
	fmt.Printf("✅ App Store Connect OK — %d app(s):\n", len(apps))
	for _, a := range apps {
		fmt.Printf("  • %s  (id: %s, bundle: %s)\n", a.Name, a.ID, a.BundleID)
	}
	return nil
}

func pingAndroid() error {
	p, err := providers.NewGooglePlay()
	if err != nil {
		return err
	}
	pkg, err := p.Ping()
	if err != nil {
		return err
	}
	if JSONOutput {
		return emitPingJSON(map[string]any{"platform": "android", "ok": true, "package": pkg})
	}
	fmt.Printf("✅ Google Play OK — package: %s\n", pkg)
	return nil
}

func emitPingJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
