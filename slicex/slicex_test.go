package slicex_test

import (
	"testing"

	"github.com/santekno/sdk/slicex"
)

func TestMap(t *testing.T) {
	got := slicex.Map([]int{1, 2, 3}, func(v, _ int) int { return v * 2 })
	want := []int{2, 4, 6}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("Map[%d] = %d, want %d", i, got[i], v)
		}
	}
}

func TestFilter(t *testing.T) {
	got := slicex.Filter([]int{1, 2, 3, 4}, func(v, _ int) bool { return v%2 == 0 })
	if len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Fatalf("Filter = %v, want [2 4]", got)
	}
}

func TestReduce(t *testing.T) {
	sum := slicex.Reduce([]int{1, 2, 3, 4}, 0, func(acc, v, _ int) int { return acc + v })
	if sum != 10 {
		t.Fatalf("Reduce = %d, want 10", sum)
	}
}

func TestGroupBy(t *testing.T) {
	m := slicex.GroupBy([]int{1, 2, 3, 4}, func(v int) string {
		if v%2 == 0 {
			return "even"
		}
		return "odd"
	})
	if len(m["even"]) != 2 || len(m["odd"]) != 2 {
		t.Fatalf("GroupBy = %v", m)
	}
}

func TestChunk(t *testing.T) {
	chunks := slicex.Chunk([]int{1, 2, 3, 4, 5}, 2)
	if len(chunks) != 3 {
		t.Fatalf("Chunk len = %d, want 3", len(chunks))
	}
}

func TestUniq(t *testing.T) {
	got := slicex.Uniq([]int{1, 2, 2, 3, 1})
	if len(got) != 3 {
		t.Fatalf("Uniq = %v, want [1 2 3]", got)
	}
}

func TestContains(t *testing.T) {
	if !slicex.Contains([]int{1, 2, 3}, 2) {
		t.Error("Contains(2) should be true")
	}
	if slicex.Contains([]int{1, 2, 3}, 9) {
		t.Error("Contains(9) should be false")
	}
}

func TestFind(t *testing.T) {
	v, ok := slicex.Find([]int{1, 2, 3}, func(v int) bool { return v > 1 })
	if !ok || v != 2 {
		t.Fatalf("Find = (%d, %v), want (2, true)", v, ok)
	}
	_, ok = slicex.Find([]int{1, 2}, func(v int) bool { return v > 9 })
	if ok {
		t.Error("Find should return false when nothing found")
	}
}

func TestFlatten(t *testing.T) {
	got := slicex.Flatten([][]int{{1, 2}, {3, 4}, {5}})
	if len(got) != 5 || got[4] != 5 {
		t.Fatalf("Flatten = %v", got)
	}
}

func TestReverse(t *testing.T) {
	got := slicex.Reverse([]int{1, 2, 3})
	if got[0] != 3 || got[2] != 1 {
		t.Fatalf("Reverse = %v, want [3 2 1]", got)
	}
}

func TestMapToMap(t *testing.T) {
	m := slicex.MapToMap([]int{1, 2, 3}, func(v int) (string, int) {
		return string(rune('a' + v - 1)), v * 10
	})
	if m["a"] != 10 || m["b"] != 20 || m["c"] != 30 {
		t.Fatalf("MapToMap = %v", m)
	}
}

func TestParallelMap(t *testing.T) {
	got := slicex.ParallelMap([]int{1, 2, 3, 4}, func(v, _ int) int { return v * v }, 2)
	want := []int{1, 4, 9, 16}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("ParallelMap[%d] = %d, want %d", i, got[i], v)
		}
	}
}

func BenchmarkMap(b *testing.B) {
	s := make([]int, 1024)
	for i := range s {
		s[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slicex.Map(s, func(v, _ int) int { return v * 2 })
	}
}

func BenchmarkFilter(b *testing.B) {
	s := make([]int, 1024)
	for i := range s {
		s[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slicex.Filter(s, func(v, _ int) bool { return v%2 == 0 })
	}
}

func ExampleMap() {
	doubled := slicex.Map([]int{1, 2, 3}, func(v, _ int) int { return v * 2 })
	_ = doubled // [2, 4, 6]
}
