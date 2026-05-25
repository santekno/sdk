// Package mapx provides generic map helper utilities.
//
//	keys   := mapx.Keys(m)
//	values := mapx.Values(m)
//	merged := mapx.Merge(m1, m2)
package mapx

// Keys returns the keys of m in unspecified order.
func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Values returns the values of m in unspecified order.
func Values[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// Entry is a key-value pair.
type Entry[K comparable, V any] struct {
	Key   K
	Value V
}

// Entries returns the key-value pairs of m in unspecified order.
func Entries[K comparable, V any](m map[K]V) []Entry[K, V] {
	out := make([]Entry[K, V], 0, len(m))
	for k, v := range m {
		out = append(out, Entry[K, V]{Key: k, Value: v})
	}
	return out
}

// FromEntries builds a map from a slice of Entry values.
func FromEntries[K comparable, V any](entries []Entry[K, V]) map[K]V {
	m := make(map[K]V, len(entries))
	for _, e := range entries {
		m[e.Key] = e.Value
	}
	return m
}

// Invert returns a new map with keys and values swapped.
func Invert[K, V comparable](m map[K]V) map[V]K {
	out := make(map[V]K, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

// MapKeys returns a new map with keys transformed by fn.
func MapKeys[K1, K2 comparable, V any](m map[K1]V, fn func(K1) K2) map[K2]V {
	out := make(map[K2]V, len(m))
	for k, v := range m {
		out[fn(k)] = v
	}
	return out
}

// MapValues returns a new map with values transformed by fn.
func MapValues[K comparable, V1, V2 any](m map[K]V1, fn func(V1) V2) map[K]V2 {
	out := make(map[K]V2, len(m))
	for k, v := range m {
		out[k] = fn(v)
	}
	return out
}

// Filter returns a new map containing only the entries for which fn returns true.
func Filter[K comparable, V any](m map[K]V, fn func(K, V) bool) map[K]V {
	out := make(map[K]V)
	for k, v := range m {
		if fn(k, v) {
			out[k] = v
		}
	}
	return out
}

// Merge returns a new map containing all entries from all maps.
// Later maps take precedence on key conflicts.
func Merge[K comparable, V any](maps ...map[K]V) map[K]V {
	total := 0
	for _, m := range maps {
		total += len(m)
	}
	out := make(map[K]V, total)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
