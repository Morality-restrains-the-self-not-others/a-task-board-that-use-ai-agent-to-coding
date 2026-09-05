# 实施计划：gitOauth → taskGitOauth

**日期：** 2026-07-15  
**设计：** `docs/superpowers/specs/2026-07-15-gitoauth-python-to-go-migration-design.md`

## 任务清单

### Phase A — 骨架与基础设施

- [x] A1. 创建 `taskGitOauth/`（go.mod、build.sh、run.sh、scripts/runall-stop.sh）
- [x] A2. 配置加载（conf/auth/git-oauth 等价 merge）
- [x] A3. SQLite 打开 `db/git-oauth/git-oauth.sqlite3`（兼容既有表，无破坏性 migrate）
- [x] A4. Fernet encrypt/decrypt + 往返单测
- [x] A5. HTTP server :8002，`GET /api/health/`

### Phase B — Internal API（优先，消费者依赖）

- [x] B1. access-for-user（含缓存、锁、provider 回退）
- [x] B2. refresh / token-use-report / delete / user-ids / summary
- [x] B3. task-credential-audit/report
- [x] B4. GitHub + GitLab 路径前缀分流

### Phase C — Browser OAuth

- [x] C1. Bridge JWT 校验
- [x] C2. Session cookie（gitoauth_sessionid）
- [x] C3. GitHub start / start-from-gateway / callback + bind
- [x] C4. GitLab start / start-from-gateway / callback + bind（provider_key）
- [x] C5. service_provider 动态 callback（404/409）

### Phase D — 文档与观测

- [x] D1. OpenAPI/Swagger 路由
- [x] D2. 意图文档 + 领域事件对照
- [x] D3. 架构 v29 三件套 + VERSION_HISTORY

### Phase E — 切流

- [x] E1. 更新 `conf/runAll.yaml` working_dir → taskGitOauth
- [x] E2. 更新语言推断 / CI / api_route_ownership 备注
- [x] E3. 废止 2026-07-05 Non-Goal 条目说明
- [x] E4. 本地 health + Go 单测通过

### Phase F — 清理 Python

- [x] F1. 移除 runAll/脚本对 `gitOauth/` Python 的依赖
- [x] F2. 删除或归档 `gitOauth/` Django 树（保留 README 指向 taskGitOauth）
- [x] F3. 全文检索确认无启动入口指向 Django gitOauth
- [x] F4. Ship：创建 PR

## 验证命令

```bash
cd taskGitOauth && ./build.sh && go test ./...
curl -sf http://127.0.0.1:8002/api/health/
```
