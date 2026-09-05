# 实施计划: gitOauth DisallowedHost 网关 Host 头校验修复

> 输入:
> - 设计文档: `docs/designs/disallowed-host-gateway-fix.md`
> - 价值流: `docs/superpowers/plans/2026-06-26-disallowed-host-gateway-fix-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-26-disallowed-host-gateway-fix-nfr-clarification.md`
> - DDD: 已跳过（纯配置变更）

## 变更概览

- **文件数**: 2
- **预计代码行数**: ~20
- **风险等级**: 低（仅扩展白名单，无破坏性变更）

## 任务清单

### Task 1: 新增 `load_gateway_public_host()` 单元测试

- [ ] 创建测试文件 `gitOauth/config/test_port_config.py`
- [ ] 编写 `test_load_gateway_public_host_returns_hostname` — 验证解析 `publicBase: http://183.250.1.132:18081` 返回 `183.250.1.132`
- [ ] 编写 `test_load_gateway_public_host_empty_when_missing` — 验证 `publicBase` 缺失时返回空字符串
- [ ] 运行: `cd gitOauth && python -m pytest config/test_port_config.py -v`

### Task 2: 实现 `load_gateway_public_host()`

- [ ] 编辑 `gitOauth/config/port_config.py`
- [ ] 新增函数：读取 `conf/gateway/task-gateway/config.yaml`，解析 `publicBase` 的 hostname
- [ ] 导入 `urlparse`（settings.py 已有，port_config.py 需新增）
- [ ] 运行 Task 1 测试确认通过

### Task 3: 新增 ALLOWED_HOSTS 集成测试

- [ ] 创建（或扩展）测试文件 `gitOauth/config/test_settings_allowed_hosts.py`
- [ ] 编写测试：验证 ALLOWED_HOSTS 包含 `183.250.1.132`（从网关配置读取）
- [ ] 编写测试：验证重复添加时不会重复
- [ ] 运行: `cd gitOauth && DJANGO_SETTINGS_MODULE=config.settings python -m pytest config/test_settings_allowed_hosts.py -v`

### Task 4: 修改 `settings.py` ALLOWED_HOSTS

- [ ] 编辑 `gitOauth/config/settings.py`
- [ ] 导入 `load_gateway_public_host`
- [ ] 在 ALLOWED_HOSTS 列表定义之后追加网关公网 hostname（去重）
- [ ] 运行 Task 3 测试确认通过

### Task 5: 端到端验证

- [ ] 启动 runAll（或确认已运行）
- [ ] 浏览器访问 `http://183.250.1.132:4000/tenant/<id>/create-project/?debug=true`
- [ ] 输入 GitLab 仓库地址，点击 OAuth 授权
- [ ] 验证 `GET /api/accounts/gitlab/oauth/start-from-gateway/...` 返回 200（非 400）
- [ ] 检查 gitOauth 日志无 DisallowedHost 异常

## 依赖关系

```
Task 2 (实现) 依赖 Task 1 (测试先行) ← TDD 红→绿
Task 4 (实现) 依赖 Task 3 (测试先行) ← TDD 红→绿
Task 5 (E2E) 依赖 Task 2 + Task 4
```

## 受影响的文件

| 文件 | 操作 | 任务 |
|------|------|------|
| `gitOauth/config/port_config.py` | 编辑 — 新增 1 函数 | Task 2 |
| `gitOauth/config/settings.py` | 编辑 — 追加 3 行 | Task 4 |
| `gitOauth/config/test_port_config.py` | 新建 | Task 1 |
| `gitOauth/config/test_settings_allowed_hosts.py` | 新建/扩展 | Task 3 |
