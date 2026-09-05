package main

import (
	"errors"
	"sync"
	"time"
)

// OPT-20260823-035：管理端用微信支付单号查本地 0 条时兜底调用 Native
// QueryOrderById。为防客服连点/脚本刷 28 位数字打到微信查单配额，加单进程
// 限流（默认 2 次/秒）+ 熔断（连续失败达到阈值后短路一段时间）。限流/熔断
// 均为内存态、按进程生效，满足内部兜底入口的放大防护。

// 包级变量以便单测收窄（限流间隔/阈值/熔断冷却）。
var (
	wechatQueryMinInterval      = 500 * time.Millisecond // 2 QPS
	wechatQueryCircuitThreshold = 5
	wechatQueryCircuitCooldown  = 60 * time.Second
)

var (
	errWechatQueryRateLimited = errors.New("微信查单请求过于频繁，请稍后重试")
	errWechatQueryCircuitOpen = errors.New("微信查单临时不可用，请稍后重试")
)

type wechatQueryGuard struct {
	mu          sync.Mutex
	nextAllowed time.Time
	failures    int
	openUntil   time.Time
}

var wechatQueryGuardInst = &wechatQueryGuard{}

// acquire 返回 nil 则放行并把下次允许时刻推到 now+minInterval；超限返回
// errWechatQueryRateLimited，熔断打开期返回 errWechatQueryCircuitOpen。
// 熔断打开期过后自然进入半开（下一次调用放行探测，成功即关闭）。
func (g *wechatQueryGuard) acquire(now time.Time) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.openUntil.IsZero() && now.Before(g.openUntil) {
		return errWechatQueryCircuitOpen
	}
	if now.Before(g.nextAllowed) {
		return errWechatQueryRateLimited
	}
	g.nextAllowed = now.Add(wechatQueryMinInterval)
	return nil
}

func (g *wechatQueryGuard) recordSuccess(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.failures = 0
	g.openUntil = time.Time{}
}

func (g *wechatQueryGuard) recordFailure(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.failures++
	if g.failures >= wechatQueryCircuitThreshold {
		g.openUntil = now.Add(wechatQueryCircuitCooldown)
		g.failures = 0
	}
}

func (g *wechatQueryGuard) reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nextAllowed = time.Time{}
	g.failures = 0
	g.openUntil = time.Time{}
}
