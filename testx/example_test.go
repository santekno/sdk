package testx_test

import (
	"fmt"
	"testing"

	"github.com/santekno/sdk/testx"
)

func ExampleAssertEqual() {
	t := &testing.T{}
	testx.AssertEqual(t, 42, 42)
	fmt.Println("equal!")
	// Output: equal!
}

func ExampleAssertContains() {
	t := &testing.T{}
	testx.AssertContains(t, "hello world", "world")
	fmt.Println("contains!")
	// Output: contains!
}
