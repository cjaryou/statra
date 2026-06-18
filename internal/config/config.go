// Package config loads credentials from the environment (see .env.example).
// It reads a local .env file if present so users don't need to export vars
// manually, keeping the tool dependency-free for configuration.
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var loaded bool

// LoadDotEnv reads ./.env into the process environment without overriding
// values already set. Safe to call multiple times.
func LoadDotEnv() {
	if loaded {
		return
	}
	loaded = true
	f, err := os.Open(".env")
	if err != nil {
		return // no .env — rely on the real environment
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if _, exists := os.LookupEnv(k); !exists {
			os.Setenv(k, v)
		}
	}
}

func required(name string) (string, error) {
	LoadDotEnv()
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("missing required env var: %s (see .env.example)", name)
	}
	return v, nil
}

// Apple holds App Store Connect API credentials.
type Apple struct {
	IssuerID     string
	KeyID        string
	PrivateKey   []byte // PEM contents of the .p8 file
	AppID        string
	VendorNumber string // required for Sales/Finance reports
}

func LoadApple() (*Apple, error) {
	issuer, err := required("ASC_ISSUER_ID")
	if err != nil {
		return nil, err
	}
	keyID, err := required("ASC_KEY_ID")
	if err != nil {
		return nil, err
	}
	keyPath, err := required("ASC_PRIVATE_KEY_PATH")
	if err != nil {
		return nil, err
	}
	pem, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading ASC_PRIVATE_KEY_PATH: %w", err)
	}
	// AppID is optional: when unset, `ping` lists all apps and shows their IDs.
	// VendorNumber is only needed for `stats` (Sales Reports).
	return &Apple{
		IssuerID:     issuer,
		KeyID:        keyID,
		PrivateKey:   pem,
		AppID:        os.Getenv("ASC_APP_ID"),
		VendorNumber: os.Getenv("ASC_VENDOR_NUMBER"),
	}, nil
}

// ServiceAccount is the subset of a Google service-account JSON we need.
type ServiceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// Google holds Play Console credentials.
type Google struct {
	Account       ServiceAccount
	RawJSON       []byte
	PackageName   string
	ReportsBucket string
}

func LoadGoogle() (*Google, error) {
	saPath, err := required("GOOGLE_SERVICE_ACCOUNT_JSON")
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(saPath)
	if err != nil {
		return nil, fmt.Errorf("reading GOOGLE_SERVICE_ACCOUNT_JSON: %w", err)
	}
	var sa ServiceAccount
	if err := json.Unmarshal(raw, &sa); err != nil {
		return nil, fmt.Errorf("parsing service account JSON: %w", err)
	}
	pkg, err := required("GOOGLE_PACKAGE_NAME")
	if err != nil {
		return nil, err
	}
	return &Google{
		Account:       sa,
		RawJSON:       raw,
		PackageName:   pkg,
		ReportsBucket: os.Getenv("GOOGLE_REPORTS_BUCKET"),
	}, nil
}
