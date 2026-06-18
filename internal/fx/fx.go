// Package fx converts monetary amounts between currencies using live rates,
// so cross-platform revenue (USD + EUR + NOK + …) can be summed into one number.
package fx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ratesURL returns USD-based rates (no API key required).
const ratesURL = "https://open.er-api.com/v6/latest/USD"

// Converter holds USD-based rates: rates[CUR] = how many CUR per 1 USD.
type Converter struct {
	rates map[string]float64
}

// Fetch loads current exchange rates.
func Fetch() (*Converter, error) {
	c := &http.Client{Timeout: 15 * time.Second}
	res, err := c.Get(ratesURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("fx rates %d: %s", res.StatusCode, string(body))
	}
	var out struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Result != "success" || len(out.Rates) == 0 {
		return nil, fmt.Errorf("fx rates unavailable")
	}
	return &Converter{rates: out.Rates}, nil
}

// Convert returns amount expressed in `from` currency, valued in `to` currency.
func (c *Converter) Convert(amount float64, from, to string) (float64, error) {
	from, to = strings.ToUpper(from), strings.ToUpper(to)
	if from == to {
		return amount, nil
	}
	rFrom, ok := c.rates[from]
	if !ok || rFrom == 0 {
		return 0, fmt.Errorf("unknown currency: %s", from)
	}
	rTo, ok := c.rates[to]
	if !ok {
		return 0, fmt.Errorf("unknown currency: %s", to)
	}
	usd := amount / rFrom // from -> USD
	return usd * rTo, nil // USD -> to
}
