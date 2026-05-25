// Package slicex provides generic slice helper utilities.
// It re-exports the samber/lo slice functions and adds Santekno-specific extensions
// such as [MapToMap] and [ParallelMap].
//
//	doubled := slicex.Map([]int{1, 2, 3}, func(v, _ int) int { return v * 2 })
//	evens   := slicex.Filter([]int{1, 2, 3, 4}, func(v, _ int) bool { return v%2 == 0 })
package slicex
