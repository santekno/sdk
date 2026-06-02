package aix

import (
	"context"
	"errors"
	"fmt"
)

// Common errors returned by [Provider] adapters and [Client].
var (
	// ErrEmptyPrompt is returned when GenerateRequest.UserMessage is blank.
	ErrEmptyPrompt = errors.New("aix: user message is empty")
	// ErrProviderUnavailable wraps any network/upstream error from the provider.
	ErrProviderUnavailable = errors.New("aix: provider unavailable")
	// ErrProviderRefused indicates the provider applied a content-safety filter.
	ErrProviderRefused = errors.New("aix: provider refused request")
	// ErrOutputValidation is returned when an OutputValidator rejects the response.
	ErrOutputValidation = errors.New("aix: output validation failed")
)

// Model identifies a provider's model. Use one of the package constants where
// possible (e.g. [ModelGPT4oMini], [ModelGPT4o]); arbitrary strings are also
// accepted for forward compatibility with new providers/models.
type Model string

// Known model identifiers. Add new entries here as providers ship new models;
// the pricing table in pricing.go MUST also be updated.
const (
	ModelGPT4oMini Model = "gpt-4o-mini"
	ModelGPT4o     Model = "gpt-4o"
)

// GenerateRequest is the provider-agnostic input shape.
type GenerateRequest struct {
	// Model is the provider-specific model identifier.
	Model Model
	// SystemPrompt is the system / role-prelude message. May be empty.
	SystemPrompt string
	// UserMessage is the user-facing prompt content. Required; non-blank.
	UserMessage string
	// MaxOutputTokens caps the response length. 0 = provider default.
	MaxOutputTokens int
	// Temperature tunes randomness. 0 = provider default (typically 1.0).
	Temperature float32
}

// GenerateResponse is the provider-agnostic output shape.
type GenerateResponse struct {
	// Output is the generated text content.
	Output string
	// Model echoes the model that produced this response (provider may
	// override the requested model — e.g. version pinning).
	Model Model
	// InputTokens is the prompt token count reported by the provider.
	InputTokens int
	// OutputTokens is the completion token count reported by the provider.
	OutputTokens int
	// CostUSD is a best-effort cost estimate populated by [Client.Generate]
	// from the pricing table. Zero when the model is not in the pricing table.
	CostUSD float64
}

// Provider is the minimal interface any LLM adapter must satisfy.
// Implementations are expected to translate the provider-agnostic
// [GenerateRequest] into the upstream API call and back.
type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
	// Name returns a short identifier of the provider (e.g. "openai",
	// "anthropic"). Used for telemetry and error messages.
	Name() string
}

// OutputValidator is an optional hook that inspects the generated text before
// it leaves [Client.Generate]. A non-nil error wraps [ErrOutputValidation].
type OutputValidator func(output string) error

// Client wraps a [Provider] with cost estimation and optional validation.
type Client struct {
	provider  Provider
	validator OutputValidator
}

// NewClient returns a Client wrapping the given provider. provider must not be nil.
func NewClient(provider Provider) *Client {
	if provider == nil {
		panic("aix.NewClient: provider must not be nil")
	}
	return &Client{provider: provider}
}

// WithValidator attaches an OutputValidator to the client. The validator
// runs after the provider returns and before the response leaves Generate.
// Returns the receiver so calls can chain.
func (c *Client) WithValidator(v OutputValidator) *Client {
	c.validator = v
	return c
}

// Generate calls the underlying provider and returns the response with
// CostUSD populated. Validation failures wrap [ErrOutputValidation].
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	if req.UserMessage == "" {
		return GenerateResponse{}, ErrEmptyPrompt
	}
	resp, err := c.provider.Generate(ctx, req)
	if err != nil {
		return resp, err
	}
	if c.validator != nil {
		if vErr := c.validator(resp.Output); vErr != nil {
			return resp, fmt.Errorf("%w: %w", ErrOutputValidation, vErr)
		}
	}
	// Populate cost estimate (zero if model unknown).
	resp.CostUSD = EstimateUSD(resp.Model, resp.InputTokens, resp.OutputTokens)
	return resp, nil
}

// Name returns the underlying provider's name. Convenience accessor for
// telemetry tags.
func (c *Client) Name() string {
	return c.provider.Name()
}
