# onlineServiceJS Grafana 日志转发 — 实施计划

> Design: `docs/superpowers/specs/2026-05-29-online-servicejs-grafana-log-forwarding-design.md`

## Task 1: 子进程 JSON 透传 ✅
- [x] `tracelog.ForwardChildLine` 结构化透传
- [x] `appendSubprocessLog` 与 relay `appendLog` 分离
- [x] `tracelog_test.go` 透传/包装测试

## Task 2: 多行堆栈合并 ✅
- [x] `subprocess_log_grouper.go`
- [x] `process.go` scanner 集成 grouper
- [x] `subprocess_log_grouper_test.go`

## Task 3: 多行 msg JSON 编码测试
- [x] `ForwardChildLine` 多行 `msg` 单条 JSON 测试
- [x] Run: `cd go_relayToTrae && go test ./...`

## Task 4: 价值流与 runbook
- [x] `value-stream.yaml` 新增 `increment2-relay-child-log-forwarding`
- [x] 更新 `2026-05-29-grafana-distributed-tracing-runbook.md` 故障排查表

## Task 5: 验收
- [x] `go test ./...` 全绿
- [x] go_relayToTrae 本地 commit（无 remote，PR 待配置）
