package providers

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cjaryou/statra/internal/types"
)

// salesReportRaw fetches a gzipped TSV sales report for a single day.
// Returns (nil, nil) when Apple has no report for that date yet (HTTP 404).
func (a *AppStore) salesReportRaw(date string) ([]byte, error) {
	tok, err := a.token()
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("filter[frequency]", "DAILY")
	q.Set("filter[reportType]", "SALES")
	q.Set("filter[reportSubType]", "SUMMARY")
	q.Set("filter[vendorNumber]", a.cfg.VendorNumber)
	q.Set("filter[reportDate]", date)
	q.Set("filter[version]", "1_1")

	req, _ := http.NewRequest(http.MethodGet, ascBase+"/salesReports?"+q.Encode(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/a-gzip")

	res, err := a.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusNotFound {
		return nil, nil // no data for this day
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("ASC salesReports %d: %s", res.StatusCode, string(body))
	}

	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gunzip sales report: %w", err)
	}
	defer gz.Close()
	return io.ReadAll(gz)
}

// parseSales turns a SALES/SUMMARY TSV into installs + revenue rows for one day.
// Columns are resolved by header name since they vary between report versions.
func parseSales(tsv []byte, date string) []types.Row {
	sc := bufio.NewScanner(bytes.NewReader(tsv))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !sc.Scan() {
		return nil
	}
	idx := map[string]int{}
	for i, h := range strings.Split(sc.Text(), "\t") {
		idx[strings.TrimSpace(h)] = i
	}
	col := func(fields []string, name string) string {
		if i, ok := idx[name]; ok && i < len(fields) {
			return strings.TrimSpace(fields[i])
		}
		return ""
	}

	// Aggregate per app across all SKUs/countries for the day.
	type agg struct {
		name     string
		id       string
		units    float64
		proceeds float64
		currency string
	}
	apps := map[string]*agg{}

	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		id := col(f, "Apple Identifier")
		if id == "" {
			continue
		}
		units, _ := strconv.ParseFloat(col(f, "Units"), 64)
		proceeds, _ := strconv.ParseFloat(col(f, "Developer Proceeds"), 64)
		a := apps[id]
		if a == nil {
			a = &agg{name: col(f, "Title"), id: id, currency: col(f, "Currency of Proceeds")}
			apps[id] = a
		}
		a.units += units
		a.proceeds += proceeds * units
	}

	rows := make([]types.Row, 0, len(apps)*2)
	for _, a := range apps {
		rows = append(rows,
			types.Row{Platform: types.IOS, App: a.name, AppID: a.id, Date: date, Metric: types.Installs, Value: a.units, Unit: "count"},
			types.Row{Platform: types.IOS, App: a.name, AppID: a.id, Date: date, Metric: types.Revenue, Value: a.proceeds, Unit: a.currency},
		)
	}
	return rows
}

// Fetch pulls daily Sales Reports across the date range and returns normalized
// installs + revenue rows per app per day.
func (a *AppStore) fetchSales(q types.Query) ([]types.Row, error) {
	if a.cfg.VendorNumber == "" {
		return nil, fmt.Errorf("ASC_VENDOR_NUMBER is required for stats — find it in App Store Connect → Payments and Financial Reports")
	}
	from, err := time.Parse("2006-01-02", q.From)
	if err != nil {
		return nil, fmt.Errorf("invalid --from: %w", err)
	}
	to, err := time.Parse("2006-01-02", q.To)
	if err != nil {
		return nil, fmt.Errorf("invalid --to: %w", err)
	}

	var all []types.Row
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		tsv, err := a.salesReportRaw(d.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		if tsv == nil {
			continue
		}
		all = append(all, parseSales(tsv, d.Format("2006-01-02"))...)
	}
	return all, nil
}
