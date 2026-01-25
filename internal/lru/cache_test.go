package lru

import (
	"fmt"
	"testing"
	"time"
)

func TestCache_Put(t *testing.T) {
	cache := NewCache(3)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)
	cache.Put("key3", "value3", time.Minute)

	if cache.Len() != 3 {
		t.Errorf("Len() = %d, want 3", cache.Len())
	}
}

func TestCache_Get(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	val, ok := cache.Get("key1")

	if !ok {
		t.Error("Get() should return true for existing key")
	}
	if val != "value1" {
		t.Errorf("Get() = %v, want value1", val)
	}
}

func TestCache_Get_NotFound(t *testing.T) {
	cache := NewCache(10)

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for nonexistent key")
	}
}

func TestCache_Remove(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	if err := cache.Remove("key1"); err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	if cache.Len() != 0 {
		t.Errorf("Len() after remove = %d, want 0", cache.Len())
	}
}

func TestCache_RemoveExpired(t *testing.T) {
	cache := NewCache(10)

	now := time.Now()
	cache.Put("key1", "value1", -time.Minute) // already expired
	cache.Put("key2", "value2", time.Minute)  // not expired

	removed := cache.RemoveExpired(now)
	if removed != 1 {
		t.Errorf("RemoveExpired() = %d, want 1", removed)
	}

	if cache.Len() != 1 {
		t.Errorf("Len() after removeExpired = %d, want 1", cache.Len())
	}
}

func TestCache_Eviction(t *testing.T) {
	cache := NewCache(3)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)
	cache.Put("key3", "value3", time.Minute)
	cache.Put("key4", "value4", time.Minute) // should evict key1

	if cache.Len() != 3 {
		t.Errorf("Len() after eviction = %d, want 3", cache.Len())
	}

	_, ok := cache.Get("key1")
	if ok {
		t.Error("key1 should have been evicted")
	}
}

func TestCache_Touch(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	if err := cache.Touch("key1", time.Minute); err != nil {
		t.Errorf("Touch() error = %v", err)
	}
}

func TestCache_Touch_NotFound(t *testing.T) {
	cache := NewCache(10)

	if err := cache.Touch("nonexistent", time.Minute); err != ErrKeyNotFound {
		t.Errorf("Touch() for nonexistent key should return ErrKeyNotFound")
	}
}

func TestCache_PeekBack(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)

	// key1 should be at the back (oldest)
	key, val, ok := cache.PeekBack()
	if !ok {
		t.Error("PeekBack() should return true when cache is not empty")
	}
	if key != "key1" {
		t.Errorf("PeekBack() key = %v, want key1", key)
	}
	if val != "value1" {
		t.Errorf("PeekBack() val = %v, want value1", val)
	}
}

func TestCache_Empty(t *testing.T) {
	cache := NewCache(10)

	if !cache.Empty() {
		t.Error("Empty() should return true for new cache")
	}

	cache.Put("key1", "value1", time.Minute)
	if cache.Empty() {
		t.Error("Empty() should return false after adding items")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Len() after clear = %d, want 0", cache.Len())
	}
}

func TestCache_MaxSize(t *testing.T) {
	cache := NewCache(2)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)
	cache.Put("key3", "value3", time.Minute) // should evict key1

	if cache.Len() != 2 {
		t.Errorf("Len() = %d, want 2", cache.Len())
	}
}

func TestCache_LRUOrder(t *testing.T) {
	cache := NewCache(3)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)
	cache.Put("key3", "value3", time.Minute)

	// Access key1 to make it most recently used
	cache.Get("key1")

	// Add new key to trigger eviction of key2
	cache.Put("key4", "value4", time.Minute)

	// key2 should be evicted
	_, ok := cache.Get("key2")
	if ok {
		t.Error("key2 should have been evicted")
	}

	// key1 and key3 and key4 should still exist
	if cache.Len() != 3 {
		t.Errorf("Len() = %d, want 3", cache.Len())
	}
}

func TestCache_Update(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key1", "value2", time.Minute) // Update

	val, ok := cache.Get("key1")
	if !ok {
		t.Error("Get() should return true for existing key")
	}
	if val != "value2" {
		t.Errorf("Get() = %v, want value2", val)
	}
}

func TestCache_RemoveExpiredRatio(t *testing.T) {
	cache := NewCache(100)

	// 先添加10个过期条目（在尾部，因为先添加）
	for i := 0; i < 10; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i, -time.Second)
	}

	// 再添加有效条目（在头部）
	for i := 10; i < 60; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i, time.Hour)
	}

	time.Sleep(100 * time.Millisecond)

	removed := cache.RemoveExpiredRatio(time.Now(), 30, 5)
	if removed < 5 {
		t.Errorf("Should have removed at least 5 expired entries, got %d", removed)
	}
}

func TestCache_RemoveNotExist(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)

	// Try to remove non-existent key - should return error
	err := cache.Remove("key2")
	if err == nil {
		t.Error("Remove() should return error for non-existent key")
	}
}

func TestCache_GetTS(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	val, ts, ok := cache.GetTS("key1")

	if !ok {
		t.Error("GetTS() should return true for existing key")
	}
	if val != "value1" {
		t.Errorf("GetTS() = %v, want value1", val)
	}
	if ts.IsZero() {
		t.Error("GetTS() timestamp should not be zero")
	}
}

func TestCache_GetTS_NotFound(t *testing.T) {
	cache := NewCache(10)

	_, _, ok := cache.GetTS("nonexistent")
	if ok {
		t.Error("GetTS() should return false for nonexistent key")
	}
}

func TestCache_PeekBackTS(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)

	// key1 should be at the back (oldest)
	key, ts, ok := cache.PeekBackTS()
	if !ok {
		t.Error("PeekBackTS() should return true when cache is not empty")
	}
	if key != "key1" {
		t.Errorf("PeekBackTS() key = %v, want key1", key)
	}
	if ts.IsZero() {
		t.Error("PeekBackTS() timestamp should not be zero")
	}
}

func TestCache_PeekBackTS_Empty(t *testing.T) {
	cache := NewCache(10)

	_, _, ok := cache.PeekBackTS()
	if ok {
		t.Error("PeekBackTS() should return false for empty cache")
	}
}

func TestCache_SetCleanupFn(t *testing.T) {
	cache := NewCache(10)

	cleanupCalled := false
	cache.SetCleanupFn(func(key, value interface{}) {
		cleanupCalled = true
	})

	cache.Put("key1", "value1", time.Minute)
	cache.Remove("key1")

	if !cleanupCalled {
		t.Error("Cleanup function should have been called")
	}
}

func TestCache_RemoveExpiredRatio_Empty(t *testing.T) {
	cache := NewCache(10)

	removed := cache.RemoveExpiredRatio(time.Now(), 30, 5)
	if removed != 0 {
		t.Errorf("RemoveExpiredRatio() on empty cache = %d, want 0", removed)
	}
}

func TestCache_RemoveExpiredRatio_MoreThanLen(t *testing.T) {
	cache := NewCache(10)

	cache.Put("key1", "value1", -time.Second) // expired
	cache.Put("key2", "value2", -time.Second) // expired

	removed := cache.RemoveExpiredRatio(time.Now(), 30, 100) // request more than exist
	if removed != 2 {
		t.Errorf("RemoveExpiredRatio() = %d, want 2", removed)
	}
}

func TestCache_RemoveExpired_CallsCleanup(t *testing.T) {
	cache := NewCache(10)

	cleanupKeys := make([]interface{}, 0)
	cache.SetCleanupFn(func(key, value interface{}) {
		cleanupKeys = append(cleanupKeys, key)
	})

	now := time.Now()
	cache.Put("key1", "value1", -time.Minute) // already expired
	cache.Put("key2", "value2", time.Minute)  // not expired

	cache.RemoveExpired(now)

	if len(cleanupKeys) != 1 || cleanupKeys[0] != "key1" {
		t.Errorf("Cleanup should have been called for key1, got %v", cleanupKeys)
	}
}

func TestCache_Clear_CallsCleanup(t *testing.T) {
	cache := NewCache(10)

	cleanupKeys := make([]interface{}, 0)
	cache.SetCleanupFn(func(key, value interface{}) {
		cleanupKeys = append(cleanupKeys, key)
	})

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)

	cache.Clear()

	if len(cleanupKeys) != 2 {
		t.Errorf("Cleanup should have been called twice, got %d", len(cleanupKeys))
	}
}

func TestCache_Eviction_CallsCleanup(t *testing.T) {
	cache := NewCache(2)

	evictedKeys := make([]interface{}, 0)
	cache.SetCleanupFn(func(key, value interface{}) {
		evictedKeys = append(evictedKeys, key)
	})

	cache.Put("key1", "value1", time.Minute)
	cache.Put("key2", "value2", time.Minute)
	cache.Put("key3", "value3", time.Minute) // should evict key1

	if len(evictedKeys) != 1 || evictedKeys[0] != "key1" {
		t.Errorf("Cleanup should have been called for evicted key1, got %v", evictedKeys)
	}
}

func TestCache_MaxSizeZero_Unlimited(t *testing.T) {
	cache := NewCache(0) // unlimited

	// Add more than would normally trigger eviction
	for i := 0; i < 100; i++ {
		cache.Put(fmt.Sprintf("key%d", i), i, time.Minute)
	}

	if cache.Len() != 100 {
		t.Errorf("Len() = %d, want 100 for unlimited cache", cache.Len())
	}
}

func TestCache_PeekBack_Empty(t *testing.T) {
	cache := NewCache(10)

	_, _, ok := cache.PeekBack()
	if ok {
		t.Error("PeekBack() should return false for empty cache")
	}
}

func TestCache_RemoveExpiredRatio_ZeroRatio(t *testing.T) {
	cache := NewCache(100)

	cache.Put("key1", "value1", -time.Second) // expired

	// ratio=0 should return 0 (safety check to avoid divide by zero)
	removed := cache.RemoveExpiredRatio(time.Now(), 0, 1)
	if removed != 0 {
		t.Errorf("RemoveExpiredRatio() with ratio=0 = %d, want 0 (safety check)", removed)
	}
}
