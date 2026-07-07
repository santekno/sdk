package pricingx

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/santekno/sdk/moneyx"
)

// troyOzPerGram is the conversion factor from troy ounces to grams.
const troyOzPerGram = "31.1034768"

// YahooGold fetches gold spot price via Yahoo Finance, converting to IDR per gram.
// It fetches GC=F (gold futures, USD/troy oz) and USDIDR=X (USD/IDR exchange rate),
// then computes: IDR/gram = goldUSD × usdIDR ÷ 31.1034768.
// The instrumentCode parameter is ignored — all gold tracks the same spot price.
type YahooGold struct {
	baseURL string
	hc      *http.Client
}

// YahooGoldOption configures a YahooGold provider.
type YahooGoldOption func(*YahooGold)

// WithYahooGoldBaseURL overrides the base URL (used in tests).
func WithYahooGoldBaseURL(u string) YahooGoldOption { return func(g *YahooGold) { g.baseURL = u } }

// WithYahooGoldHTTPClient sets the HTTP client.
func WithYahooGoldHTTPClient(hc *http.Client) YahooGoldOption {
	return func(g *YahooGold) { g.hc = hc }
}

// NewYahooGold builds a YahooGold provider.
func NewYahooGold(opts ...YahooGoldOption) *YahooGold {
	g := &YahooGold{
		baseURL: "https://query1.finance.yahoo.com",
		hc:      &http.Client{Timeout: 15 * time.Second},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

// Category implements Provider.
func (g *YahooGold) Category() string { return "gold" }

// GetPrice implements Provider. instrumentCode is ignored; always fetches GC=F + USDIDR=X.
func (g *YahooGold) GetPrice(ctx context.Context, _ string) (Price, error) {
	goldUSD, err := fetchYahooChart(ctx, g.hc, g.baseURL, "GC=F")
	if err != nil {
		return Price{}, fmt.Errorf("pricingx: gold spot (GC=F): %w", err)
	}
	usdIDR, err := fetchYahooChart(ctx, g.hc, g.baseURL, "USDIDR=X")
	if err != nil {
		return Price{}, fmt.Errorf("pricingx: gold USD/IDR (USDIDR=X): %w", err)
	}
	troy, _ := moneyx.Parse(troyOzPerGram)
	idrPerGram := goldUSD.Mul(usdIDR).Div(troy)
	return Price{Value: idrPerGram, AsOf: time.Now(), Source: "yahoo-gold"}, nil
}
