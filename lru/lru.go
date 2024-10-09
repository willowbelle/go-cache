package lru

import "container/list"

// Cache 缓存类的实现
// 使用双向链表和调用与数据的快速查找
// maxBytes: 最大可缓存的存储大小
// usedBytes: 已经使用的缓存大小
// ll: 使用的双向链表
// cache: 快速查找存储的键值对应到链表节点的实体
// OnEvicted: 退出时调用的可选功能
type Cache struct {
	maxBytes  int64
	usedBytes int64
	ll        *list.List
	cache     map[string]*list.Element
	// optional and executed when an entry is purged
	OnEvicted func(key string, value Value)
}

// 记录类
// 包含键值对值的实体
type entry struct {
	key   string
	value Value
}

// Value 接口
// 用于返回存储的值的大小
type Value interface {
	Len() int
}

// NewCache 函数创建一个新的 Cache 实例
// maxBytes: 指定最大存储大小
// oe: 退出调用的可选函数
func NewCache(maxBytes int64, oe func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: oe,
	}
}

// Get 函数通过键获取对应的值
// 如果在缓存中找到，将该节点移至双向链表的头部
// key: 需要获取的键
// value: 返回对应的值
// ok: 是否找到该键的标志
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)    // 将找到的节点移到双向链表的头部
		kv := ele.Value.(*entry) // 类型断言将 interface 转为 *entry
		return kv.value, true
	}
	return
}

// Remove 函数移除最后一个双向链表节点
// 通过移除链表的尾节点来继续使用最新使用的数据
func (c *Cache) Remove() {
	ele := c.ll.Back() // 获取双向链表的最后一个节点
	if ele != nil {
		c.ll.Remove(ele) // 从双向链表移除该节点
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                   // 从快速查找存储中删除该节点
		c.usedBytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新已使用的缓存大小
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value) // 调用退出函数
		}
	}
}

// Add 函数添加一个键值对应的结构到缓存中
// 如果该键已经存在，更新它的值并将这个节点移到链表头部
// 如果该键不存在，则创建新的节点并添加到双向链表头部
// 如果缓存超过最大大小，移除最旧的节点
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) // 将已经存在的节点移到双向链表头部
		// 更新值
		kv := ele.Value.(*entry)
		c.usedBytes += int64(value.Len()) - int64(kv.value.Len()) // 更新已经使用的缓存大小
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value}) // 创建新的节点并将它添加到双向链表头部
		c.cache[key] = ele
		c.usedBytes += int64(len(key)) + int64(value.Len())
	}
	// 移除过余的节点，直到缓存不超过最大缓存大小
	for c.maxBytes > 0 && c.usedBytes >= c.maxBytes {
		c.Remove()
	}
}

// Len 函数返回双向链表的节点数
func (c *Cache) Len() int {
	return c.ll.Len()
}
