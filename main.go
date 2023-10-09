//Code Created By Data Gurtchumelia And little Chat-Gpt :))
//https://github.com/datagurtchumelia
//LICENSE GNU General Public License v3.0
package main

import (
	"context"
	"fmt"
	"sync"
)

const (
	maxRetries        = 1 
	numWorkers        = 8192 //This is for 32 core server 128gb ram
	startingNumbersRange = 1000000 
	chunkSize         = startingNumbersRange / numWorkers
)

type safeMap struct {
	sync.RWMutex
	data map[int][]int
}

func newSafeMap() *safeMap {
	return &safeMap{
		data: make(map[int][]int),
	}
}

func (m *safeMap) store(key int, value []int) {
	m.Lock()
	defer m.Unlock()
	m.data[key] = value
}

func (m *safeMap) load(key int) ([]int, bool) {
	m.RLock()
	defer m.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func collatzSequence(n int) []int {
	sequence := []int{n}
	for n != 1 {
		if n%2 == 0 {
			n = n / 2
		} else {
			n = 3*n + 1
		}
		sequence = append(sequence, n)
	}
	return sequence
}

func calculateSequence(ctx context.Context, startingNumbers []int, results chan<- map[int][]int, failedChunks chan<- []int) {
	sequenceMap := newSafeMap()
	for _, num := range startingNumbers {
		select {
		case <-ctx.Done():
			return
		default:
			sequenceMap.store(num, collatzSequence(num))
		}
	}
	results <- sequenceMap.data
}

func main() {
	startingNumbers := make([]int, startingNumbersRange)
	for i := 1; i <= startingNumbersRange; i++ {
		startingNumbers[i-1] = i
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startingNumberChunks := make([][]int, numWorkers)
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == numWorkers-1 {
			end = len(startingNumbers)
		}
		startingNumberChunks[i] = startingNumbers[start:end]
	}

	var wg sync.WaitGroup
	results := make(chan map[int][]int, numWorkers)
	failedChunks := make(chan []int, numWorkers)

	for i, chunk := range startingNumberChunks {
		wg.Add(1)
		go func(i int, chunk []int) {
			defer wg.Done()
			fmt.Printf("Processing chunk %d...\n", i)
			retries := 0
			for retries < maxRetries {
				select {
				case <-ctx.Done():
					return
				default:
					calculateSequence(ctx, chunk, results, failedChunks)
					retries++
					fmt.Printf("Retrying chunk %d, attempt %d...\n", i, retries)
				}
			}
			if retries == maxRetries {
				fmt.Printf("Failed %d i %d attempts. Skipping.\n", i, maxRetries)
			}
			fmt.Printf("Finished processing chunk %d.\n", i)
		}(i, chunk)
	}

	go func() {
		wg.Wait()
		close(results)
		close(failedChunks)
	}()

	for result := range results {
		for num, sequence := range result {
			fmt.Printf("starting number %d: %v\n", num, sequence)
		}
	}

	for failedChunk := range failedChunks {
		fmt.Printf("Failed chunk: %v\n", failedChunk)
	}
}
