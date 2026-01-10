	package flags

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use different DB for tests
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clear test DB
	err = client.FlushDB(ctx).Err()
	require.NoError(t, err)

	return client
}

// cleanupTestRedis intentionally accepts *testing.T for future debugging use, even if 't' is currently unused.
func cleanupTestRedis(_ *testing.T, client *redis.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = client.FlushDB(ctx).Err()
	_ = client.Close()
}

func TestStore_Upsert(t *testing.T) {
	client := setupTestRedis(t)
	defer cleanupTestRedis(t, client)

	store, err := NewStore(client)
	require.NoError(t, err)

	ctx := context.Background()

	// Test setting a new flag
	flag, err := store.Upsert(ctx, "test.flag", true)
	assert.NoError(t, err)
	assert.NotNil(t, flag)
	assert.Equal(t, "test.flag", flag.Key)
	assert.True(t, flag.Value)
	assert.NotZero(t, flag.UpdatedAt)

	// Verify flag was set
	retrievedFlag, err := store.Get(ctx, "test.flag")
	assert.NoError(t, err)
	assert.Equal(t, flag.Key, retrievedFlag.Key)
	assert.Equal(t, flag.Value, retrievedFlag.Value)
	assert.Equal(t, flag.UpdatedAt, retrievedFlag.UpdatedAt)

	// Test updating existing flag
	time.Sleep(time.Millisecond) // Ensure different timestamp
	flag2, err := store.Upsert(ctx, "test.flag", false)
	assert.NoError(t, err)
	assert.True(t, flag2.UpdatedAt.After(flag.UpdatedAt))

	// Verify flag was updated
	retrievedFlag, err = store.Get(ctx, "test.flag")
	assert.NoError(t, err)
	assert.False(t, retrievedFlag.Value)
	assert.Equal(t, flag2.UpdatedAt, retrievedFlag.UpdatedAt)
}

func TestStore_Get(t *testing.T) {
	client := setupTestRedis(t)
	defer cleanupTestRedis(t, client)

	store, err := NewStore(client)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting non-existent flag
	flag, err := store.Get(ctx, "nonexistent.flag")
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, flag)

	// Set a flag first
	_, err = store.Upsert(ctx, "test.flag", true)
	require.NoError(t, err)

	// Test getting existing flag
	flag, err = store.Get(ctx, "test.flag")
	assert.NoError(t, err)
	assert.NotNil(t, flag)
	assert.Equal(t, "test.flag", flag.Key)
	assert.True(t, flag.Value)
	assert.NotZero(t, flag.UpdatedAt)
}

func TestStore_Delete(t *testing.T) {
	client := setupTestRedis(t)
	defer cleanupTestRedis(t, client)

	store, err := NewStore(client)
	require.NoError(t, err)

	ctx := context.Background()

	// Set a flag first
	_, err = store.Upsert(ctx, "test.flag", true)
	require.NoError(t, err)

	// Verify flag exists
	_, err = store.Get(ctx, "test.flag")
	assert.NoError(t, err)

	// Delete the flag
	err = store.Delete(ctx, "test.flag")
	assert.NoError(t, err)

	// Verify flag is deleted
	_, err = store.Get(ctx, "test.flag")
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)

	// Test deleting non-existent flag
	err = store.Delete(ctx, "nonexistent.flag")
	assert.NoError(t, err) // Should not error
}

func TestStore_List(t *testing.T) {
	client := setupTestRedis(t)
	defer cleanupTestRedis(t, client)

	store, err := NewStore(client)
	require.NoError(t, err)

	ctx := context.Background()

	// Test empty list
	flags, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, flags)

	// Add some flags
	flagUpdates := map[string]bool{
		"flag1": true,
		"flag2": false,
		"flag3": true,
	}

	for key, value := range flagUpdates {
		_, err := store.Upsert(ctx, key, value)
		require.NoError(t, err)
	}

	// List flags
	flags, err = store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, flags, 3)

	// Create a map for easier verification
	flagMap := make(map[string]bool)
	for _, flag := range flags {
		flagMap[flag.Key] = flag.Value
	}

	for key, expectedValue := range flagUpdates {
		actualValue, exists := flagMap[key]
		assert.True(t, exists, "Flag %s should exist", key)
		assert.Equal(t, expectedValue, actualValue, "Flag %s should have correct value", key)
	}
}

func TestStore_ConcurrentOperations(t *testing.T) {
	client := setupTestRedis(t)
	defer cleanupTestRedis(t, client)

	store, err := NewStore(client)
	require.NoError(t, err)

	ctx := context.Background()

	// Test concurrent sets
	const numGoroutines = 10
	const numOps = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("flag.%d.%d", id, j)
				value := (id+j)%2 == 0

				_, err := store.Upsert(ctx, key, value)
				assert.NoError(t, err)

				retrievedFlag, err := store.Get(ctx, key)
				assert.NoError(t, err)
				assert.Equal(t, value, retrievedFlag.Value)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all flags exist
	flags, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, flags, numGoroutines*numOps)
}

func TestStore_InvalidKeys(t *testing.T) {
	client := setupTestRedis(t)
	defer cleanupTestRedis(t, client)

	store, err := NewStore(client)
	require.NoError(t, err)

	ctx := context.Background()

	// Test empty key
	_, err = store.Upsert(ctx, "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid flag key")

	// Test key with spaces
	_, err = store.Upsert(ctx, "invalid key", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid flag key")

	// Test key with colon
	_, err = store.Upsert(ctx, "invalid:key", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid flag key")
}

func TestStore_KeyValidation(t *testing.T) {
	client := setupTestRedis(t)
	defer cleanupTestRedis(t, client)

	store, err := NewStore(client)
	require.NoError(t, err)

	ctx := context.Background()

	// Valid keys
	validKeys := []string{
		"simple.flag",
		"flag.with.dots",
		"flag123",
		"a",
		"very.long.flag.name.with.many.parts",
	}

	for _, key := range validKeys {
		_, err := store.Upsert(ctx, key, true)
		assert.NoError(t, err, "Key %s should be valid", key)
	}

	// Invalid keys
	invalidKeys := []string{
		"",
		" ",
		"flag with spaces",
		"flag:with:colons",
		"flag\twith\ttabs",
		"flag\nwith\nnewlines",
	}

	for _, key := range invalidKeys {
		_, err := store.Upsert(ctx, key, true)
		assert.Error(t, err, "Key %s should be invalid", key)
	}
}
