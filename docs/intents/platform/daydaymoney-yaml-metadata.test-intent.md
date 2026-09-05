# 测试意图：daydaymoney.yaml 仓库元信息全链路

**对应功能意图**: `daydaymoney-yaml-metadata.intent.md`

| # | 测试点 | 类型 | 期望 |
|---|--------|------|------|
| T1 | parse 合法 YAML | unit | 返回 service_id + 含 svc: 的 tags |
| T2 | parse 含 workspace_id | unit | 400 / 校验失败 |
| T3 | resolve 同一 service_id 两项目两 WS | unit/Go | matches 长度 ≥ 2，含不同 workspace_id |
| T4 | resolve 大小写不敏感 tag | unit/Go | 命中 |
| T5 | resolve 空查询 | unit/Go | 400 |
| T6 | projects?tag= 过滤 | unit/Go | 仅返回带该 tag 的项目 |
| T7 | 项目页 merge tags | unit/前端 | normalize 后去重合并 |
| T8 | head meta 与 YAML 一致 | CI | check 脚本通过 |
| T9 | Chrome 读 meta 后 resolve 预选 | unit | 多 match 全部勾选 |
| T10 | 日志含 daydaymoney_service_id | unit | JSON 字段存在 |
| T11 | Grafana 优先 meta 精确匹配 | unit | 有 daydaymoney 字段时不用 regex 也能命中 |
| T12 | Grafana 无 meta 回退 regex | unit | 行为与旧版一致 |
