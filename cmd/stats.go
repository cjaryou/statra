package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/cjaryou/statra/internal/providers"
	"github.com/cjaryou/statra/internal/types"
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

		// Default to the last 7 full days if no range is given.
		if statsTo == "" {
			statsTo = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		}
		if statsFrom == "" {
			statsFrom = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		}
		q := types.Query{From: statsFrom, To: statsTo}

		var rows []types.Row
		if platform == "ios" || platform == "all" {
			p, err := providers.NewAppStore()
			if err != nil {
				return err
			}
			r, err := p.Fetch(q, nil)
			if err != nil {
				return err
			}
			rows = append(rows, r...)
		}
		if platform == "android" || platform == "all" {
			p, err := providers.NewGooglePlay()
			if err != nil {
				// Android optional in `all` mode if not configured yet.
				if platform == "android" {
					return err
				}
			} else {
				r, err := p.Fetch(q, nil)
				if err != nil && platform == "android" {
					return err
				}
				rows = append(rows, r...)
			}
		}

		printStats(rows, q)
		return nil
	},
}

// printStats aggregates rows per app and renders a table.
func printStats(rows []types.Row, q types.Query) {
	type agg struct {
		name     string
		platform types.Platform
		installs float64
		revenue  float64
		currency string
	}
	byApp := map[string]*agg{}
	for _, r := range rows {
		key := string(r.Platform) + "|" + r.AppID + "|" + r.App
		a := byApp[key]
		if a == nil {
			a = &agg{name: r.App, platform: r.Platform}
			byApp[key] = a
		}
		switch r.Metric {
		case types.Installs:
			a.installs += r.Value
		case types.Revenue:
			a.revenue += r.Value
			a.currency = r.Unit
		}
	}

	if len(byApp) == 0 {
		fmt.Printf("No data for %s → %s (Apple sales reports can lag ~1-2 days).\n", q.From, q.To)
		return
	}

	list := make([]*agg, 0, len(byApp))
	for _, a := range byApp {
		list = append(list, a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].installs > list[j].installs })

	fmt.Printf("\nstatra — stats %s → %s\n\n", q.From, q.To)
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "PLATFORM\tAPP\tINSTALLS\tREVENUE")
	var totalInstalls, totalRevenue float64
	var cur string
	for _, a := range list {
		rev := "-"
		if a.currency != "" {
			rev = fmt.Sprintf("%.2f %s", a.revenue, a.currency)
			cur = a.currency
		}
		fmt.Fprintf(w, "%s\t%s\t%.0f\t%s\n", a.platform, a.name, a.installs, rev)
		totalInstalls += a.installs
		totalRevenue += a.revenue
	}
	fmt.Fprintf(w, "\tTOTAL\t%.0f\t%.2f %s\n", totalInstalls, totalRevenue, cur)
	w.Flush()
}

func init() {
	statsCmd.Flags().StringVar(&statsFrom, "from", "", "start date YYYY-MM-DD (default: 7 days ago)")
	statsCmd.Flags().StringVar(&statsTo, "to", "", "end date YYYY-MM-DD (default: yesterday)")
	rootCmd.AddCommand(statsCmd)
}
