package main

import (
	"fmt"
)

func main() {
	jobs := make(chan uint64, 100)
	results := make(chan uint64, 100)

	go worker(jobs, results)

	for i := uint64(0); i < 100; i++ {
		jobs <- i
	}
	close(jobs)

	for j := 0; j < 100; j++ {
		fmt.Println(<-results)
	}
}

func worker(jobs <-chan uint64, results chan<- uint64) {
	for n := range jobs {
		results <- Fib(n)
	}
}

func Fib(n uint64) uint64 {
	cache := make(map[uint64]uint64)
	return fib(n, cache)
}

func fib(n uint64, cache map[uint64]uint64) uint64 {
	if n <= 2 {
		return 1
	}
	if _, ok := cache[n]; ok {
		return cache[n]
	}
	cache[n] = fib(n-1, cache) + fib(n-2, cache)
	return cache[n]
}
