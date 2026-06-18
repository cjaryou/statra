package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/cjaryou/statra/internal/providers"
	"github.com/cjaryou/statra/internal/types"
)

var (
	statsFrom string
	statsTo   string
	statsApp  string
	statsCSV  bool
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
				if platform == "android" {
					return err
				} // in `all` mode, skip Android if not configured
			} else {
				r, err := p.Fetch(q, nil)
				if err != nil {
					if platform == "android" {
						return err
					}
					fmt.Fprintf(os.Stderr, "warning: android skipped: %v\n", err)
				}
				rows = append(rows, r...)
			}
		}

		rows = filterApp(rows, statsApp)

		switch {
		case JSONOutput:
			return emitJSON(rows, q)
		case statsCSV:
			return emitCSV(rows)
		default:
			printStats(rows, q)
			return nil
		}
	},
}

// filterApp keeps rows whose app id equals, or name contains, the query.
func filterApp(rows []types.Row, query string) []types.Row {
	if query == "" {
		return rows
	}
	q := strings.ToLower(query)
	out := rows[:0]
	for _, r := range rows {
		if r.AppID == query || strings.Contains(strings.ToLower(r.App), q) {
			out = append(out, r)
		}
	}
	return out
}

func emitJSON(rows []types.Row, q types.Query) error {
	if rows == nil {
		rows = []types.Row{}
	}
	out := struct {
		From string      `json:"from"`
		To   string      `json:"to"`
		Rows []types.Row `json:"rows"`
	}{From: q.From, To: q.To, Rows: rows}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func emitCSV(rows []types.Row) error {
	w := csv.NewWriter(os.Stdout)
	_ = w.Write([]string{"platform", "app", "app_id", "kind", "date", "metric", "value", "unit"})
	for _, r := range rows {
		_ = w.Write([]string{
			string(r.Platform), r.App, r.AppID, string(r.Kind), r.Date,
			string(r.Metric), strconv.FormatFloat(r.Value, 'f', -1, 64), r.Unit,
		})
	}
	w.Flush()
	return w.Error()
}

type appAgg struct {
	name     string
	platform types.Platform
	kind     types.Kind
	installs float64
	revenue  map[string]float64 // currency -> amount
}

// printStats aggregates rows per product and renders app + subscription tables.
func printStats(rows []types.Row, q types.Query) {
	byKey := map[string]*appAgg{}
	for _, r := range rows {
		key := string(r.Platform) + "|" + string(r.Kind) + "|" + r.AppID + "|" + r.App
		a := byKey[key]
		if a == nil {
			a = &appAgg{name: r.App, platform: r.Platform, kind: r.Kind, revenue: map[string]float64{}}
			byKey[key] = a
		}
		switch r.Metric {
		case types.Installs:
			a.installs += r.Value
		case types.Revenue:
			if r.Unit != "" {
				a.revenue[r.Unit] += r.Value
			}
		}
	}
	if len(byKey) == 0 {
		fmt.Printf("No data for %s → %s.\n", q.From, q.To)
		fmt.Println("  • Apple sales reports can lag ~1-2 days.")
		fmt.Println("  • Google vitals are hidden for apps below its volume threshold.")
		return
	}

	var apps, iaps []*appAgg
	for _, a := range byKey {
		if a.kind == types.KindIAP {
			iaps = append(iaps, a)
		} else if a.installs != 0 || totalMoney(a.revenue) != 0 {
			// skip metric-only entries (e.g. Android crash rate) from the install table
			apps = append(apps, a)
		}
	}

	fmt.Printf("\nstatra — stats %s → %s\n", q.From, q.To)
	grandRev := map[string]float64{}

	if len(apps) > 0 {
		sort.Slice(apps, func(i, j int) bool { return apps[i].installs > apps[j].installs })
		fmt.Printf("\nAPPS (downloads)\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "PLATFORM\tAPP\tINSTALLS\tREVENUE")
		var totalInstalls float64
		for _, a := range apps {
			fmt.Fprintf(w, "%s\t%s\t%.0f\t%s\n", a.platform, a.name, a.installs, fmtMoney(a.revenue))
			totalInstalls += a.installs
			addMoney(grandRev, a.revenue)
		}
		fmt.Fprintf(w, "\tTOTAL\t%.0f\t\n", totalInstalls)
		w.Flush()
	}

	if len(iaps) > 0 {
		sort.Slice(iaps, func(i, j int) bool { return totalMoney(iaps[i].revenue) > totalMoney(iaps[j].revenue) })
		fmt.Printf("\nSUBSCRIPTIONS / IN-APP (revenue)\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "PRODUCT\tREVENUE")
		for _, a := range iaps {
			fmt.Fprintf(w, "%s\t%s\n", a.name, fmtMoney(a.revenue))
			addMoney(grandRev, a.revenue)
		}
		w.Flush()
	}

	fmt.Printf("\nTOTAL REVENUE: %s\n", fmtMoney(grandRev))
}

func addMoney(dst, src map[string]float64) {
	for c, v := range src {
		dst[c] += v
	}
}

func totalMoney(m map[string]float64) float64 {
	var t float64
	for _, v := range m {
		t += v
	}
	return t
}

// fmtMoney renders per-currency amounts; mixed currencies are never summed.
func fmtMoney(m map[string]float64) string {
	if len(m) == 0 {
		return "-"
	}
	curs := make([]string, 0, len(m))
	for c := range m {
		if m[c] != 0 { // hide zero-revenue currencies as noise
			curs = append(curs, c)
		}
	}
	if len(curs) == 0 {
		return "-"
	}
	sort.Slice(curs, func(i, j int) bool { return m[curs[i]] > m[curs[j]] })
	parts := make([]string, 0, len(curs))
	for _, c := range curs {
		parts = append(parts, fmt.Sprintf("%.2f %s", m[c], c))
	}
	return strings.Join(parts, " + ")
}

func init() {
	statsCmd.Flags().StringVar(&statsFrom, "from", "", "start date YYYY-MM-DD (default: 7 days ago)")
	statsCmd.Flags().StringVar(&statsTo, "to", "", "end date YYYY-MM-DD (default: yesterday)")
	statsCmd.Flags().StringVar(&statsApp, "app", "", "filter by app id or name substring")
	statsCmd.Flags().BoolVar(&statsCSV, "csv", false, "output CSV")
	rootCmd.AddCommand(statsCmd)
}
