# Code Review: taskAuth 认证拆分

> 计划: `docs/superpowers/plans/2026-05-28-taskauth-split-plan.md`
> 日期: 2026-05-28

## 结论: **通过（可合并）**

无 critical 阻塞项。Important 项 2 条建议后续迭代。

## 验证证据

| 检查 | 结果 |
|------|------|
| Go test `taskAuth/src/...` | ✅ pass |
| pytest user-auth 19 项 | ✅ pass |
| valueStream `go test ./...` | ✅ pass（前序） |
| API 路径不变 | ✅ delegate 透明 |
| Token 兼容 | ✅ 同表 `accounts_customtoken` |

## Findings

### 🟡 Important: 密码重置未迁入 taskAuth
- **描述**: M4 reset-password 仍全在 Django，与「认证拆分」目标部分未完成
- **建议**: Increment 4 Task 4.1 后续 PR
- **→ 下一步**: `/7-build-构建` 按计划 Task 4.1

### 🟡 Important: pytest 与 taskAuth 集成测试缺口
- **描述**: `settings_test` 禁用桥接，TASKAUTH_ENABLED=true 路径无自动化 E2E
- **建议**: 补 `tests/test_taskauth_bridge_integration.py` 或 runAll playwright
- **→ 下一步**: `/7-build-构建` Task 4.3

### 🟢 优点
- Strangler + fallback 设计符合 NFR 可用性 L3
- internal secret 保护副作用 API
- runAll 依赖链正确（task-auth → saas-backend）

## DDD 合规

- Django 桥接在 application 层，副作用在 internal views（infrastructure）
- Go handlers 含 DB 直连（基础设施层），领域文档已标注后续提取 `taskAuth/domain/`
- 无 cross-aggregate 违规

## 安全

- internal secret 从 port_config 读取 ✅
- 密码未写入日志 ✅
- 凭证仍为前端哈希比较（与既有行为一致）
