package aix

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// openAIRawClient is the minimal subset of the OpenAI SDK that this adapter
// uses. Implementations include `*openai.Client` from
// github.com/sashabaranov/go-openai. Test-time mocks satisfy this interface
// directly without importing the real SDK.
//
// The shape mirrors the upstream client's CreateChatCompletion method.
type openAIRawClient interface {
	CreateChatCompletion(ctx context.Context, req OpenAIChatRequest) (OpenAIChatResponse, error)
}

// OpenAIChatRequest is the minimal request shape we pass to the OpenAI client.
// Fields match the upstream sashabaranov/go-openai struct so callers can adapt
// trivially. Defined here so the SDK package has no upstream dependency.
type OpenAIChatRequest struct {
	Model     string
	Messages  []OpenAIMessage
	MaxTokens int
	// Pointer so we can distinguish "0 means default" from "0 means greedy".
	Temperature *float32
}

// OpenAIMessage is the single message in a chat request.
type OpenAIMessage struct {
	Role    string
	Content string
}

// OpenAIChatResponse is the minimal response shape.
type OpenAIChatResponse struct {
	Choices []OpenAIChoice
	Usage   OpenAIUsage
	Model   string
}

// OpenAIChoice is one completion choice.
type OpenAIChoice struct {
	Message      OpenAIMessage
	FinishReason string // "stop", "length", "content_filter", etc.
}

// OpenAIUsage is the token-count summary.
type OpenAIUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// openAIProvider implements [Provider] by delegating to an openAIRawClient.
type openAIProvider struct {
	rc openAIRawClient
}

// NewOpenAIProvider wraps the given raw OpenAI client in a [Provider].
// The raw client must implement [openAIRawClient]; the official
// github.com/sashabaranov/go-openai client satisfies this once adapted via
// a small shim (see examples_test.go).
//
// rc must not be nil.
func NewOpenAIProvider(rc openAIRawClient) Provider {
	if rc == nil {
		panic("aix.NewOpenAIProvider: rc must not be nil")
	}
	return &openAIProvider{rc: rc}
}

// Name returns "openai".
func (p *openAIProvider) Name() string { return "openai" }

// Generate translates a [GenerateRequest] into an OpenAI chat-completion call
// and maps the response back to a [GenerateResponse].
func (p *openAIProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	messages := make([]OpenAIMessage, 0, 2)
	if req.SystemPrompt != "" {
		messages = append(messages, OpenAIMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, OpenAIMessage{Role: "user", Content: req.UserMessage})

	raw := OpenAIChatRequest{
		Model:     string(req.Model),
		Messages:  messages,
		MaxTokens: req.MaxOutputTokens,
	}
	if req.Temperature > 0 {
		t := req.Temperature
		raw.Temperature = &t
	}

	resp, err := p.rc.CreateChatCompletion(ctx, raw)
	if err != nil {
		return GenerateResponse{}, mapOpenAIError(err)
	}
	if len(resp.Choices) == 0 {
		return GenerateResponse{}, fmt.Errorf("%w: empty choices", ErrProviderRefused)
	}
	choice := resp.Choices[0]
	if choice.FinishReason == "content_filter" {
		return GenerateResponse{}, fmt.Errorf("%w: content_filter", ErrProviderRefused)
	}

	return GenerateResponse{
		Output:       choice.Message.Content,
		Model:        Model(resp.Model),
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}, nil
}

// mapOpenAIError normalises common upstream error patterns into the package's
// sentinel errors. Best-effort: anything unrecognised becomes
// [ErrProviderUnavailable].
func mapOpenAIError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "content_filter"),
		strings.Contains(msg, "policy"),
		strings.Contains(msg, "refused"):
		return fmt.Errorf("%w: %s", ErrProviderRefused, err.Error())
	case strings.Contains(msg, "timeout"),
		strings.Contains(msg, "deadline"),
		strings.Contains(msg, "503"),
		strings.Contains(msg, "502"),
		strings.Contains(msg, "connection refused"):
		return fmt.Errorf("%w: %s", ErrProviderUnavailable, err.Error())
	default:
		return fmt.Errorf("%w: %s", ErrProviderUnavailable, err.Error())
	}
}

// Compile-time assertion that openAIProvider satisfies Provider.
var _ Provider = (*openAIProvider)(nil)

// Unused but kept for callers who want to type-assert against http.RoundTripper
// behaviour in the underlying transport (rare).
var _ = errors.New
var _ http.Header
