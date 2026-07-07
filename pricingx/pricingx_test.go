package pricingx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/santekno/sdk/moneyx"
)

func TestManualProvider(t *testing.T) {
	m := NewManualProvider("p2p")
	m.Set("RD-001", moneyx.MustParse("1000000"))
	p, err := m.GetPrice(context.Background(), "RD-001")
	if err != nil {
		t.Fatal(err)
	}
	if p.Value.StringFixed(0) != "1000000" || p.Source != "manual" {
		t.Errorf("price = %s/%s, want 1000000/manual", p.Value.StringFixed(0), p.Source)
	}
	if _, err := m.GetPrice(context.Background(), "unknown"); err == nil {
		t.Error("expected error for unknown instrument")
	}
}

func TestYahoo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v8/finance/chart/BBCA.JK") {
			http.Error(w, "bad", http.StatusNotFound)
			return
		}
		io.WriteString(w, `{"chart":{"result":[{"meta":{"regularMarketPrice":9525.5,"currency":"IDR","symbol":"BBCA.JK"}}],"error":null}}`)
	}))
	defer srv.Close()

	y := NewYahoo("stock", WithYahooBaseURL(srv.URL), WithYahooHTTPClient(srv.Client()))
	if y.Category() != "stock" {
		t.Fatalf("category = %s", y.Category())
	}
	p, err := y.GetPrice(context.Background(), "BBCA.JK")
	if err != nil {
		t.Fatalf("get price: %v", err)
	}
	if p.Value.StringFixed(2) != "9525.50" || p.Source != "yahoo" {
		t.Errorf("price = %s/%s, want 9525.50/yahoo", p.Value.StringFixed(2), p.Source)
	}
}

func TestCoinGecko(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"bitcoin":{"idr":1650000000}}`)
	}))
	defer srv.Close()

	c := NewCoinGecko(WithCoinGeckoBaseURL(srv.URL), WithCoinGeckoHTTPClient(srv.Client()))
	p, err := c.GetPrice(context.Background(), "bitcoin")
	if err != nil {
		t.Fatalf("get price: %v", err)
	}
	if p.Value.StringFixed(0) != "1650000000" || p.Source != "coingecko" {
		t.Errorf("price = %s/%s, want 1650000000/coingecko", p.Value.StringFixed(0), p.Source)
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	m := NewManualProvider("gold")
	m.Set("EMAS-ANTAM", moneyx.MustParse("1365000"))
	reg.Register(m)

	p, err := reg.Price(context.Background(), "gold", "EMAS-ANTAM")
	if err != nil {
		t.Fatal(err)
	}
	if p.Value.StringFixed(0) != "1365000" {
		t.Errorf("registry price = %s, want 1365000", p.Value.StringFixed(0))
	}
	if _, err := reg.Price(context.Background(), "stock", "X"); err == nil {
		t.Error("expected error for unregistered category")
	}
}

func TestYahooGold(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "GC=F"):
			io.WriteString(w, `{"chart":{"result":[{"meta":{"regularMarketPrice":3000,"currency":"USD","symbol":"GC=F"}}],"error":null}}`)
		case strings.Contains(r.URL.Path, "USDIDR=X"):
			io.WriteString(w, `{"chart":{"result":[{"meta":{"regularMarketPrice":16000,"currency":"IDR","symbol":"USDIDR=X"}}],"error":null}}`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	g := NewYahooGold(WithYahooGoldBaseURL(srv.URL), WithYahooGoldHTTPClient(srv.Client()))
	if g.Category() != "gold" {
		t.Fatalf("category = %s, want gold", g.Category())
	}
	p, err := g.GetPrice(context.Background(), "")
	if err != nil {
		t.Fatalf("GetPrice: %v", err)
	}
	if p.Source != "yahoo-gold" {
		t.Errorf("source = %s, want yahoo-gold", p.Source)
	}
	// 3000 USD/oz * 16000 IDR/USD / 31.1034768 g/oz ≈ 1,543,952 IDR/gram
	threshold, _ := moneyx.Parse("1000000")
	if p.Value.Cmp(threshold) < 0 {
		t.Errorf("price %s too low, expected > 1000000 IDR/gram", p.Value.StringFixed(0))
	}
}

func TestYahooMutualFund(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"chart":{"result":[{"meta":{"regularMarketPrice":1250,"currency":"IDR","symbol":"0P0001.JK"}}],"error":null}}`)
	}))
	defer srv.Close()

	mf := NewYahooMutualFund(WithYahooBaseURL(srv.URL), WithYahooHTTPClient(srv.Client()))
	if mf.Category() != "mutual_fund" {
		t.Fatalf("category = %s, want mutual_fund", mf.Category())
	}
	p, err := mf.GetPrice(context.Background(), "0P0001.JK")
	if err != nil {
		t.Fatalf("GetPrice: %v", err)
	}
	if p.Value.StringFixed(0) != "1250" || p.Source != "yahoo" {
		t.Errorf("price = %s/%s, want 1250/yahoo", p.Value.StringFixed(0), p.Source)
	}
}

var _ Provider = (*Yahoo)(nil)
var _ Provider = (*CoinGecko)(nil)
var _ Provider = (*ManualProvider)(nil)
var _ Provider = (*YahooGold)(nil)
