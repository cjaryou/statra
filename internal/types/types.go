// Package types holds the cross-platform data shapes so iOS and Android
// stats can be normalized into a single mergeable form.
package types

type Platform string

const (
	IOS     Platform = "ios"
	Android Platform = "android"
)

// Query is an inclusive date range, dates formatted YYYY-MM-DD.
type Query struct {
	From string
	To   string
}

type Metric string

const (
	Installs      Metric = "installs"
	Uninstalls    Metric = "uninstalls"
	ActiveDevices Metric = "active_devices"
	Crashes       Metric = "crashes"
	ANR           Metric = "anr" // android only
	Rating        Metric = "rating"
	Revenue       Metric = "revenue"
)

// Row is one normalized metric data point shared across both stores.
type Row struct {
	Platform Platform `json:"platform"`
	Date     string   `json:"date"` // YYYY-MM-DD
	Metric   Metric   `json:"metric"`
	Value    float64  `json:"value"`
	Unit     string   `json:"unit,omitempty"` // e.g. "USD", "count"
}

// Provider is implemented by each store backend.
type Provider interface {
	Platform() Platform
	Ping() (string, error)
	Fetch(q Query, metrics []Metric) ([]Row, error)
}
