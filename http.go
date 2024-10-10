package distributecache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/distributeCache/consistenthash"
)

const defaultBasePath = "/Distribute_cache"
const defaultReplicas = 50

// HttpPool 代表一个分布式缓存节点的 HTTP 池
// 它包含当前服务器的地址 (self) 和所有 HTTP 请求的基础 URL 路径
type HttpPool struct {
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter
}

func NewHttpPool(self string) *HttpPool {
	return &HttpPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 函数用于记录服务器的日志信息
func (p *HttpPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 函数处理所有匹配 HttpPool 的 basePath 的 HTTP 请求
// 这个函数运行请求，并从指定的缓存组和键中查询数据
func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log("%s %s", r.Method, r.URL.Path)

	// 分割 URL 路径，将 basePath 之后的部分分割为 groupName 和 key
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 根据名称获取缓存组
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 从组中查询指定键的缓存值
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置返回的内容类型为 "application/octet-stream"，并将查询到的缓存值写入响应中
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// httpGetter 代表 HTTP 的客户端
type httpGetter struct {
	baseURL string
}

// 客户端功能，通过远程请求获取指定缓存组和键对应的值
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := h.baseURL + url.QueryEscape(group) + url.QueryEscape(key)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned : %v", res.Status)
	}
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body : %v", err)
	}
	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)

// 实例化一致性哈希算法
func (p *HttpPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.NewHash(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter)
}

// PickPeer 方法根据键选择合适的 PeerGetter,返回相应的 PeerGetter 并表示是否已找到合适的节点
func (p *HttpPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick Peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// 类型检查
var _ PeerPicker = (*HttpPool)(nil)
