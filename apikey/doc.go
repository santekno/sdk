// Package apikey provides API key lifecycle helpers: generation with a
// prefix, SHA-256 hashing for at-rest storage, and constant-time verification.
//
//	plaintext, hash, _ := apikey.Generate("sk_live")
//	// store hash in the database; show plaintext to the user once
//	ok := apikey.Verify(plaintext, hash)
package apikey
