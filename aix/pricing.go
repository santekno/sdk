package aix

// Pricing is the per-million-token price for input and output of a given model.
// Values are USD. Sourced from each provider's published pricing page; treat as
// a best-effort snapshot — refresh manually when providers change pricing.
type Pricing struct {
	// InputUSDPerMillion is the cost for 1,000,000 input tokens.
	InputUSDPerMillion float64
	// OutputUSDPerMillion is the cost for 1,000,000 output tokens.
	OutputUSDPerMillion float64
}

// pricingTable maps [Model] to [Pricing]. Last refreshed: 2026-06-01.
// Add new entries when a new model is added to [Model] constants.
//
// Source: https://platform.openai.com/docs/pricing
var pricingTable = map[Model]Pricing{
	ModelGPT4oMini: {
		InputUSDPerMillion:  0.15,
		OutputUSDPerMillion: 0.60,
	},
	ModelGPT4o: {
		InputUSDPerMillion:  5.00,
		OutputUSDPerMillion: 20.00,
	},
}

// EstimateUSD returns the estimated cost of a single Generate call using the
// model's pricing entry. Returns 0 if the model is not in the pricing table.
// Callers MUST NOT use this for billing — it is a best-effort observability
// signal only.
func EstimateUSD(model Model, inputTokens, outputTokens int) float64 {
	p, ok := pricingTable[model]
	if !ok {
		return 0
	}
	const million = 1_000_000.0
	return (float64(inputTokens)/million)*p.InputUSDPerMillion +
		(float64(outputTokens)/million)*p.OutputUSDPerMillion
}

// PricingFor returns the pricing entry for the given model and a bool indicating
// whether the model is known. Test-friendly accessor.
func PricingFor(model Model) (Pricing, bool) {
	p, ok := pricingTable[model]
	return p, ok
}
