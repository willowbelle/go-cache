package distributecache

import (
	"fmt"
	"log"
	"sync"

	"github.com/distributeCache/singleflight"
)

// Getter 接口用于获取指定键的数据
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 函数类，实现 Getter 接口
// 便于通过函数进行数据的获取
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 属于缓存名称空间，用于缓存分类和数据加载的分发
// name: 缓存空间名称
// getter: 获取数据的 Getter
// mainCache: 主缓存中的数据
// peers: PeerPicker 用于选择同伴
// loader: singlefight
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(key string) *Group {
	mu.RLock()
	g := groups[key]
	mu.RUnlock()
	return g
}

// RegisterPeers 方法将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPick called more than once") // 禁止重复注册同伴
	}
	g.peers = peers
}

// Get 方法根据键获取缓存中的数据;如果缓存中已经存在该值，将返回,否则将加载这个数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key isn't existed")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Cache hit]")
		return v, nil
	}
	return g.load(key) // 如果缓存中不存在，通过回调函数进行加载
}

// load 方法用于引入该键的值
// 该值会从临远程同伴加载，不能从同伴中加载，则调用当前空间进行加载
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (any, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[Cache] failed to get from peer") // 无法从同伴获取
			}
		}
		return g.getLocally(key) // 此地进行数据加载
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// getFromPeer 方法用于从同伴中获取指定键的数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key) // 远程同伴的 HTTPGetter
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

// getLocally 方法用于此地加载数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, nil
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
