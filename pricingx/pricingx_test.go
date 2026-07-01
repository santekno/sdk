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

var _ Provider = (*Yahoo)(nil)
var _ Provider = (*CoinGecko)(nil)
var _ Provider = (*ManualProvider)(nil)
