package internal_test

import (
	"sync"
	"testing"

	"github.com/lewisgibson/go-vdf/internal"
	"github.com/stretchr/testify/require"
)

func TestPool_EmptyPool(t *testing.T) {
	t.Parallel()

	// Arrange: Create a pool for testing
	pool := internal.NewPool(func() *int {
		val := 42
		return &val
	})

	// Act: Get multiple objects without putting any back
	val1 := pool.Get()
	val2 := pool.Get()

	// Assert: Pool should create new objects when empty
	require.NotNil(t, val1, "First Get should not return nil")
	require.Equal(t, 42, *val1, "First value should be correct")

	require.NotNil(t, val2, "Second Get should not return nil")
	require.Equal(t, 42, *val2, "Second value should be correct")
}

func TestPool_GetPut(t *testing.T) {
	t.Parallel()

	// Arrange: Create a pool for testing
	pool := internal.NewPool(func() *int {
		val := 42
		return &val
	})

	// Act: Test basic Get/Put cycle
	val1 := pool.Get()
	require.NotNil(t, val1, "Get() should not return nil")
	require.Equal(t, 42, *val1, "Initial value should be 42")

	// Act: Modify the value and put it back
	*val1 = 100
	pool.Put(val1)

	// Act: Get another value - might reuse the same instance or create a new one
	val2 := pool.Get()
	require.NotNil(t, val2, "Get() should not return nil after Put")

	// Assert: Verify the behavior based on whether the object was reused
	if val1 == val2 {
		// If it's the same pointer, it should have the modified value
		require.Equal(t, 100, *val2, "Reused value should retain modified value")
	} else {
		// If it's a new instance, it should have the original value
		require.Equal(t, 42, *val2, "New instance should have original value")
	}
}

func TestPool_ReuseBehavior(t *testing.T) {
	t.Parallel()

	// Arrange: Create a pool for testing
	pool := internal.NewPool(func() *int {
		val := 0
		return &val
	})

	// Act: Get an object, modify it, and put it back
	val1 := pool.Get()
	require.NotNil(t, val1, "Get() should not return nil")
	*val1 = 42
	pool.Put(val1)

	// Act: Get another object
	val2 := pool.Get()
	require.NotNil(t, val2, "Get() should not return nil")

	// Assert: The pool may reuse objects when available
	// Note: sync.Pool doesn't reset reused objects, so we just verify
	// that we get a valid object back
	require.NotNil(t, val2, "Should get a valid object from pool")

	// If it's the same object, it retains the modified value
	// If it's a new object, it has the constructor's initial value
	if val1 == val2 {
		require.Equal(t, 42, *val2, "Reused object retains its value")
	} else {
		require.Equal(t, 0, *val2, "New object has constructor's initial value")
	}
}

func TestPool_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	// Arrange: Create a pool for testing
	pool := internal.NewPool(func() *int {
		val := 0
		return &val
	})

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	results := make(chan *int, numGoroutines*numOperations)

	// Act: Spawn multiple goroutines that Get/Put from the pool
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				val := pool.Get()
				*val = j // Set some value
				results <- val
				pool.Put(val)
			}
		}()
	}

	wg.Wait()
	close(results)

	// Assert: Verify we got the expected number of results
	count := 0
	for range results {
		count++
	}

	expected := numGoroutines * numOperations
	require.Equal(t, expected, count, "Should process all operations")
}

func TestPool_MultiplePuts(t *testing.T) {
	t.Parallel()

	// Arrange: Create a pool for testing
	pool := internal.NewPool(func() *int {
		val := 42
		return &val
	})

	// Act: Get an object and put it back multiple times
	val := pool.Get()
	require.NotNil(t, val, "Get should not return nil")
	*val = 100
	pool.Put(val)
	pool.Put(val) // Put same object again

	// Assert: Multiple puts should not cause issues
	// This test verifies that the pool handles multiple puts gracefully
	require.NotNil(t, val, "Object should still be valid after multiple puts")
}
