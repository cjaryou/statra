package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"golang.org/x/oauth2/google"

	"github.com/cjaryou/statra/internal/types"
)

const storageScope = "https://www.googleapis.com/auth/devstorage.read_only"

// storageClient returns an HTTP client scoped to read the Play reports bucket.
func (g *GooglePlay) storageClient(ctx context.Context) (*http.Client, error) {
	creds, err := google.CredentialsFromJSON(ctx, g.cfg.RawJSON, storageScope)
	if err != nil {
		return nil, fmt.Errorf("loading service account (storage): %w", err)
	}
	return oauthHTTPClient(creds.TokenSource), nil
}

// downloadObject fetches a single object from the reports bucket.
// Returns (nil, nil) when the object does not exist (e.g. no report yet).
func (g *GooglePlay) downloadObject(ctx context.Context, c *http.Client, object string) ([]byte, error) {
	u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s?alt=media",
		g.cfg.ReportsBucket, url.PathEscape(object))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("Cloud Storage 403: the service account can't read the reports bucket yet. "+
			"After granting 'View app information and download bulk reports' in Play Console, Google can take "+
			"a few hours (up to 24h) to propagate bucket access. Retry later.\nraw: %s", string(body))
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("Cloud Storage %d for %s: %s", res.StatusCode, object, string(body))
	}
	return body, nil
}

// decodeUTF16 converts Play's UTF-16 (BOM-prefixed) report bytes to a string.
func decodeUTF16(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE { // UTF-16 LE BOM
		b = b[2:]
		u16 := make([]uint16, len(b)/2)
		for i := range u16 {
			u16[i] = binary.LittleEndian.Uint16(b[i*2:])
		}
		return string(utf16.Decode(u16))
	}
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF { // UTF-16 BE BOM
		b = b[2:]
		u16 := make([]uint16, len(b)/2)
		for i := range u16 {
			u16[i] = binary.BigEndian.Uint16(b[i*2:])
		}
		return string(utf16.Decode(u16))
	}
	return string(b) // assume UTF-8
}

// fetchInstalls downloads the monthly install "overview" reports spanning the
// query range and returns daily install rows within [from, to].
func (g *GooglePlay) fetchInstalls(ctx context.Context, q types.Query) ([]types.Row, error) {
	if g.cfg.ReportsBucket == "" {
		return nil, nil // installs are optional; skip when no bucket configured
	}
	c, err := g.storageClient(ctx)
	if err != nil {
		return nil, err
	}
	from, err := time.Parse("2006-01-02", q.From)
	if err != nil {
		return nil, fmt.Errorf("invalid --from: %w", err)
	}
	to, err := time.Parse("2006-01-02", q.To)
	if err != nil {
		return nil, fmt.Errorf("invalid --to: %w", err)
	}

	var rows []types.Row
	// Iterate month by month (reports are one file per YYYYMM).
	for m := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(to); m = m.AddDate(0, 1, 0) {
		object := fmt.Sprintf("stats/installs/installs_%s_%s_overview.csv", g.cfg.PackageName, m.Format("200601"))
		raw, err := g.downloadObject(ctx, c, object)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			continue
		}
		rows = append(rows, parseInstalls(decodeUTF16(raw), g.cfg.PackageName, from, to)...)
	}
	return rows, nil
}

// parseInstalls reads the install overview CSV and emits daily install rows
// whose date falls within [from, to]. Columns are resolved by header name.
func parseInstalls(csv, pkg string, from, to time.Time) []types.Row {
	sc := bufio.NewScanner(bytes.NewReader([]byte(csv)))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !sc.Scan() {
		return nil
	}
	idx := map[string]int{}
	for i, h := range splitCSV(sc.Text()) {
		idx[strings.TrimSpace(h)] = i
	}
	dateCol, ok1 := idx["Date"]
	instCol, ok2 := idx["Daily Device Installs"]
	if !ok2 {
		instCol, ok2 = idx["Daily User Installs"]
	}
	if !ok1 || !ok2 {
		return nil
	}

	var rows []types.Row
	for sc.Scan() {
		f := splitCSV(sc.Text())
		if dateCol >= len(f) || instCol >= len(f) {
			continue
		}
		date := strings.TrimSpace(f[dateCol])
		d, err := time.Parse("2006-01-02", date)
		if err != nil || d.Before(from) || d.After(to) {
			continue
		}
		installs, _ := strconv.ParseFloat(strings.TrimSpace(f[instCol]), 64)
		rows = append(rows, types.Row{
			Platform: types.Android, App: pkg, AppID: pkg, Kind: types.KindApp,
			Date: date, Metric: types.Installs, Value: installs, Unit: "count",
		})
	}
	return rows
}

// splitCSV splits a simple comma-separated line, trimming surrounding quotes.
func splitCSV(line string) []string {
	parts := strings.Split(line, ",")
	for i, p := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(p), `"`)
	}
	return parts
}
