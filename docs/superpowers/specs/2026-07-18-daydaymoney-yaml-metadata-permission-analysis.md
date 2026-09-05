# 角色权限分析：daydaymoney.yaml 元信息全链路

**日期**: 2026-07-18  
**设计**: `2026-07-18-daydaymoney-yaml-metadata-design.md`  
**状态**: 已完成（goal-mode 自动推进）

## 1. 端点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `GET .../daydaymoney/resolve` | 租户内已认证用户 | Tenant | read | gateway JWT + tenant path | ✅ | 仅返回该 tenant 下项目；不跨租户 |
| `POST .../daydaymoney/parse-yaml` | 租户内已认证用户 | Tenant | read（无副作用） | 同项目列表 | ✅ | 不落库；禁止回显密钥类字段（YAML 无密钥） |
| `GET .../projects/?tag=` | 租户内已认证用户 | Tenant(/Workspace) | read | 同 list projects | ✅ | 与 `workspace_id` 组合时仍校验归属 |
| 项目 tags PATCH（同步） | 具备项目写权限的成员 | Project | write | 既有 update project | ✅ | 不因「来自 YAML」放宽写权限 |
| Chrome resolve 聚合多 company | 已登录用户 | User→Companies | read | 各 tenant 分别调用 | ✅ | 插件侧不得把 A 公司 match 写入 B 公司创建任务 |
| 读取页面 meta / `/daydaymoney.yaml` | 浏览器同源 | Public page | read | 无 | ✅ | 元信息非秘密；勿放 token |
| Grafana 匹配建任务 | 配置了 CaptureRule 的运维身份 | Workspace | write todo | 既有 rule 凭证 | ✅ | 仍受 rule 的 tenant/workspace 约束 |

## 2. 风险与缓解

| 风险 | 缓解 |
|------|------|
| IDOR：用 service_id 扫出他租户项目 | resolve 严格限定 path 中 `tenant_id`；插件按 company 分次调用 |
| YAML 被投毒写入敏感 id | schema 拒绝 `workspace_id` 等键；CI 校验 |
| 标签同步覆盖人工标签 | merge 去重，不删除用户已有非 `svc:` 标签（实现：union） |

## 3. 结论

无需新角色。所有新 API 复用项目列表级租户读权限；写仍走项目更新权限。可进入价值流。
