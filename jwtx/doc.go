// Package jwtx provides JWT signing and verification with explicit algorithm
// whitelisting. The "alg=none" attack is always rejected. Supported algorithms:
// HS256, HS512, RS256.
//
//	tok, _ := jwtx.Sign(jwtx.Claims{Subject: "u-42"}, jwtx.HS256([]byte("secret")))
//	claims, err := jwtx.Verify(tok, jwtx.HS256([]byte("secret")))
package jwtx
