package distributecache

// 保证并发安全
import (
	"sync"

	"github.com/distributeCache/lru"
)

// 缓存类实体
// 包含了一个并发锁和 LRU 缓存的新层封
type cache struct {
	mu         sync.Mutex // 用于保证缓存操作的并发安全
	lru        *lru.Cache // 包含 LRU 缓存的指针
	cacheBytes int64      // 最大可使用的缓存容量
}

// add 函数用于添加一个键值对应到缓存中
// 在添加前使用并发锁确保并发安全
// 如果 LRU 缓存还没有创建，则将它延迟创建
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()         // 上锁确保下面的操作并发安全
	defer c.mu.Unlock() // 在函数返回时释放锁
	if c.lru == nil {
		c.lru = lru.NewCache(c.cacheBytes, nil) // 延迟创建 LRU 实例，创建时还没有使用存储过
	}
	c.lru.Add(key, value) // 添加键值对应
}

// get 函数通过键获取对应的值
// 如果 LRU 还没有创建，则返回空
// 在获取前使用并发锁保证并发安全
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()         // 上锁确保下面的操作并发安全
	defer c.mu.Unlock() // 在函数返回时释放锁
	if c.lru == nil {
		return // 如果 LRU 还没有创建，返回空值
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok // 转为 ByteView 类型并返回
	}
	return
}
