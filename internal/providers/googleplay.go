package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	c := http.Client{
		Transport: &oauth2.Transport{Source: creds.TokenSource, Base: http.DefaultTransport},
		Timeout:   30 * time.Second,
	}
	return &c, nil
}

// Ping verifies credentials by hitting the app's reporting endpoint.
func (g *GooglePlay) Ping() (string, error) {
	ctx := context.Background()
	c, err := g.client(ctx)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/apps/%s:fetchReleaseFilterOptions", reportingBase, g.cfg.PackageName)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	res, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("Reporting API %d: %s", res.StatusCode, string(body))
	}
	return g.cfg.PackageName, nil
}

func (g *GooglePlay) Fetch(q types.Query, metrics []types.Metric) ([]types.Row, error) {
	// Wired next: POST /vitals/crashrate:query etc. plus GCS CSV download.
	_ = q
	_ = metrics
	return nil, fmt.Errorf("GooglePlay.Fetch: metric-set queries not wired yet — run `statra ping android` first to confirm auth, then we implement the crashrate/errors queries")
}
