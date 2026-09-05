# Value Stream: 修复 git-service GitLab 容器 crash loop（模板变量未解析）

> Derived from design: `.claude/skills/1-brainstorming-设计文档/design.md`

## Value Summary

开发者启动 runAll 后，git-service 能正常启动 GitLab 容器并完成 bootstrap，不再因 `${subdomains.gitlab}` 模板变量未解析导致容器 crash loop。

## Related Value Streams

- **git-service-mkdir-permission-fix** (2026-06-30): 前置依赖 — 该修复使启动流程能到达配置加载阶段
- **gitlab-oauth-app-bootstrap-fix** (2026-06-22): 相关 — 同一 bootstrap 流程中的不同环节

## End-to-End Flow

[runAll 触发 git-service DAG 启动] → [run.sh load_gitservice_config 读取配置] → [检测未解析模板 → 回退到 host] → [docker-compose 传入具体 hostname] → [GitLab 容器正常初始化] → [bootstrap 脚本执行] → [服务就绪]

## Value Increments

### Increment 1: 模板回退检测（唯一增量）

**Value to user:** GitLab 容器正常启动，不再 crash loop。

**Scope:**
- `gitService/run.sh` Python 内联脚本：`allowed_host` 包含 `${` 时清空，使 `hostname` 回退到 `host`（`127.0.0.1`）

**Depends on:** `git-service-mkdir-permission-fix`（前置修复）

## Fields Impact

无数据字段变更。纯 Python 内联脚本防御逻辑。

## Test Impact

无自动化测试 — 手动验证。
