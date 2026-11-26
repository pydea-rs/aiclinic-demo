package main

import (
	"math/rand"
	"sync"
	"time"
)

// rng is a concurrency-safe random source shared across the seeding phases,
// which run in parallel goroutines. math/rand's top-level funcs are safe but a
// dedicated, seedable source keeps runs reproducible when RANDOM_SEED is set.
type safeRand struct {
	mu sync.Mutex
	r  *rand.Rand
}

var rng *safeRand

// initRNG seeds the global RNG. seed == 0 => time-based (non-reproducible).
func initRNG(seed int64) {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng = &safeRand{r: rand.New(rand.NewSource(seed))}
}

func (s *safeRand) intn(n int) int {
	if n <= 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.r.Intn(n)
}

// between returns a random int in [min, max] inclusive.
func (s *safeRand) between(min, max int) int {
	if max <= min {
		return min
	}
	return min + s.intn(max-min+1)
}

// pick returns a random element from a slice.
func pick[T any](s *safeRand, items []T) T {
	return items[s.intn(len(items))]
}

// subset returns a random non-empty subset of items with a size in [min, max]
// (clamped to len(items)). Order is randomized.
func subset[T any](s *safeRand, items []T, min, max int) []T {
	n := len(items)
	if n == 0 {
		return nil
	}
	if max > n {
		max = n
	}
	if min < 0 {
		min = 0
	}
	if min > max {
		min = max
	}
	k := s.between(min, max)
	if k == 0 {
		return nil
	}

	// Partial Fisher-Yates on a copy of indices.
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	s.mu.Lock()
	for i := 0; i < k; i++ {
		j := i + s.r.Intn(n-i)
		idx[i], idx[j] = idx[j], idx[i]
	}
	s.mu.Unlock()

	out := make([]T, k)
	for i := 0; i < k; i++ {
		out[i] = items[idx[i]]
	}
	return out
}
