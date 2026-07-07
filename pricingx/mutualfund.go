package pricingx

// NewYahooMutualFund builds a Yahoo Finance NAV provider for mutual funds (T049).
// The instrumentCode should be the fund's Yahoo Finance ticker (e.g. "0P0001ABCD.JK").
// When Yahoo has no data for a fund, GetPrice returns an error and the pricing service
// marks the last cached NAV as stale (Constitution IV: graceful degradation).
func NewYahooMutualFund(opts ...YahooOption) *Yahoo {
	return NewYahoo("mutual_fund", opts...)
}
