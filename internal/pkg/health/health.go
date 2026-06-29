package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Checker 检查单个依赖的健康状态。
type Checker interface {
	Check(ctx context.Context) error
}

// CheckerFunc 将函数适配为 Checker。
type CheckerFunc func(ctx context.Context) error

func (f CheckerFunc) Check(ctx context.Context) error {
	return f(ctx)
}

// Result 是健康检查响应。
type Result struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Registry 管理一组命名的 Checker。
type Registry struct {
	mu       sync.RWMutex
	checkers map[string]Checker
}

// NewRegistry 创建空的健康检查注册器。
func NewRegistry() *Registry {
	return &Registry{checkers: make(map[string]Checker)}
}

// Register 注册一个命名 Checker。
func (r *Registry) Register(name string, c Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers[name] = c
}

const checkTimeout = 3 * time.Second

// snapshot 保存检查器名称和实例的快照。
type snapshot struct {
	name    string
	checker Checker
}

// runCheck 执行单个 Checker，带超时与 panic 恢复，返回状态字符串（"ok" 或错误信息）。
func runCheck(ctx context.Context, checker Checker) (status string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			status = fmt.Sprintf("panic: %v", recovered)
		}
	}()

	checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	if err := checker.Check(checkCtx); err != nil {
		return err.Error()
	}
	return "ok"
}

// RunAll 并发执行所有已注册的 Checker，返回聚合结果。
func (r *Registry) RunAll(ctx context.Context) Result {
	r.mu.RLock()
	snapshots := make([]snapshot, 0, len(r.checkers))
	for name, checker := range r.checkers {
		snapshots = append(snapshots, snapshot{name: name, checker: checker})
	}
	r.mu.RUnlock()

	if len(snapshots) == 0 {
		return Result{Status: "ok"}
	}

	type entry struct {
		name   string
		status string
	}

	results := make(chan entry, len(snapshots))
	for _, snap := range snapshots {
		go func(s snapshot) {
			results <- entry{name: s.name, status: runCheck(ctx, s.checker)}
		}(snap)
	}

	checks := make(map[string]string, len(snapshots))
	overall := "ok"
	for range snapshots {
		e := <-results
		checks[e.name] = e.status
		if e.status != "ok" {
			overall = "degraded"
		}
	}

	return Result{Status: overall, Checks: checks}
}
