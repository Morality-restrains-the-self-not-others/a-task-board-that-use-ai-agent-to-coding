# 角色权限分析 — 9999 未 migrate 标注

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-runall-pending-migrate-badge-design.md`

## 结论

不引入新角色。:9999 仍是内网运维台；新接口为只读，权限边界与现有 `/api/dev/logs` 同级（能打开 Status 页即可读），**不**套用 `allowDevDatabaseReset`（那是防误清库，不是防读巡检）。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/dev/migrate-status` | 能访问 :9999 的运维（内网） | System / 各库 `data_migrate_log` | read | 无租户鉴权（与 `/api/status` 一致） | ✅ 充分 | 不公开到公网网关；禁止返回 DSN/密码 |
| POST `/api/dev/init-databases` | 同上 | System | write | confirm=INIT_ALL + `allowDevDatabaseReset` | ✅ 充分 | 不改 |
| UI 徽章 | 同上 | System | read | 同源 :9999 | ✅ 充分 | 徽章点击不触发 POST |

## 未引入新角色

```
superuser / 租户角色 — 不适用（本页不是 SaaS 控制台）
└─ 系统运维（runAll Status，内网）
```

## 安全审计

- [x] 无 IDOR（无资源 id 参数）
- [x] 无跨租户业务数据（只读追踪表文件名）
- [x] 库名来自 registry，不接受 query 指定任意 DB（防探测）
- [x] 错误信息不含连接口令
- [x] 写路径仍需 confirm token

## 权限测试

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 运维打开 9999 | 内网 | GET migrate-status | 200 |
| 任意 query db= | 内网 | GET ?db=mysql | 忽略，仍扫 registry 全量 |
| 公网用户 | n/a | 不经 APISIX 暴露本 API | 保持现状（9999 不进业务网关） |
