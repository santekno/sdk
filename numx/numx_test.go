package numx_test

import (
	"testing"

	"github.com/santekno/sdk/numx"
)

func TestMinMax(t *testing.T) {
	if numx.Min(3, 5) != 3 {
		t.Error("Min failed")
	}
	if numx.Max(3, 5) != 5 {
		t.Error("Max failed")
	}
}

func TestClamp(t *testing.T) {
	if numx.Clamp(15, 0, 10) != 10 {
		t.Error("Clamp upper bound")
	}
	if numx.Clamp(-5, 0, 10) != 0 {
		t.Error("Clamp lower bound")
	}
	if numx.Clamp(5, 0, 10) != 5 {
		t.Error("Clamp pass-through")
	}
}

func TestAbs(t *testing.T) {
	if numx.Abs(-5) != 5 {
		t.Error("Abs(-5)")
	}
	if numx.Abs(3.5) != 3.5 {
		t.Error("Abs(3.5)")
	}
}

func TestSumAverage(t *testing.T) {
	if numx.Sum([]int{1, 2, 3, 4}) != 10 {
		t.Error("Sum")
	}
	if numx.Average([]int{2, 4, 6}) != 4.0 {
		t.Error("Average")
	}
	if numx.Average([]int{}) != 0 {
		t.Error("Average empty")
	}
}

func TestRound(t *testing.T) {
	if numx.Round(3.5) != 4 {
		t.Error("Round(3.5)")
	}
	if got := numx.RoundToDecimal(3.14159, 2); got != 3.14 {
		t.Errorf("RoundToDecimal = %v", got)
	}
}
