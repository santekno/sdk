// Package pricingx provides a swappable market-price Provider abstraction and
// concrete providers (Yahoo Finance for stocks/indices, CoinGecko for crypto,
// plus a Manual provider). Prices are exact decimals (moneyx) — the source's
// numeric text is parsed directly, never via float (financial correctness).
package pricingx

import (
	"context"
	"fmt"
	"time"

	"github.com/santekno/sdk/moneyx"
)

// Price is a market price with its source and freshness.
type Price struct {
	Value  moneyx.Dec
	AsOf   time.Time
	Source string
}

// Provider fetches the current price for an instrument in a category.
type Provider interface {
	Category() string // "gold" | "stock" | "mutual_fund" | "crypto" | "benchmark"
	GetPrice(ctx context.Context, instrumentCode string) (Price, error)
}

// Registry selects a Provider by category.
type Registry struct {
	byCategory map[string]Provider
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry { return &Registry{byCategory: map[string]Provider{}} }

// Register adds (or replaces) the provider for its category.
func (r *Registry) Register(p Provider) { r.byCategory[p.Category()] = p }

// Get returns the provider for a category.
func (r *Registry) Get(category string) (Provider, bool) {
	p, ok := r.byCategory[category]
	return p, ok
}

// Price fetches a price for (category, instrumentCode) via the registered provider.
func (r *Registry) Price(ctx context.Context, category, instrumentCode string) (Price, error) {
	p, ok := r.byCategory[category]
	if !ok {
		return Price{}, fmt.Errorf("pricingx: no provider for category %q", category)
	}
	return p.GetPrice(ctx, instrumentCode)
}

// ManualProvider serves prices from an in-memory map — used for manual valuation
// (P2P, crowdfunding, non-tradeable SBN) and in tests.
type ManualProvider struct {
	category string
	prices   map[string]moneyx.Dec
	now      func() time.Time
}

// NewManualProvider builds a manual provider for a category.
func NewManualProvider(category string) *ManualProvider {
	return &ManualProvider{category: category, prices: map[string]moneyx.Dec{}, now: time.Now}
}

// Set records a price for an instrument.
func (m *ManualProvider) Set(instrumentCode string, price moneyx.Dec) {
	m.prices[instrumentCode] = price
}

// Category implements Provider.
func (m *ManualProvider) Category() string { return m.category }

// GetPrice implements Provider.
func (m *ManualProvider) GetPrice(_ context.Context, instrumentCode string) (Price, error) {
	p, ok := m.prices[instrumentCode]
	if !ok {
		return Price{}, fmt.Errorf("pricingx: no manual price for %q", instrumentCode)
	}
	return Price{Value: p, AsOf: m.now(), Source: "manual"}, nil
}
