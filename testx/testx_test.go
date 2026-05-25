package testx_test

import (
	"errors"
	"testing"

	"github.com/santekno/sdk/testx"
)

func TestAssertNoError(t *testing.T) {
	testx.AssertNoError(t, nil)
}

func TestAssertEqual(t *testing.T) {
	testx.AssertEqual(t, 42, 42)
	testx.AssertEqual(t, "hello", "hello")
}

func TestAssertNotEqual(t *testing.T) {
	testx.AssertNotEqual(t, 1, 2)
}

func TestAssertContains(t *testing.T) {
	testx.AssertContains(t, "hello world", "world")
}

func TestAssertTrue(t *testing.T) {
	testx.AssertTrue(t, true)
}

func TestAssertFalse(t *testing.T) {
	testx.AssertFalse(t, false)
}

func TestAssertError(t *testing.T) {
	testx.AssertError(t, errors.New("some error"))
}

func TestAssertNil(t *testing.T) {
	testx.AssertNil(t, nil)
}
