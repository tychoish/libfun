package irtx

import (
	"context"
	"iter"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestShardByHash(t *testing.T) {
	t.Run("ParallelProcessing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// Input workload
		workload := Slice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		numShards := 4

		// Hash function to distribute workload
		hashFunc := func(value int) int64 {
			return int64(value % numShards)
		}

		// Start time
		start := time.Now()

		// Shard the workload
		shards := ShardByHash(ctx, int64(numShards), workload, hashFunc)

		// Process shards in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make([]int, 0)

		for shard := range shards {
			wg.Add(1)
			go func(shard iter.Seq[int]) {
				defer wg.Done()

				for value := range shard {
					// Simulate processing time with a short sleep
					time.Sleep(10 * time.Millisecond)

					// Collect results
					mu.Lock()
					results = append(results, value)
					mu.Unlock()
				}
			}(shard)
		}

		wg.Wait()

		// End time
		duration := time.Since(start)

		// Verify results
		sort.Ints(results)
		expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		if !slices.Equal(results, expected) {
			t.Errorf("ShardByHash() results = %v, want %v", results, expected)
		}

		// Assert that the total time taken is less than the total sleep time
		// Total sleep time = 10ms * 10 items = 100ms
		// Parallel processing should take less than 100ms
		if duration >= 100*time.Millisecond {
			t.Errorf("ShardByHash() took %v, want less than 100ms", duration)
		}
	})
	t.Run("DeterministicRouting", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Input workload
		workload := Slice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		numShards := 2

		// Hash function: Route even numbers to shard 0, odd numbers to shard 1
		hashFunc := func(value int) int64 {
			if value%2 == 0 {
				return 0 // Even numbers
			}
			return 1 // Odd numbers
		}

		// Start time
		start := time.Now()

		// Shard the workload
		shards := ShardByHash(ctx, int64(numShards), workload, hashFunc)

		// Process shards in parallel
		var wg sync.WaitGroup
		results := make([][]int, numShards)
		var mu sync.Mutex

		for shardID, shard := range Collect2(Index(shards)) {
			wg.Add(1)
			go func(shardID int, shard iter.Seq[int]) {
				defer wg.Done()

				shardResults := []int{}
				for value := range shard {
					// Simulate processing time with a short sleep
					time.Sleep(10 * time.Millisecond)
					shardResults = append(shardResults, value)
				}

				// Store results for this shard
				mu.Lock()
				results[shardID] = shardResults
				mu.Unlock()
			}(shardID, shard)
		}

		wg.Wait()

		// End time
		duration := time.Since(start)

		// Verify results
		expected := [][]int{
			{2, 4, 6, 8, 10}, // Even numbers
			{1, 3, 5, 7, 9},  // Odd numbers
		}

		// Sort results for comparison
		for i := range results {
			sort.Ints(results[i])
		}

		if len(results) != len(expected) {
			t.Fatalf("ShardByHash() produced %d shards, want %d", len(results), len(expected))
		}

		for i := range results {
			if !slices.Equal(results[i], expected[i]) {
				t.Errorf("ShardByHash() shard %d = %v, want %v", i, results[i], expected[i])
			}
		}

		// Assert that the total time taken is less than the total sleep time
		// Total sleep time = 10ms * 10 items = 100ms
		// Parallel processing should take less than 100ms
		if duration >= 100*time.Millisecond {
			t.Errorf("ShardByHash() took %v, want less than 100ms", duration)
		}
	})
	t.Run("EarlyReturnFromOneShard", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// Input workload
		workload := Slice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		numShards := 3

		// Hash function: Route items to shards based on value % numShards
		hashFunc := func(value int) int64 {
			return int64(value % numShards)
		}

		// Start time
		start := time.Now()

		// Shard the workload
		shards := ShardByHash(ctx, int64(numShards), workload, hashFunc)

		// Process shards in parallel
		var wg sync.WaitGroup
		results := make([][]int, numShards)
		var mu sync.Mutex

		for shardID, shard := range Collect2(Index(shards)) {
			wg.Add(1)
			go func(shardID int, shard iter.Seq[int]) {
				defer wg.Done()

				shardResults := []int{}
				count := 0
				for value := range shard {
					// Simulate processing time with a short sleep
					time.Sleep(5 * time.Millisecond)

					// Simulate early return for shard 1
					if shardID == 1 && count > 1 {
						break
					}
					count++
					shardResults = append(shardResults, value)
				}

				// Store results for this shard
				mu.Lock()
				results[shardID] = shardResults
				mu.Unlock()
			}(shardID, shard)
		}

		wg.Wait()

		// End time
		duration := time.Since(start)

		// Verify results
		expected := [][]int{
			{3, 6, 9},     // Shard 0
			{1, 4},        // Shard 1 (early return, missing some items)
			{2, 5, 8, 10}, // Shard 2 (reassigned items from shard 1)
			// TODO where'd 7 go
		}

		// Sort results for comparison
		for i := range results {
			sort.Ints(results[i])
		}

		if len(results) != len(expected) {
			t.Fatalf("ShardByHash() produced %d shards, want %d", len(results), len(expected))
		}

		for i := range results {
			if !slices.Equal(results[i], expected[i]) {
				t.Errorf("ShardByHash() shard %d = %v, want %v", i, results[i], expected[i])
			}
		}

		// Assert that the total time taken is less than the total sleep time
		// Total sleep time = 10ms * 10 items = 100ms
		// Parallel processing should take less than 100ms
		if duration >= 100*time.Millisecond {
			t.Errorf("ShardByHash() took %v, want less than 100ms", duration)
		}
	})
	t.Run("Smoke", func(t *testing.T) {
		ctx := t.Context()

		workload := Slice([]string{"apple", "banana", "cherry", "date", "elderberry", "fig", "grape"})
		numShards := 3

		hashFunc := func(item string) int64 {
			return int64(len(item) % numShards)
		}

		shards := ShardByHash(ctx, int64(numShards), workload, hashFunc)

		// Ensure each shard contains a portion of the workload based on the hash
		shardContents := make([][]string, numShards)
		for shardID, shard := range Collect2(Index(shards)) {
			shardContents[shardID] = Collect(shard)
		}

		for shardID, items := range shardContents {
			t.Logf("Shard %d: %v", shardID, items)
			for _, item := range items {
				if hashFunc(item) != int64(shardID) {
					t.Errorf("Item %s routed to wrong shard %d", item, shardID)
				}
			}
		}

		t.Run("Compensate for early return", func(t *testing.T) {
			// Ensure early return does not block other shards
			shards = ShardByHash(ctx, int64(numShards), workload, hashFunc)
			seen := 0
			for shard := range shards {
				seen++
				if seen == 2 {
					break
				}
				inner := 0
				for in := range shard {
					inner++
					if seen == 3 && inner == 1 {
						break
					}
					if in == "" {
						t.Error("saw empty string")
					}
				}

				break // Simulate early return
			}
		})
	})
}
