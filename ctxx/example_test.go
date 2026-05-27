package ctxx_test

import (
	"context"
	"fmt"

	"github.com/santekno/sdk/ctxx"
)

func ExampleWithUserID() {
	ctx := ctxx.WithUserID(context.Background(), "user-42")
	fmt.Println(ctxx.UserID(ctx))
	// Output: user-42
}

func ExampleWithRequestID() {
	ctx := ctxx.WithRequestID(context.Background(), "req-abc")
	fmt.Println(ctxx.RequestID(ctx))
	// Output: req-abc
}
