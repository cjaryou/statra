package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/cjaryou/statra/internal/config"
	"github.com/cjaryou/statra/internal/types"
)

const reportingBase = "https://playdeveloperreporting.googleapis.com/v1beta1"
const reportingScope = "https://www.googleapis.com/auth/playdeveloperreporting"

// GooglePlay implements types.Provider for Google Play Console.
//
// Auth: service-account JWT -> OAuth2 access token (x/oauth2/google).
// Vitals (crashes, ANR, slow start, active devices) come from the Play
// Developer Reporting API metric sets. Downloads/revenue come from the
// monthly CSV reports Play writes to a GCS bucket.
type GooglePlay struct {
	cfg *config.Google
	hc  *http.Client
}

func NewGooglePlay() (*GooglePlay, error) {
	cfg, err := config.LoadGoogle()
	if err != nil {
		return nil, err
	}
	return &GooglePlay{cfg: cfg, hc: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (g *GooglePlay) Platform() types.Platform { return types.Android }

// client returns an HTTP client that injects fresh OAuth2 access tokens.
func (g *GooglePlay) client(ctx context.Context) (*http.Client, error) {
	creds, err := google.CredentialsFromJSON(ctx, g.cfg.RawJSON, reportingScope)
	if err != nil {
		return nil, fmt.Errorf("loading service account: %w", err)
	}
	return oauthHTTPClient(creds.TokenSource), nil
}

// oauthHTTPClient builds an HTTP client that injects tokens from the source.
func oauthHTTPClient(src oauth2.TokenSource) *http.Client {
	return &http.Client{
		Transport: &oauth2.Transport{Source: src, Base: http.DefaultTransport},
		Timeout:   60 * time.Second,
	}
}

// Ping verifies credentials by reading the app's crash-rate metric set
// metadata (a documented GET resource that requires app access).
func (g *GooglePlay) Ping() (string, error) {
	ctx := context.Background()
	c, err := g.client(ctx)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/apps/%s/crashRateMetricSet", reportingBase, g.cfg.PackageName)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	res, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("Reporting API %d: %s", res.StatusCode, string(body))
	}
	var ms struct {
		FreshnessInfo struct {
			Freshnesses []struct {
				AggregationPeriod string `json:"aggregationPeriod"`
				LatestEndTime     struct{ Year, Month, Day int } `json:"latestEndTime"`
			} `json:"freshnesses"`
		} `json:"freshnessInfo"`
	}
	_ = json.Unmarshal(body, &ms)
	if len(ms.FreshnessInfo.Freshnesses) == 0 {
		return g.cfg.PackageName + " (auth OK, but no vitals data — app volume likely below Google's reporting threshold)", nil
	}
	info := g.cfg.PackageName + " — vitals available through:"
	for _, f := range ms.FreshnessInfo.Freshnesses {
		t := f.LatestEndTime
		info += fmt.Sprintf(" %s=%04d-%02d-%02d", f.AggregationPeriod, t.Year, t.Month, t.Day)
	}
	return info, nil
}

// Fetch queries the Play Developer Reporting API for vitals (crash rate).
//
// Note: installs and revenue are NOT exposed by the Reporting API — Play writes
// those as monthly CSV reports to a GCS bucket (GOOGLE_REPORTS_BUCKET), wired
// separately. This returns daily crash-rate rows per the configured app.
func (g *GooglePlay) Fetch(q types.Query, metrics []types.Metric) ([]types.Row, error) {
	_ = metrics
	ctx := context.Background()

	// Installs come from the GCS bulk reports (no privacy threshold).
	installs, err := g.fetchInstalls(ctx, q)
	if err != nil {
		return nil, err
	}

	c, err := g.client(ctx)
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

	body := map[string]any{
		"timelineSpec": map[string]any{
			"aggregationPeriod": "DAILY",
			"startTime":         date(from),
			"endTime":           date(to.AddDate(0, 0, 1)), // endTime is exclusive
		},
		"metrics": []string{"crashRate"},
	}
	buf, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/apps/%s/crashRateMetricSet:query", reportingBase, g.cfg.PackageName)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("Reporting API %d: %s", res.StatusCode, string(raw))
	}
	if os.Getenv("STATRA_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[debug] crashRate query response:\n%s\n", string(raw))
	}

	var out struct {
		Rows []struct {
			StartTime struct {
				Year, Month, Day int
			} `json:"startTime"`
			Metrics []struct {
				Metric       string `json:"metric"`
				DecimalValue struct {
					Value string `json:"value"`
				} `json:"decimalValue"`
			} `json:"metrics"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	rows := installs // start with install rows, append vitals below
	for _, r := range out.Rows {
		d := fmt.Sprintf("%04d-%02d-%02d", r.StartTime.Year, r.StartTime.Month, r.StartTime.Day)
		for _, m := range r.Metrics {
			if m.Metric != "crashRate" {
				continue
			}
			v, _ := strconv.ParseFloat(m.DecimalValue.Value, 64)
			rows = append(rows, types.Row{
				Platform: types.Android, App: g.cfg.PackageName, AppID: g.cfg.PackageName,
				Kind: types.KindApp, Date: d, Metric: types.Crashes, Value: v, Unit: "rate",
			})
		}
	}
	return rows, nil
}

// date renders a Go time as the API's {year,month,day} object.
func date(t time.Time) map[string]int {
	return map[string]int{"year": t.Year(), "month": int(t.Month()), "day": t.Day()}
}
