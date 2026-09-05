# NFR Clarification: conf 硬化 Phase 4.5

| 属性 | 级别 | 场景 |
|------|------|------|
| 可维护性 | L2 | `conf-sync` 后零 diff；碎片禁止手改 |
| 可靠性 | L2 | `write_port_config` 后自动 sync；加载失败回退默认端口 |
| 安全性 | L2 | local+example 文档；**不**本期迁出明文密钥（D2=A） |
| 可测试性 | L2 | pytest 覆盖 conf-read、sync CI、loader |
| 性能 | L1 | Playwright 直读 YAML，避免每进程 exec Python 聚合 |
