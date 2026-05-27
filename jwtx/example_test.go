package jwtx_test

import (
	"fmt"
	"time"

	"github.com/santekno/sdk/jwtx"
)

func ExampleSign() {
	key := jwtx.HS256([]byte("supersecret"))
	claims := jwtx.Claims{
		Subject:   "user-123",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	token, err := jwtx.Sign(claims, key)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(token) > 0)
	// Output: true
}

func ExampleVerify() {
	key := jwtx.HS256([]byte("supersecret"))
	claims := jwtx.Claims{
		Subject:   "alice",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	token, _ := jwtx.Sign(claims, key)

	got, err := jwtx.Verify(token, key)
	if err != nil {
		panic(err)
	}
	fmt.Println(got.Subject)
	// Output: alice
}
