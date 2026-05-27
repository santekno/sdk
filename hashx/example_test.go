package hashx_test

import (
	"fmt"

	"github.com/santekno/sdk/hashx"
)

func ExampleHashPassword() {
	hash, err := hashx.HashPassword("mysecretpassword")
	if err != nil {
		panic(err)
	}
	ok := hashx.VerifyPassword("mysecretpassword", hash)
	fmt.Println(ok)
	// Output: true
}

func ExampleVerifyPassword() {
	hash, _ := hashx.HashPassword("correct-horse")
	fmt.Println(hashx.VerifyPassword("correct-horse", hash))
	fmt.Println(hashx.VerifyPassword("wrong-horse", hash))
	// Output:
	// true
	// false
}

func ExampleConstantTimeEqual() {
	a := []byte("same")
	b := []byte("same")
	fmt.Println(hashx.ConstantTimeEqual(a, b))
	// Output: true
}
