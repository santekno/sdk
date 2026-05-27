package errx

// WithHTTPStatus sets the HTTP status code mapping and returns e for chaining.
func (e *AppError) WithHTTPStatus(status int) *AppError {
	e.HTTPStatus = status
	return e
}

// WithDetail adds a single key-value pair to the Details map and returns e for chaining.
func (e *AppError) WithDetail(key string, value any) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithDetails merges the given map into the Details map and returns e for chaining.
func (e *AppError) WithDetails(kv map[string]any) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]any, len(kv))
	}
	for k, v := range kv {
		e.Details[k] = v
	}
	return e
}
