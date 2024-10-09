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

// Default base path for HTTPPool to handle HTTP requests
const defaultBasePath = "/Distribute_cache"
const defaultReplicas = 50

// HttpPool represents an HTTP pool of distributed cache nodes
// It contains the address of the current server (self) and the base URL path
// for all HTTP requests directed at this pool.
type HttpPool struct {
	self        string // Address of this HTTP server, used for logging purposes
	basePath    string // Base URL path for this server's HTTP handler
	mu          sync.Mutex
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter //映射远程节点与对应的httpGetter
}

// NewHttpPool initializes an instance of HttpPool with the given server address.
// It sets the basePath to the default base path.
func NewHttpPool(self string) *HttpPool {
	return &HttpPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log prints server log messages with the server's address and a custom message.
// It uses variadic arguments to format the log message.
func (p *HttpPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handles all HTTP requests that match the basePath of the HttpPool.
// It processes requests to retrieve cached data from a specified group and key.
func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Ensure that the requested URL path starts with the expected basePath.
	// If not, panic to indicate an unexpected path is being accessed.
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	// Log the HTTP method and URL path for the request.
	p.Log("%s %s", r.Method, r.URL.Path)

	// Split the URL path after the basePath into two parts: groupName and key.
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		// If there are not exactly two parts, return a "bad request" error.
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Extract groupName and key from the URL parts.
	groupName := parts[0]
	key := parts[1]

	// Get the cache group by name.
	group := GetGroup(groupName)
	if group == nil {
		// If the group does not exist, return a "not found" error.
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// Retrieve the cached value for the specified key from the group.
	view, err := group.Get(key)
	if err != nil {
		// If an error occurs while retrieving the value, return an "internal server error".
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the content type to "application/octet-stream" and write the cached value to the response.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

type httpGetter struct {
	baseURL string
}

// 客户端功能
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprint(
		"%v%v%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
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

func (p *HttpPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick Peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HttpPool)(nil)
