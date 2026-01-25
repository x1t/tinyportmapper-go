package lru

import (
	"container/list"
	"sync"
	"time"
)

// node LRU 节点
type node struct {
	key     interface{}
	value   interface{}
	expTime time.Time
	element *list.Element // 双向链表中的元素
}

// Cache LRU 缓存收集器
// 对应 C++ 中的 lru_collector_t
type Cache struct {
	mp        map[interface{}]*list.Element // key → element
	li        *list.List                    // 双向链表 (头部最新,尾部最老)
	mu        sync.RWMutex
	maxSize   int
	cleanupFn func(key, value interface{}) // 清理回调
}

// NewCache 创建新的 LRU 缓存
// maxSize: 最大容量,超过时淘汰尾部。0表示无限制（仅超时淘汰，与C++行为一致）
func NewCache(maxSize int) *Cache {
	return &Cache{
		mp:        make(map[interface{}]*list.Element),
		li:        list.New(),
		maxSize:   maxSize,
		cleanupFn: nil,
	}
}

// SetCleanupFn 设置清理回调函数
func (c *Cache) SetCleanupFn(fn func(key, value interface{})) {
	c.cleanupFn = fn
}

// Get 获取值
// 对应 C++ 中的 lru_collector_t::peek_back()
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.mp[key]
	if !ok {
		return nil, false
	}
	c.li.MoveToFront(elem)
	return elem.Value.(*node).value, true
}

// GetTS 获取值和最后活跃时间
func (c *Cache) GetTS(key interface{}) (interface{}, time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.mp[key]
	if !ok {
		return nil, time.Time{}, false
	}
	c.li.MoveToFront(elem)
	n := elem.Value.(*node)
	return n.value, n.expTime, true
}

// Put 添加或更新值
// 对应 C++ 中的 lru_collector_t::new_key() 和 update()
func (c *Cache) Put(key, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// 如果已存在,更新并移动到头部
	if elem, ok := c.mp[key]; ok {
		n := elem.Value.(*node)
		n.value = value
		n.expTime = now.Add(ttl)
		c.li.MoveToFront(elem)
		return
	}

	// 新节点
	newNode := &node{
		key:     key,
		value:   value,
		expTime: now.Add(ttl),
	}
	elem := c.li.PushFront(newNode)
	c.mp[key] = elem

	// 超过最大容量时移除尾部，maxSize=0表示无限制（与C++行为一致）
	if c.maxSize > 0 {
		for c.li.Len() > c.maxSize {
			c.removeBack()
		}
	}
}

// Touch 更新节点的活跃时间
// 对应 C++ 中的 lru_collector_t::update()
func (c *Cache) Touch(key interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.mp[key]
	if !ok {
		return ErrKeyNotFound
	}

	now := time.Now()
	n := elem.Value.(*node)
	n.expTime = now.Add(ttl)
	c.li.MoveToFront(elem)
	return nil
}

// Remove 移除指定 key
// 对应 C++ 中的 lru_collector_t::erase()
func (c *Cache) Remove(key interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.mp[key]
	if !ok {
		return ErrKeyNotFound
	}
	n := elem.Value.(*node)
	delete(c.mp, key)
	c.li.Remove(elem)
	if c.cleanupFn != nil {
		c.cleanupFn(key, n.value)
	}
	return nil
}

// RemoveExpired 移除过期节点,返回移除数量
// 对应 C++ 中的 clear_inactive0()
func (c *Cache) RemoveExpired(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for {
		elem := c.li.Back()
		if elem == nil {
			break
		}
		n := elem.Value.(*node)
		if n.expTime.After(now) {
			break
		}
		delete(c.mp, n.key)
		c.li.Remove(elem)
		removed++
		if c.cleanupFn != nil {
			c.cleanupFn(n.key, n.value) // 同步调用，避免竞态条件
		}
	}
	return removed
}

// RemoveExpiredRatio 按比例清理过期节点
// 对应 C++ 中的 clear_inactive0()
func (c *Cache) RemoveExpiredRatio(now time.Time, ratio int, minCount int) int {
	if c.li.Len() == 0 || ratio <= 0 {
		return 0
	}

	numToClean := c.li.Len()/ratio + minCount
	if numToClean > c.li.Len() {
		numToClean = c.li.Len()
	}

	removed := 0
	for i := 0; i < numToClean; i++ {
		elem := c.li.Back()
		if elem == nil {
			break
		}
		n := elem.Value.(*node)
		if n.expTime.After(now) {
			break // 已按时间排序,后面的都不会过期
		}
		delete(c.mp, n.key)
		c.li.Remove(elem)
		removed++
		if c.cleanupFn != nil {
			c.cleanupFn(n.key, n.value) // 同步调用，避免竞态条件
		}
	}
	return removed
}

// PeekBack 查看尾部节点
// 对应 C++ 中的 lru_collector_t::peek_back()
func (c *Cache) PeekBack() (interface{}, interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	elem := c.li.Back()
	if elem == nil {
		return nil, nil, false
	}
	n := elem.Value.(*node)
	return n.key, n.value, true
}

// PeekBackTS 查看尾部节点及其时间戳
func (c *Cache) PeekBackTS() (interface{}, time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	elem := c.li.Back()
	if elem == nil {
		return nil, time.Time{}, false
	}
	n := elem.Value.(*node)
	return n.key, n.expTime, true
}

// removeBack 移除尾部节点 (内部方法,已加锁)
func (c *Cache) removeBack() {
	elem := c.li.Back()
	if elem != nil {
		n := elem.Value.(*node)
		delete(c.mp, n.key)
		c.li.Remove(elem)
		if c.cleanupFn != nil {
			c.cleanupFn(n.key, n.value) // 同步调用，避免竞态条件
		}
	}
}

// Len 返回当前大小
// 对应 C++ 中的 lru_collector_t::size()
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.li.Len()
}

// Empty 检查是否为空
// 对应 C++ 中的 lru_collector_t::empty()
func (c *Cache) Empty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.li.Len() == 0
}

// Clear 清空缓存
// 对应 C++ 中的 lru_collector_t::clear()
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cleanupFn != nil {
		for elem := c.li.Front(); elem != nil; elem = elem.Next() {
			n := elem.Value.(*node)
			c.cleanupFn(n.key, n.value) // 同步调用，避免竞态条件
		}
	}

	c.mp = make(map[interface{}]*list.Element)
	c.li = list.New()
}

// ErrKeyNotFound 键未找到错误
var ErrKeyNotFound = &KeyNotFoundError{}

// KeyNotFoundError 键未找到错误
type KeyNotFoundError struct{}

func (e *KeyNotFoundError) Error() string {
	return "LRU 缓存中未找到键"
}
