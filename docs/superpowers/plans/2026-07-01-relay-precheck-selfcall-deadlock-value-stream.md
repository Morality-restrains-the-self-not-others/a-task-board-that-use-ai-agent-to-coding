# Value Stream: relay 预检 self-call 死锁修复（进程内直接调用）

> Derived from design: `docs/superpowers/specs/2026-07-01-relay-precheck-selfcall-deadlock-design.md`

## Value Summary

runAll 部署模式下（`--noreload` 单线程 Django），直启 relay 预检不再因 HTTP 自调用死锁超时（8s → 502），预检改为进程内直接调用凭证构建逻辑，响应时间从 ~8s 降为 ~100ms。

## Related Value Streams

- **relay-precheck-internal-origin**: **extension** — 本流是该流的延续，实现其"非目标"第 3 条（进程内直接调用）。原流的 Increment 1 将 origin 从公网网关固定为 `internalApiBase`；本流进一步消除 HTTP 自调用——`internalApiBase` 配置字段从预检路径移除，`_resolve_relay_precheck_task_api_origin()` 函数删除。
- **task-detail-repo-clone-credentials-contract**: 不变 — 凭证构建逻辑 `_build_repo_clone_credentials_with_diagnostics` 保持不变，仅调用方式从 HTTP → 函数调用
- **task-detail-oauth-binding-guidance**: 前端消息分流逻辑简化 — 不再有"连接失败"引导（5xx 非 token-refresh 场景统一为"预检服务异常"）

## End-to-End Flow

```
[用户点击直启「启动」]
  → [token-init: 签发 ACCESS_TOKEN]
  → [precheck: 进程内调用 _build_credentials_for_precheck()]
       ├─ 200: 凭证完整 → 继续 start
       ├─ 409: 缺凭证 → 前端展示缺失仓库引导
       └─ 502: token 换发失败 → 前端展示精准确认信息
  → [start: 启动 Go relay worker]
  → [容器启动 → reachability 注册 → SSE 状态推送]
```

## Value Increments

### Increment 1: 预检进程内直接调用（Thin Slice）
**Value to user:** runAll 单线程模式下直启预检不再死锁超时（8s → <1s），token-init 成功后预检立即返回。  
**Scope:**
- `relay_to_trae_proxy.py`: 重构 `relay_to_trae_repo_credentials_precheck()`，移除 `requests.post` 自调用，改为直接调用凭证构建编排函数
- `relay_to_trae_proxy.py`: 删除 `_resolve_relay_precheck_task_api_origin()`（不再需要）
- `container_task_detail_views.py`: 提取 `_build_credentials_for_precheck()` 编排函数（薄封装，复用 `_build_repo_clone_credentials_with_diagnostics`）
**Depends on:** 无（前序流 `relay-precheck-internal-origin` 已部署）

### Increment 2: 预检错误消息精准化（Core Value）
**Value to user:** 5xx 错误不再显示误导性"请确认 Django 服务（如 8001）已运行"，改为精准提示。  
**Scope:** `ServerConfig.logic.vue` 第 1735-1737 行——修改 `>=500` 分支的错误文案。  
**Depends on:** Increment 1

### Increment 3: 测试覆盖更新（Essential Support）
**Value to user:** 行为变更经过完整测试回归，长期稳定。  
**Scope:**
- 更新 `test_relay_to_trae_proxy.py`: 预检测试改为 mock 编排函数（不再 mock `requests.post`）
- 删除 `test_relay_to_trae_repo_credentials_precheck_uses_internal_task_api_origin`
- 新增 `test_precheck_handles_no_project_repos`
- 新增 `test_precheck_handles_token_lookup_failure`
- Playwright E2E 回归: `TaskDetail.relay-to-trae-direct-start.playwright.test.js`
**Depends on:** Increment 1-2

## Affected Existing Streams

| Stream | Impact |
|--------|--------|
| `relay-precheck-internal-origin` | **修改** — Increment 1 替换 origin 选择逻辑为直接调用；YAML 删除 `saas-backend.django.internal_api_base` 字段（预检不再使用 HTTP origin），新增 `saas-backend.cloud_cloudserverconfig.task_id`（预检直接查询 Todo 所需） |
| `task-detail-oauth-binding-guidance` | 前端文案微调（5xx 非 token-refresh 路径提示变更） |
| `task-detail-repo-clone-credentials-contract` | 不变（凭证构建逻辑未变） |
