package pricingx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/santekno/sdk/moneyx"
)

// CoinGecko is a Provider for crypto prices in IDR (category "crypto"). The
// instrument code is a CoinGecko id, e.g. "bitcoin".
type CoinGecko struct {
	baseURL string
	vs      string
	hc      *http.Client
}

// CoinGeckoOption configures a CoinGecko provider.
type CoinGeckoOption func(*CoinGecko)

// WithCoinGeckoBaseURL overrides the base URL (tests).
func WithCoinGeckoBaseURL(u string) CoinGeckoOption { return func(c *CoinGecko) { c.baseURL = u } }

// WithCoinGeckoHTTPClient sets the HTTP client.
func WithCoinGeckoHTTPClient(hc *http.Client) CoinGeckoOption {
	return func(c *CoinGecko) { c.hc = hc }
}

// NewCoinGecko builds a CoinGecko provider (prices in IDR).
func NewCoinGecko(opts ...CoinGeckoOption) *CoinGecko {
	c := &CoinGecko{
		baseURL: "https://api.coingecko.com",
		vs:      "idr",
		hc:      &http.Client{Timeout: 15 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Category implements Provider.
func (c *CoinGecko) Category() string { return "crypto" }

// GetPrice implements Provider using /api/v3/simple/price.
func (c *CoinGecko) GetPrice(ctx context.Context, instrumentCode string) (Price, error) {
	url := fmt.Sprintf("%s/api/v3/simple/price?ids=%s&vs_currencies=%s", c.baseURL, instrumentCode, c.vs)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Price{}, err
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return Price{}, err
	}
	defer res.Body.Close()

	// { "bitcoin": { "idr": 1234567890 } }
	var body map[string]map[string]json.Number
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Price{}, fmt.Errorf("pricingx: decode coingecko: %w", err)
	}
	coin, ok := body[instrumentCode]
	if !ok {
		return Price{}, fmt.Errorf("pricingx: coingecko: no data for %q", instrumentCode)
	}
	raw, ok := coin[c.vs]
	if !ok {
		return Price{}, fmt.Errorf("pricingx: coingecko: no %s price for %q", c.vs, instrumentCode)
	}
	val, err := moneyx.Parse(raw.String())
	if err != nil {
		return Price{}, fmt.Errorf("pricingx: coingecko: invalid price %q", raw.String())
	}
	return Price{Value: val, AsOf: time.Now(), Source: "coingecko"}, nil
}
