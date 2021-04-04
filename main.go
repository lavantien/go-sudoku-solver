package main

import (
	"fmt"
)

func main() {
	const NUM = 50

	jobs := make(chan uint64, NUM)
	results := make(chan uint64, NUM)

	go worker(jobs, results)
	go worker(jobs, results)
	go worker(jobs, results)
	go worker(jobs, results)

	for i := uint64(0); i < NUM; i++ {
		jobs <- i
	}
	close(jobs)

	for j := 0; j < NUM; j++ {
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
