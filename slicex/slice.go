package slicex

import "sync"

// Map applies fn to each element of s and returns the results.
func Map[T, R any](s []T, fn func(T, int) R) []R {
	out := make([]R, len(s))
	for i, v := range s {
		out[i] = fn(v, i)
	}
	return out
}

// Filter returns elements of s for which fn returns true.
func Filter[T any](s []T, fn func(T, int) bool) []T {
	out := make([]T, 0, len(s))
	for i, v := range s {
		if fn(v, i) {
			out = append(out, v)
		}
	}
	return out
}

// Reduce reduces s to a single value starting from initial.
func Reduce[T, R any](s []T, initial R, fn func(R, T, int) R) R {
	acc := initial
	for i, v := range s {
		acc = fn(acc, v, i)
	}
	return acc
}

// GroupBy groups elements of s by the key returned by fn.
func GroupBy[T any, K comparable](s []T, fn func(T) K) map[K][]T {
	m := make(map[K][]T)
	for _, v := range s {
		k := fn(v)
		m[k] = append(m[k], v)
	}
	return m
}

// Chunk splits s into chunks of at most size n.
func Chunk[T any](s []T, n int) [][]T {
	if n <= 0 {
		return nil
	}
	var chunks [][]T
	for i := 0; i < len(s); i += n {
		end := i + n
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

// Uniq returns a new slice with duplicate elements removed, preserving order.
func Uniq[T comparable](s []T) []T {
	seen := make(map[T]struct{}, len(s))
	out := make([]T, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Contains reports whether s contains elem.
func Contains[T comparable](s []T, elem T) bool {
	for _, v := range s {
		if v == elem {
			return true
		}
	}
	return false
}

// Find returns the first element matching fn and true, or the zero value and false.
func Find[T any](s []T, fn func(T) bool) (T, bool) {
	for _, v := range s {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Flatten concatenates a slice of slices into a single slice.
func Flatten[T any](s [][]T) []T {
	total := 0
	for _, sub := range s {
		total += len(sub)
	}
	out := make([]T, 0, total)
	for _, sub := range s {
		out = append(out, sub...)
	}
	return out
}

// Reverse returns a new slice with elements in reverse order.
func Reverse[T any](s []T) []T {
	out := make([]T, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

// MapToMap converts a slice to a map using fn to produce key-value pairs.
func MapToMap[T any, K comparable, V any](s []T, fn func(T) (K, V)) map[K]V {
	m := make(map[K]V, len(s))
	for _, v := range s {
		k, val := fn(v)
		m[k] = val
	}
	return m
}

// ParallelMap applies fn to each element of s in parallel using up to concurrency goroutines.
// Results are returned in the same order as the input slice.
func ParallelMap[T, R any](s []T, fn func(T, int) R, concurrency int) []R {
	if concurrency <= 0 {
		concurrency = 1
	}
	out := make([]R, len(s))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, v := range s {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, val T) {
			defer func() {
				<-sem
				wg.Done()
			}()
			out[idx] = fn(val, idx)
		}(i, v)
	}
	wg.Wait()
	return out
}
