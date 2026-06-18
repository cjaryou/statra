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

// App is a minimal App Store Connect app record.
type App struct {
	ID       string
	Name     string
	BundleID string
	SKU      string
}

// Apps lists all apps visible to the API key.
func (a *AppStore) Apps() ([]App, error) {
	body, err := a.get("/apps?limit=200")
	if err != nil {
		return nil, err
	}
	var out struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name     string `json:"name"`
				BundleID string `json:"bundleId"`
				SKU      string `json:"sku"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	apps := make([]App, 0, len(out.Data))
	for _, d := range out.Data {
		apps = append(apps, App{ID: d.ID, Name: d.Attributes.Name, BundleID: d.Attributes.BundleID, SKU: d.Attributes.SKU})
	}
	return apps, nil
}

// Ping verifies credentials. With ASC_APP_ID set it fetches that app; without
// it, it lists all apps so you can discover the IDs.
func (a *AppStore) Ping() (string, error) {
	if a.cfg.AppID == "" {
		apps, err := a.Apps()
		if err != nil {
			return "", err
		}
		if len(apps) == 0 {
			return "no apps found (auth OK, but this key sees no apps)", nil
		}
		out := fmt.Sprintf("%d app(s) found:\n", len(apps))
		for _, app := range apps {
			out += fmt.Sprintf("  • %s  (id: %s, bundle: %s)\n", app.Name, app.ID, app.BundleID)
		}
		out += "\nTip: set ASC_APP_ID in .env to one of the ids above."
		return out, nil
	}
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

// Fetch pulls daily Sales Reports (installs + revenue) for the date range.
func (a *AppStore) Fetch(q types.Query, metrics []types.Metric) ([]types.Row, error) {
	_ = metrics // currently Sales Reports yield installs + revenue together
	return a.fetchSales(q)
}
