// Package testx provides test helper utilities for the Santekno Go SDK and
// applications built with it. Import only in test files.
//
// # Assertion Helpers
//
// Thin wrappers over standard testing.T that produce clear failure messages:
//
//	testx.AssertNoError(t, err)
//	testx.AssertEqual(t, got, want)
//	testx.AssertContains(t, haystack, needle)
//
// # Container Helpers
//
// Functions that spin up real database containers for integration tests via
// testcontainers-go. Requires Docker running locally.
//
//	dsn := testx.PostgresContainer(t)
//	addr := testx.RedisContainer(t)
package testx
