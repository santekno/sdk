package aix_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/santekno/sdk/aix"
)

// fakeProvider is a deterministic test double for [aix.Provider].
type fakeProvider struct {
	name    string
	resp    aix.GenerateResponse
	err     error
	lastReq aix.GenerateRequest
}

func (f *fakeProvider) Generate(_ context.Context, req aix.GenerateRequest) (aix.GenerateResponse, error) {
	f.lastReq = req
	return f.resp, f.err
}

func (f *fakeProvider) Name() string {
	if f.name == "" {
		return "fake"
	}
	return f.name
}

// fakeOpenAIClient implements aix.openAIRawClient (unexported). To exercise
// the OpenAI adapter we use the public NewOpenAIProvider against this fake.
type fakeOpenAIClient struct {
	req    aix.OpenAIChatRequest
	resp   aix.OpenAIChatResponse
	err    error
	called bool
}

func (f *fakeOpenAIClient) CreateChatCompletion(_ context.Context, req aix.OpenAIChatRequest) (aix.OpenAIChatResponse, error) {
	f.req = req
	f.called = true
	return f.resp, f.err
}

// ============================================================================
// Client tests
// ============================================================================

func TestClient_Generate_HappyPath(t *testing.T) {
	fp := &fakeProvider{
		resp: aix.GenerateResponse{
			Output:       "^[a-z]+@[a-z]+$",
			Model:        aix.ModelGPT4oMini,
			InputTokens:  100,
			OutputTokens: 50,
		},
	}
	client := aix.NewClient(fp)

	resp, err := client.Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "match email",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "^[a-z]+@[a-z]+$" {
		t.Errorf("Output mismatch: %q", resp.Output)
	}
	if resp.CostUSD == 0 {
		t.Errorf("CostUSD should be populated for known model")
	}
}

func TestClient_Generate_EmptyMessage(t *testing.T) {
	client := aix.NewClient(&fakeProvider{})
	_, err := client.Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "",
	})
	if !errors.Is(err, aix.ErrEmptyPrompt) {
		t.Errorf("expected ErrEmptyPrompt, got %v", err)
	}
}

func TestClient_Generate_ProviderError(t *testing.T) {
	fp := &fakeProvider{err: aix.ErrProviderUnavailable}
	client := aix.NewClient(fp)
	_, err := client.Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "x",
	})
	if !errors.Is(err, aix.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestClient_WithValidator_Pass(t *testing.T) {
	fp := &fakeProvider{
		resp: aix.GenerateResponse{Output: "valid"},
	}
	client := aix.NewClient(fp).WithValidator(func(out string) error {
		if out == "" {
			return errors.New("empty")
		}
		return nil
	})
	_, err := client.Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "x",
	})
	if err != nil {
		t.Errorf("expected validator to pass, got %v", err)
	}
}

func TestClient_WithValidator_Fail(t *testing.T) {
	fp := &fakeProvider{
		resp: aix.GenerateResponse{Output: "bad"},
	}
	client := aix.NewClient(fp).WithValidator(func(out string) error {
		return errors.New("rejected")
	})
	_, err := client.Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "x",
	})
	if !errors.Is(err, aix.ErrOutputValidation) {
		t.Errorf("expected ErrOutputValidation, got %v", err)
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("expected wrapped validator error in message, got %v", err)
	}
}

func TestClient_Name(t *testing.T) {
	fp := &fakeProvider{name: "fake-llm"}
	if got := aix.NewClient(fp).Name(); got != "fake-llm" {
		t.Errorf("Name() = %q, want %q", got, "fake-llm")
	}
}

func TestNewClient_NilPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on nil provider")
		}
	}()
	aix.NewClient(nil)
}

// ============================================================================
// OpenAI adapter tests
// ============================================================================

func TestOpenAIProvider_Generate_HappyPath(t *testing.T) {
	fake := &fakeOpenAIClient{
		resp: aix.OpenAIChatResponse{
			Choices: []aix.OpenAIChoice{
				{Message: aix.OpenAIMessage{Role: "assistant", Content: "hello world"}, FinishReason: "stop"},
			},
			Usage: aix.OpenAIUsage{PromptTokens: 12, CompletionTokens: 5, TotalTokens: 17},
			Model: "gpt-4o-mini-2024-07-18",
		},
	}
	provider := aix.NewOpenAIProvider(fake)

	resp, err := provider.Generate(context.Background(), aix.GenerateRequest{
		Model:           aix.ModelGPT4oMini,
		SystemPrompt:    "You are a poet.",
		UserMessage:     "Write a haiku.",
		MaxOutputTokens: 256,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fake.called {
		t.Fatal("upstream client not invoked")
	}
	if len(fake.req.Messages) != 2 {
		t.Fatalf("expected 2 messages (system+user), got %d", len(fake.req.Messages))
	}
	if fake.req.Messages[0].Role != "system" || fake.req.Messages[0].Content != "You are a poet." {
		t.Errorf("system message mismatch: %+v", fake.req.Messages[0])
	}
	if fake.req.Messages[1].Role != "user" || fake.req.Messages[1].Content != "Write a haiku." {
		t.Errorf("user message mismatch: %+v", fake.req.Messages[1])
	}
	if fake.req.MaxTokens != 256 {
		t.Errorf("MaxTokens = %d, want 256", fake.req.MaxTokens)
	}

	if resp.Output != "hello world" {
		t.Errorf("Output = %q", resp.Output)
	}
	if resp.InputTokens != 12 || resp.OutputTokens != 5 {
		t.Errorf("usage mismatch: %+v", resp)
	}
	if resp.Model != "gpt-4o-mini-2024-07-18" {
		t.Errorf("Model echo mismatch: %q", resp.Model)
	}
}

func TestOpenAIProvider_Generate_OmitsSystemWhenBlank(t *testing.T) {
	fake := &fakeOpenAIClient{
		resp: aix.OpenAIChatResponse{
			Choices: []aix.OpenAIChoice{{Message: aix.OpenAIMessage{Content: "ok"}, FinishReason: "stop"}},
		},
	}
	provider := aix.NewOpenAIProvider(fake)
	_, err := provider.Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "x",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.req.Messages) != 1 {
		t.Errorf("expected 1 message (no system), got %d", len(fake.req.Messages))
	}
	if fake.req.Messages[0].Role != "user" {
		t.Errorf("expected user role, got %q", fake.req.Messages[0].Role)
	}
}

func TestOpenAIProvider_Generate_ContentFilter(t *testing.T) {
	fake := &fakeOpenAIClient{
		resp: aix.OpenAIChatResponse{
			Choices: []aix.OpenAIChoice{{Message: aix.OpenAIMessage{Content: ""}, FinishReason: "content_filter"}},
		},
	}
	_, err := aix.NewOpenAIProvider(fake).Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "x",
	})
	if !errors.Is(err, aix.ErrProviderRefused) {
		t.Errorf("expected ErrProviderRefused, got %v", err)
	}
}

func TestOpenAIProvider_Generate_EmptyChoices(t *testing.T) {
	fake := &fakeOpenAIClient{resp: aix.OpenAIChatResponse{Choices: nil}}
	_, err := aix.NewOpenAIProvider(fake).Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "x",
	})
	if !errors.Is(err, aix.ErrProviderRefused) {
		t.Errorf("expected ErrProviderRefused on empty choices, got %v", err)
	}
}

func TestOpenAIProvider_Generate_UpstreamTimeout(t *testing.T) {
	fake := &fakeOpenAIClient{err: errors.New("context deadline exceeded")}
	_, err := aix.NewOpenAIProvider(fake).Generate(context.Background(), aix.GenerateRequest{
		Model:       aix.ModelGPT4oMini,
		UserMessage: "x",
	})
	if !errors.Is(err, aix.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable on timeout, got %v", err)
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	fake := &fakeOpenAIClient{}
	if got := aix.NewOpenAIProvider(fake).Name(); got != "openai" {
		t.Errorf("Name() = %q, want %q", got, "openai")
	}
}

func TestNewOpenAIProvider_NilPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on nil client")
		}
	}()
	aix.NewOpenAIProvider(nil)
}

// ============================================================================
// Pricing tests
// ============================================================================

func TestEstimateUSD_KnownModel(t *testing.T) {
	// gpt-4o-mini: $0.15/$0.60 per million tokens
	// 1000 input + 1000 output = 1000*0.15/1M + 1000*0.60/1M = 0.00015 + 0.0006 = 0.00075
	cost := aix.EstimateUSD(aix.ModelGPT4oMini, 1000, 1000)
	want := 0.00075
	if cost < want*0.99 || cost > want*1.01 {
		t.Errorf("EstimateUSD = %.6f, want ~%.6f", cost, want)
	}
}

func TestEstimateUSD_UnknownModel(t *testing.T) {
	if got := aix.EstimateUSD(aix.Model("nonexistent"), 1000, 1000); got != 0 {
		t.Errorf("expected 0 for unknown model, got %f", got)
	}
}

func TestEstimateUSD_ZeroTokens(t *testing.T) {
	if got := aix.EstimateUSD(aix.ModelGPT4o, 0, 0); got != 0 {
		t.Errorf("expected 0 for zero tokens, got %f", got)
	}
}

func TestPricingFor_Known(t *testing.T) {
	p, ok := aix.PricingFor(aix.ModelGPT4o)
	if !ok {
		t.Fatal("expected ModelGPT4o to be in pricing table")
	}
	if p.InputUSDPerMillion <= 0 || p.OutputUSDPerMillion <= 0 {
		t.Errorf("pricing values look wrong: %+v", p)
	}
}

func TestPricingFor_Unknown(t *testing.T) {
	if _, ok := aix.PricingFor(aix.Model("nonexistent")); ok {
		t.Error("expected unknown model to return ok=false")
	}
}
