# 实施计划: gitOauth HTTP Client Proxy Bypass Fix

> 输入:
> - 设计文档: `docs/specs/oauth-gitlab-token-exchange-fix-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-27-gitoauth-http-client-proxy-bypass-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-27-gitoauth-http-client-proxy-bypass-nfr-clarification.md`
> - DDD: N/A (基础设施修复)

## 任务清单

### T1: 创建 `gitOauth/api/http_client.py`（trust_env=False session）

- [ ] 创建 `gitOauth/api/http_client.py`
- [ ] 实现 `_get_http_session()` — thread-local `requests.Session`，`trust_env=False`
- [ ] Session 复用（thread-local 单例模式，参考 `task2app/Saas_project/core/utils/http_client.py`）
- [ ] 暴露 `session` 模块级变量供其他模块导入（或保持 `_get_http_session()` 函数）

**验证:** `python -c "from api.http_client import _get_http_session; s = _get_http_session(); assert s.trust_env == False"`

### T2: 替换 `gitlab_tokens.py` 中的 requests 调用

- [ ] 将 `import requests` 替换为 `from .http_client import _get_http_session`
- [ ] `exchange_authorization_code_for_tokens()`: `requests.post(...)` → `_get_http_session().post(...)`
- [ ] `refresh_user_access_token()`: `requests.post(...)` → `_get_http_session().post(...)`
- [ ] `fetch_gitlab_user_profile()`: `requests.get(...)` → `_get_http_session().get(...)`
- [ ] 增强异常日志：区分 `requests.exceptions.ProxyError`、`requests.exceptions.ConnectionError`、`requests.exceptions.HTTPError`，记录 status_code 和 body 摘要（NFR QS-03）

**验证:** 单元测试 mock session 验证调用路径

### T3: 替换 `github_tokens.py` 中的 requests 调用

- [ ] 将 `import requests` 替换为 `from .http_client import _get_http_session`
- [ ] `exchange_code_for_token()`: `requests.post(...)` → `_get_http_session().post(...)`
- [ ] `refresh_user_access_token()`: `requests.post(...)` → `_get_http_session().post(...)`
- [ ] `fetch_github_user_profile()`: `requests.get(...)` → `_get_http_session().get(...)`

### T4: 替换 `gitlab_browser_views.py` 中的 requests 调用

- [ ] `GitlabOAuthCallbackView.get()` 中 `requests.post(bind_url, ...)` → `_get_http_session().post(...)`

### T5: 替换 `github_browser_views.py` 中的 requests 调用

- [ ] `GithubOAuthCallbackView.get()` 中 `requests.post(bind_url, ...)` → `_get_http_session().post(...)`

### T6: run.sh NO_PROXY 加固（防御纵深）

- [ ] 在 `gitOauth/run.sh` 第 8 行 `NO_PROXY` 中追加 `183.250.1.132`
- [ ] 保留现有 `127.0.0.1,localhost`

### T7: 编写 `test_http_client.py` 测试

- [ ] **test_session_trust_env_false**: 验证 session 的 `trust_env` 属性为 `False`
- [ ] **test_session_thread_local**: 验证不同线程获取不同的 session 实例
- [ ] **test_session_same_thread_same_instance**: 验证同一线程多次调用返回同一 session
- [ ] **test_session_not_reading_proxy_env**: mock 环境变量 `ALL_PROXY`，验证 session 不读取代理设置

### T8: 运行现有测试确保无回归

- [ ] `cd gitOauth && python manage.py test api.tests -v2`
- [ ] 确认所有现有测试通过（mock `requests.post` 的测试需适配为 mock session）

### T9: 端到端验证

- [ ] 启动服务 (`runAll` 或手动 `run.sh`)
- [ ] 设置 `ALL_PROXY=socks5://127.0.0.1:7890` 模拟代理不可用
- [ ] 执行 OAuth 完整流程，确认 token 交换成功
- [ ] 或运行 `node e2e-tests/oauth-full-trace.js` 验证 `gitlab=ok`

## 依赖顺序

```
T1 (http_client.py) → T2, T3, T4, T5 (替换调用)
                    → T7 (测试)
T6 (run.sh) 可独立执行
T8 (回归测试) ← T2, T3, T4, T5
T9 (E2E) ← T1-T8
```

## 预估

| 任务 | 预估时间 | 类型 |
|------|---------|------|
| T1 | 5 min | 新建 |
| T2 | 10 min | 修改 |
| T3 | 5 min | 修改 |
| T4 | 2 min | 修改 |
| T5 | 2 min | 修改 |
| T6 | 1 min | 修改 |
| T7 | 10 min | 测试 |
| T8 | 5 min | 验证 |
| T9 | 10 min | E2E |
| **合计** | **~50 min** | |
