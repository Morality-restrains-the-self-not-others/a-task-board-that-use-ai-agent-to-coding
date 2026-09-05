# NFR Clarification: port_config split

| 类别 | 级别 | 说明 |
|------|------|------|
| 可维护性 | L2 | 每 app 独立 YAML；碎片仅 sync 生成 |
| 一致性 | L3 | manifest 驱动跨 app 键同步；CI/doctor 可校验 GENERATED 头 |
| 安全 | L3 | 机密仍在 `config.local.yaml`（gitignore）；可提交文件不含生产密钥（迁移时保留 dev 示例） |
| 可用性 | L2 | 迁移不改变端口；克隆后 `conf-sync-all` 即可 |
| 可测试性 | L2 | 对照测试：迁移前后 dict 等价（django/task-events 子集） |
