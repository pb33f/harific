package motor

import (
	"testing"

	"github.com/pb33f/harhar"
)

func TestNoOpCache(t *testing.T) {
	cache := NewNoOpCache()

	// test get on empty cache
	entry, ok := cache.Get(0)
	if ok {
		t.Error("expected cache miss")
	}
	if entry != nil {
		t.Error("expected nil entry")
	}

	// test size before put
	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}

	// test put (should do nothing)
	testEntry := &harhar.Entry{
		Start: "2024-01-01T00:00:00Z",
		Time:  100.0,
	}
	cache.Put(0, testEntry)

	// verify it's still a miss
	entry, ok = cache.Get(0)
	if ok {
		t.Error("expected cache miss after put")
	}
	if entry != nil {
		t.Error("expected nil entry after put")
	}

	// test size after put (should still be 0)
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after put, got %d", cache.Size())
	}

	// test clear (should do nothing)
	cache.Clear()

	// verify size is still 0
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}
}

func TestNoOpCacheMultipleEntries(t *testing.T) {
	cache := NewNoOpCache()

	// put multiple entries
	for i := 0; i < 100; i++ {
		cache.Put(i, &harhar.Entry{
			Start: "2024-01-01T00:00:00Z",
			Time:  float64(i),
		})
	}

	// verify none are cached
	for i := 0; i < 100; i++ {
		if _, ok := cache.Get(i); ok {
			t.Errorf("expected miss for entry %d", i)
		}
	}

	// size should still be 0
	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}
}
