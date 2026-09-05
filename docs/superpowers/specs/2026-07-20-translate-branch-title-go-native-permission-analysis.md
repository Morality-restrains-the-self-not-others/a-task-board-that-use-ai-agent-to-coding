# Permission Analysis: translate-branch-title Go native

- **日期**: 2026-07-20
- **关联设计**: `2026-07-20-translate-branch-title-go-native-design.md`

## 结论

无权限模型变更。公网入口仍经 task-gateway 鉴权；handler 不新增匿名面。fanyi `api_key` 仅服务端读取，不下发浏览器。

| 改动点 | 角色 | 权限边界 |
|--------|------|----------|
| POST translate-branch-title | 已登录租户成员 | 与改造前相同（网关） |
| fanyi_agent 出站 | 服务进程 | 密钥仅 conf；DirectClient 直连 |
