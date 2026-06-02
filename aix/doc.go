// Package aix provides LLM provider helpers built on top of a pluggable
// Provider interface. The package is dependency-free: bring your own client
// (e.g. github.com/sashabaranov/go-openai for OpenAI, Anthropic SDK, etc.)
// and satisfy the interface.
//
// # Design
//
// aix defines a minimal [Provider] interface with a single Generate method
// taking a typed [GenerateRequest] and returning a typed [GenerateResponse].
// Provider adapters (OpenAI, Anthropic, ...) implement the interface; the
// [Client] wrapper layers on cost estimation, output validation, and
// observability hooks without touching the provider-specific code.
//
// This separation lets the consumer (e.g. apps/api) swap LLM providers
// without changing usecase code — a Phase 3+ requirement for the
// tools.santekno.com platform.
//
// # Usage
//
//	import "github.com/sashabaranov/go-openai"
//	import "github.com/santekno/sdk/aix"
//
//	rawClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
//	provider  := aix.NewOpenAIProvider(rawClient)
//	llm       := aix.NewClient(provider)
//
//	resp, err := llm.Generate(ctx, aix.GenerateRequest{
//	    Model:           aix.ModelGPT4oMini,
//	    SystemPrompt:    "You are a regex generator. Output ONLY the regex pattern.",
//	    UserMessage:     "Match an email address.",
//	    MaxOutputTokens: 256,
//	})
//	// resp.Output is the generated text; resp.CostUSD is a best-effort estimate.
//
// # Cost estimation
//
// [Client.Generate] populates [GenerateResponse.CostUSD] using the published
// per-model pricing table in pricing.go. Pricing is a best-effort snapshot;
// applications using these values for hard cost ceilings should refresh
// the table periodically.
//
// # Mocking in tests
//
// Implement [Provider] with an httptest.Server or a literal struct returning
// canned [GenerateResponse] values. The [Client] wrapper passes through
// unchanged, so tests assert against the deterministic mock output.
package aix
