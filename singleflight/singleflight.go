package singleflight

import "sync"

// call 类用于记录正在运行或已经结束的请求
// wg: 使用 sync.WaitGroup 来等待并发请求的结束
// val: 返回的结果值
// err: 返回的错误信息

type call struct {
	wg  sync.WaitGroup
	val any
	err error
}

// Group 用于管理一组正在运行的请求
// mu: 互斥锁，保证多人访问并发安全
// m: 一个实时请求的应用转应的键和 call 的 map

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 方法用于执行一个请求，防止同时多个经过同一键的请求
// key: 需要运行的请求的唯一标识
// fn: 运行该键的任务函数
// 返回函数的返回值，包括返回结果和错误

func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock() // 加互斥锁，以保证多人并发操作的安全
	if g.m == nil {
		g.m = make(map[string]*call) // 如果应用 map 为 nil，创建 map
	}
	if c, ok := g.m[key]; ok {
		// 如果该 key 的请求已经在 map 中，表明该请求已经被执行，就等待它结束
		g.mu.Unlock()
		c.wg.Wait()         // 等待对应请求的完成
		return c.val, c.err // 返回请求的结果
	}
	// 否则，创建一个新的 call 实例
	c := new(call)
	c.wg.Add(1)   // 添加等待计数
	g.m[key] = c  // 将这个 call 设置为当前请求的值
	g.mu.Unlock() // 锁释放

	// 执行函数，获取返回值和错误
	c.val, c.err = fn()
	c.wg.Done() // 调用 Done 来使等待的中任完成

	// 开始写销，将该请求从 map 中删除
	g.mu.Lock()
	delete(g.m, key) // 将完成的 call 从 map 中删除
	g.mu.Unlock()

	// 返回请求的值
	return c.val, c.err
}
