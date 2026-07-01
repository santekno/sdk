package pricingx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/santekno/sdk/moneyx"
)

// Yahoo is a Provider backed by Yahoo Finance's chart endpoint. It serves IDX
// stocks (ticker `.JK`, e.g. "BBCA.JK") and indices (e.g. "^JKSE" for IHSG).
type Yahoo struct {
	category string
	baseURL  string
	hc       *http.Client
}

// YahooOption configures a Yahoo provider.
type YahooOption func(*Yahoo)

// WithYahooBaseURL overrides the base URL (tests).
func WithYahooBaseURL(u string) YahooOption { return func(y *Yahoo) { y.baseURL = u } }

// WithYahooHTTPClient sets the HTTP client.
func WithYahooHTTPClient(hc *http.Client) YahooOption { return func(y *Yahoo) { y.hc = hc } }

// NewYahoo builds a Yahoo provider for a category ("stock" or "benchmark").
func NewYahoo(category string, opts ...YahooOption) *Yahoo {
	y := &Yahoo{
		category: category,
		baseURL:  "https://query1.finance.yahoo.com",
		hc:       &http.Client{Timeout: 15 * time.Second},
	}
	for _, o := range opts {
		o(y)
	}
	return y
}

// Category implements Provider.
func (y *Yahoo) Category() string { return y.category }

type yahooResp struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice json.Number `json:"regularMarketPrice"`
				Currency           string      `json:"currency"`
				Symbol             string      `json:"symbol"`
			} `json:"meta"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	} `json:"chart"`
}

// GetPrice implements Provider using /v8/finance/chart/<symbol>.
func (y *Yahoo) GetPrice(ctx context.Context, instrumentCode string) (Price, error) {
	url := fmt.Sprintf("%s/v8/finance/chart/%s", y.baseURL, instrumentCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Price{}, err
	}
	res, err := y.hc.Do(req)
	if err != nil {
		return Price{}, err
	}
	defer res.Body.Close()

	var body yahooResp
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Price{}, fmt.Errorf("pricingx: decode yahoo: %w", err)
	}
	if len(body.Chart.Result) == 0 {
		return Price{}, fmt.Errorf("pricingx: yahoo: no result for %q", instrumentCode)
	}
	raw := body.Chart.Result[0].Meta.RegularMarketPrice.String()
	val, err := moneyx.Parse(raw)
	if err != nil {
		return Price{}, fmt.Errorf("pricingx: yahoo: invalid price %q for %q", raw, instrumentCode)
	}
	return Price{Value: val, AsOf: time.Now(), Source: "yahoo"}, nil
}
