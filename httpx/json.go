package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const contentTypeJSON = "application/json"

// GetJSON performs a GET request and decodes the JSON response into out.
func (c *Client) GetJSON(ctx context.Context, url string, out any) error {
	return c.doJSON(ctx, http.MethodGet, url, nil, out)
}

// PostJSON performs a POST with body encoded as JSON, then decodes the JSON
// response into out. out may be nil to discard the response body.
func (c *Client) PostJSON(ctx context.Context, url string, body, out any) error {
	return c.doJSON(ctx, http.MethodPost, url, body, out)
}

// PutJSON performs a PUT with body encoded as JSON, then decodes the response.
func (c *Client) PutJSON(ctx context.Context, url string, body, out any) error {
	return c.doJSON(ctx, http.MethodPut, url, body, out)
}

// PatchJSON performs a PATCH with body encoded as JSON, then decodes the response.
func (c *Client) PatchJSON(ctx context.Context, url string, body, out any) error {
	return c.doJSON(ctx, http.MethodPatch, url, body, out)
}

// DeleteJSON performs a DELETE and optionally decodes the JSON response into out.
func (c *Client) DeleteJSON(ctx context.Context, url string, out any) error {
	return c.doJSON(ctx, http.MethodDelete, url, nil, out)
}

func (c *Client) doJSON(ctx context.Context, method, url string, body, out any) error {
	fullURL := c.resolveURL(url)

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("santekno/httpx: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("santekno/httpx: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", contentTypeJSON)
	}
	req.Header.Set("Accept", contentTypeJSON)

	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		excerpt := strings.TrimSpace(string(body))
		return fmt.Errorf("%w: status %d: %s", ErrNon2xx, resp.StatusCode, excerpt)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("santekno/httpx: decode response: %w", err)
	}
	return nil
}

// resolveURL prepends BaseURL to relative URLs.
func (c *Client) resolveURL(u string) string {
	if c.cfg.BaseURL == "" {
		return u
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return strings.TrimRight(c.cfg.BaseURL, "/") + "/" + strings.TrimLeft(u, "/")
}
