package main

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"

	"tracelog"
)

// 轻量业务指标（OPT-20260823-062）：GitLab 流量闸门判定按 code 计数。
// 不引入外部库；/metrics 聚合 tracelog 的 HTTP RED 指标 + 本业务计数器。
// 当 8004 隧道中断时 taskBill 收不到任何 gate 请求，计数归零 ——
// Grafana 可按「活跃时段 rate 掉零」告警，避免 fail-open 静默失效。

var (
	gateMetricsMu sync.Mutex
	gateCodeCount = map[string]int64{}
)

// incGateCode 累加闸门判定计数；空 code 忽略。
func incGateCode(code string) {
	if code == "" {
		return
	}
	gateMetricsMu.Lock()
	gateCodeCount[code]++
	gateMetricsMu.Unlock()
}

func writeGateMetrics(w io.Writer) {
	gateMetricsMu.Lock()
	defer gateMetricsMu.Unlock()
	keys := make([]string, 0, len(gateCodeCount))
	for k := range gateCodeCount {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	_, _ = io.WriteString(w, "# HELP taskBill_gitlab_traffic_gate_total GitLab 流量闸门判定计数（按 code）\n")
	_, _ = io.WriteString(w, "# TYPE taskBill_gitlab_traffic_gate_total counter\n")
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "taskBill_gitlab_traffic_gate_total{code=%q} %d\n", k, gateCodeCount[k])
	}
}

// handleMetrics GET /metrics
// 聚合 tracelog HTTP RED 指标与业务计数器（Prometheus 文本格式）。
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	tracelog.WriteMetricsBody(w)
	writeGateMetrics(w)
}
