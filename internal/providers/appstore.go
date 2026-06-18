package providers

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cjaryou/statra/internal/config"
	"github.com/cjaryou/statra/internal/types"
)

const ascBase = "https://api.appstoreconnect.apple.com/v1"

// AppStore implements types.Provider for App Store Connect.
//
// Auth: ES256-signed JWT from the .p8 key (20-min expiry per Apple's spec).
// Stats: downloads/revenue come from the Analytics Reports API (async:
// request a report -> poll instances -> download gzipped TSV); sales numbers
// come from salesReports (synchronous gzipped TSV).
type AppStore struct {
	cfg *config.Apple
	hc  *http.Client
}

func NewAppStore() (*AppStore, error) {
	cfg, err := config.LoadApple()
	if err != nil {
		return nil, err
	}
	return &AppStore{cfg: cfg, hc: &http.Client{Timeout: 30 * time.Second}}, nil
}

func (a *AppStore) Platform() types.Platform { return types.IOS }

// token mints a short-lived bearer token signed with the App Store Connect key.
func (a *AppStore) token() (string, error) {
	block, _ := pem.Decode(a.cfg.PrivateKey)
	if block == nil {
		return "", fmt.Errorf("invalid .p8: no PEM block found")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing .p8 key: %w", err)
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": a.cfg.IssuerID,
		"iat": now.Unix(),
		"exp": now.Add(20 * time.Minute).Unix(),
		"aud": "appstoreconnect-v1",
	})
	tok.Header["kid"] = a.cfg.KeyID
	return tok.SignedString(key)
}

func (a *AppStore) get(path string) ([]byte, error) {
	tok, err := a.token()
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodGet, ascBase+path, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := a.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("ASC %d: %s", res.StatusCode, string(body))
	}
	return body, nil
}

// Ping verifies credentials by fetching the configured app and returns its name.
func (a *AppStore) Ping() (string, error) {
	body, err := a.get("/apps/" + a.cfg.AppID)
	if err != nil {
		return "", err
	}
	var out struct {
		Data struct {
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.Data.Attributes.Name == "" {
		return a.cfg.AppID, nil
	}
	return out.Data.Attributes.Name, nil
}

func (a *AppStore) Fetch(q types.Query, metrics []types.Metric) ([]types.Row, error) {
	// Wired next: POST /analyticsReportRequests, poll instances, gunzip+parse TSV.
	_ = q
	_ = metrics
	return nil, fmt.Errorf("AppStore.Fetch: report pipeline not wired yet — run `statra ping ios` first to confirm auth, then we implement analyticsReportRequests")
}
