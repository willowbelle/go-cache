package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 类型，用于进行数据的哈希值计算
type Hash func(data []byte) uint32

// Map 为一致性哈希的实现
// 包含了运行哈希算法的函数，并指定的转换节点个数和哈希结果序列
// hash: 用来计算哈希值的函数
// replicas: 虚拟节点的倍数
// keys: 哈希环（虚拟节点的哈希值）
// hashMap: 虚拟节点和实际节点的对应关系
type Map struct {
	hash     Hash
	replicas int            // 虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点与实际节点的映射
}

// NewHash 函数用于创建一个新的哈希场景
// replicas: 虚拟节点的倍数
// fn: 哈希函数，用来计算哈希值
func NewHash(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE // 如果未指定哈希函数，则使用默认的循环冷余校验算法
	}
	return m
}

// Add 方法用于添加实际节点，并为每个节点生成指定倍数的虚拟节点
// keys: 可变的实际节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) // 生成虚拟节点的哈希值
			m.keys = append(m.keys, hash)                      // 将哈希值添加到哈希环中
			m.hashMap[hash] = key                              // 将哈希值与实际节点应用
		}
	}
	sort.Ints(m.keys) // 按哈希值序列排序
}

// Get 方法根据指定的 key 找到最近的节点
// key: 需要查询的键
// 返回实际节点的名称
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return "" // 如果哈希环为空，返回空字符串
	}
	hash := int(m.hash([]byte(key))) // 计算输入 key 的哈希值
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	}) // 查询哈希环中最接近的节点

	return m.hashMap[m.keys[idx%len(m.keys)]] // 返回最近节点的实际节点
}
