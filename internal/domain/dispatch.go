package domain

import "sync"

// Dispatch 操作分发基座：查 handlers 表执行回调，未知 op 统一报错。
// mu 非空 = 整个回调在锁内执行（纯内存快操作工具）；
// mu 为 nil = 回调自行加锁（含磁盘 I/O 等慢操作的工具，
// 避免长时间持锁阻塞 Actions/State）。
func Dispatch(mu sync.Locker, handlers map[string]OpHandler, op string, arg string) Result {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	h, ok := handlers[op]
	if !ok {
		return Result{IsError: true, Notice: "未知操作: " + op}
	}
	return h(arg)
}
