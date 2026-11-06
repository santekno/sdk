// Package jwtx provides JWT signing and verification with explicit algorithm
// whitelisting. The "alg=none" attack is always rejected. Supported algorithms:
// HS256, HS512, RS256, and ES256 (verification only).
//
//	tok, _ := jwtx.Sign(jwtx.Claims{Subject: "u-42"}, jwtx.HS256([]byte("secret")))
//	claims, err := jwtx.Verify(tok, jwtx.HS256([]byte("secret")))
//
// Tokens issued by a third party are verified against its published JWKS with
// VerifyWithKeySet, which is stricter than Verify. See its documentation.
//
//	client, err := jwtx.NewJWKSClient("https://issuer.example/.well-known/jwks.json")
//	claims, err := jwtx.VerifyWithKeySet(tok, client,
//		[]jwtx.Algorithm{jwtx.AlgES256, jwtx.AlgRS256},
//		jwtx.WithLeeway(30*time.Second), jwtx.WithContext(ctx))
//
// Callers must still check "iss" and "aud": VerifyWithKeySet does not.
//
// JWKSClient keeps using its keys while the issuer is unreachable, for at most
// 24 hours after the last successful fetch (WithMaxStale). After that it returns
// ErrKeySetUnavailable, which callers should report as a server-side failure.
package jwtx
