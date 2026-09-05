# 计划：25 天分账兜底

## 切片

1. **Red**：`wechat_profit_sharing_fallback_test.go` — 26d pending 未来 settle 仍执行；24d 不执行；failed 同单号重试；finished 不打微信；重放只成功一次；>30d 不执行；无行有边则补行并立即执行。
2. **Green**：`wechat_profit_sharing_fallback.go` + 把 `processPendingProfitSharings` 抽到 `wechat_profit_sharing_scan.go`（原文件已 >500 行），合并 due + fallback，去重后 `executeProfitSharing`。
3. **文档**：意图 / 设计 / NFR / 权限 / 价值流 T59。
4. **验证**：`go test` 本包相关测例；gofmt。

## 完成标准

- 默认 8 天延迟未改
- 无新 HTTP、无新 timer、无新 MQ
- 资金路径同一 `out_profit_sharing_no`
