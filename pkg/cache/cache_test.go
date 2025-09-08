package cache_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yeisme/notevault/pkg/cache"
)

// TestUser 测试用的用户结构体.
type TestUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// mockKVStore 模拟KV存储实现，用于基准测试.
type mockKVStore struct {
	data map[string][]byte
}

func newMockKVStore() *mockKVStore {
	return &mockKVStore{
		data: make(map[string][]byte),
	}
}

func (m *mockKVStore) Get(ctx context.Context, key string) ([]byte, error) {
	if value, exists := m.data[key]; exists {
		return value, nil
	}

	return nil, fmt.Errorf("key not found")
}

func (m *mockKVStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockKVStore) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockKVStore) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

func (m *mockKVStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}

	return keys, nil
}

func (m *mockKVStore) Close() error {
	return nil
}

// TestNewCache 测试 NewCache 函数.
func TestNewCache(t *testing.T) {
	mockStore := newMockKVStore()
	cache := cache.NewCache(mockStore)

	if cache == nil {
		t.Fatal("NewCache returned nil")
	}
}

// TestCache_Get 测试 Get 方法.
func TestCache_Get(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	// 测试获取不存在的键
	_, err := cache.Get[TestUser](ctx, c, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent key")
	}

	// 设置测试数据
	user := TestUser{ID: 1, Name: "Alice", Age: 30}

	err = cache.Set(ctx, c, "user:1", user, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 获取存在的键
	retrievedUser, err := cache.Get[TestUser](ctx, c, "user:1")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if retrievedUser.ID != user.ID || retrievedUser.Name != user.Name || retrievedUser.Age != user.Age {
		t.Errorf("Retrieved user %+v does not match original %+v", retrievedUser, user)
	}
}

// TestCache_Set 测试 Set 方法.
func TestCache_Set(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	user := TestUser{ID: 2, Name: "Bob", Age: 25}

	err := cache.Set(ctx, c, "user:2", user, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 验证数据是否正确存储
	data, exists := mockStore.data["user:2"]
	if !exists {
		t.Fatal("Data not stored in mock store")
	}

	if len(data) == 0 {
		t.Error("Stored data is empty")
	}
}

// TestCache_Delete 测试 Delete 方法.
func TestCache_Delete(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	// 设置数据
	user := TestUser{ID: 3, Name: "Charlie", Age: 35}

	err := cache.Set(ctx, c, "user:3", user, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 验证存在
	exists, err := c.Exists(ctx, "user:3")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}

	if !exists {
		t.Error("Key should exist before deletion")
	}

	// 删除数据
	err = c.Delete(ctx, "user:3")
	if err != nil {
		t.Fatalf("Failed to delete cache: %v", err)
	}

	// 验证不存在
	exists, err = c.Exists(ctx, "user:3")
	if err != nil {
		t.Fatalf("Failed to check existence after deletion: %v", err)
	}

	if exists {
		t.Error("Key should not exist after deletion")
	}
}

// TestCache_Exists 测试 Exists 方法.
func TestCache_Exists(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	// 测试不存在的键
	exists, err := c.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}

	if exists {
		t.Error("Nonexistent key should not exist")
	}

	// 设置数据
	user := TestUser{ID: 4, Name: "David", Age: 40}

	err = cache.Set(ctx, c, "user:4", user, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// 测试存在的键
	exists, err = c.Exists(ctx, "user:4")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}

	if !exists {
		t.Error("Existing key should exist")
	}
}

// TestGetOrSet 测试 GetOrSet 方法.
func TestGetOrSet(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	callCount := 0
	getter := func() (TestUser, error) {
		callCount++
		return TestUser{ID: 5, Name: "Eve", Age: 28}, nil
	}

	// 第一次调用，应该调用getter
	user1, err := cache.GetOrSet(ctx, c, "user:5", getter, 0)
	if err != nil {
		t.Fatalf("Failed to get or set: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected getter to be called once, got %d", callCount)
	}

	// 第二次调用，应该从缓存获取
	user2, err := cache.GetOrSet(ctx, c, "user:5", getter, 0)
	if err != nil {
		t.Fatalf("Failed to get or set: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected getter to be called only once, got %d", callCount)
	}

	// 验证两次结果相同
	if user1.ID != user2.ID || user1.Name != user2.Name || user1.Age != user2.Age {
		t.Errorf("Results don't match: %+v vs %+v", user1, user2)
	}
}

// TestGetOrSet_GetterError 测试 GetOrSet 方法中 getter 返回错误的情况.
func TestGetOrSet_GetterError(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	getter := func() (TestUser, error) {
		return TestUser{}, errors.New("getter error")
	}

	_, err := cache.GetOrSet(ctx, c, "user:error", getter, 0)
	if err == nil {
		t.Error("Expected error from getter")
	}

	if err.Error() != "getter error" {
		t.Errorf("Expected 'getter error', got '%s'", err.Error())
	}
}

// TestCache_Clear 测试 Clear 方法.
func TestCache_Clear(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	// 设置多个数据
	users := []TestUser{
		{ID: 6, Name: "Frank", Age: 45},
		{ID: 7, Name: "Grace", Age: 32},
		{ID: 8, Name: "Henry", Age: 50},
	}

	for i, user := range users {
		key := fmt.Sprintf("user:%d", user.ID)

		err := cache.Set(ctx, c, key, user, 0)
		if err != nil {
			t.Fatalf("Failed to set cache for user %d: %v", i, err)
		}
	}

	// 验证数据存在
	if len(mockStore.data) != len(users) {
		t.Errorf("Expected %d items, got %d", len(users), len(mockStore.data))
	}

	// 清空缓存
	err := c.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// 验证数据被清空
	if len(mockStore.data) != 0 {
		t.Errorf("Expected 0 items after clear, got %d", len(mockStore.data))
	}
}

// TestCache_GenericTypes 测试缓存对不同数据类型的支持.
func TestCache_GenericTypes(t *testing.T) {
	mockStore := newMockKVStore()
	c := cache.NewCache(mockStore)
	ctx := context.Background()

	// 测试字符串类型
	err := cache.Set(ctx, c, "string:key", "hello world", 0)
	if err != nil {
		t.Fatalf("Failed to set string: %v", err)
	}

	str, err := cache.Get[string](ctx, c, "string:key")
	if err != nil {
		t.Fatalf("Failed to get string: %v", err)
	}

	if str != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", str)
	}

	// 测试整数类型
	err = cache.Set(ctx, c, "int:key", 42, 0)
	if err != nil {
		t.Fatalf("Failed to set int: %v", err)
	}

	num, err := cache.Get[int](ctx, c, "int:key")
	if err != nil {
		t.Fatalf("Failed to get int: %v", err)
	}

	if num != 42 {
		t.Errorf("Expected 42, got %d", num)
	}

	// 测试切片类型
	slice := []string{"a", "b", "c"}

	err = cache.Set(ctx, c, "slice:key", slice, 0)
	if err != nil {
		t.Fatalf("Failed to set slice: %v", err)
	}

	retrievedSlice, err := cache.Get[[]string](ctx, c, "slice:key")
	if err != nil {
		t.Fatalf("Failed to get slice: %v", err)
	}

	if len(retrievedSlice) != len(slice) {
		t.Errorf("Slice length mismatch: expected %d, got %d", len(slice), len(retrievedSlice))
	}

	for i, v := range slice {
		if retrievedSlice[i] != v {
			t.Errorf("Slice element %d mismatch: expected %s, got %s", i, v, retrievedSlice[i])
		}
	}
}
