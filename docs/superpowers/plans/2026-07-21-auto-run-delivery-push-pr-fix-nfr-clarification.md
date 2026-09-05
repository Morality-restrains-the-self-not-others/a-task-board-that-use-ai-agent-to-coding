# NFR — auto_run 交付修复

| 类别 | 级别 | 说明 |
|------|------|------|
| 可靠性 | L2 | 失败可重试（默认 3 次）+ 进程重启补跑 |
| 可观测性 | L2 | `AUTO_RUN_DELIVERY_*` runtime-event（须经 Cloud，非 Django 404） |
| 安全 | L2 | 不新增凭证面；日志不输出 token |
| 性能 | L1 | 交付异步；补跑仅扫 completed auto_run_first |

无金融/鉴权面升级 → 不抬到 L3。
