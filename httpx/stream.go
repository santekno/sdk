package httpx

import (
	"context"
	"io"
	"net/http"
)

// GetStream performs a GET request and returns the response body for streaming.
// The caller MUST close the returned io.ReadCloser.
func (c *Client) GetStream(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolveURL(url), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// PostStream performs a POST with the supplied body reader and returns the
// response body for streaming. The caller MUST close the returned io.ReadCloser.
func (c *Client) PostStream(ctx context.Context, url string, body io.Reader) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.resolveURL(url), body)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
