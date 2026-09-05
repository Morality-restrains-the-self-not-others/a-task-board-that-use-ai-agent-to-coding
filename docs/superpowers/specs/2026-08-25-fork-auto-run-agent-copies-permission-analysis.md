# 角色权限分析：Fork 自动运行按智能体（模型）复制

**日期**: 2026-08-25  
**设计**: `docs/superpowers/specs/2026-08-25-fork-auto-run-agent-copies-design.md`

## 权限影响分析

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 确认弹窗单选派生方式（纯 UI） | 能打开任务详情的工作区成员 | Workspace / Task | 无写 | 页面级鉴权 | ✅ 充分 | — |
| 智能体资源选择器（公司/工作空间/个人） | 同上 | Tenant / Workspace / User | read | 与创建任务 `ServerConfigFeatureParamsBlock` 同源 GET | ✅ 充分 | 个人配置仅列当前用户；禁止写未授权 config id |
| 智能体（模型）多选 | 同上 | FeatureParamsConfig | read | 清单来自所选配置 `supported_models`（+ 默认 `agent_model`） | ✅ 充分 | 模型名必须属于该清单；前端禁用确认；服务端 auto_run 校验 1 项非空 provider+model |
| `POST .../todos/` × N（每份 `agent_models`） | 工作区可创建任务的用户 | Workspace | write | `hasWorkspaceAccess` + `consumeTaskPostQuota` | ✅ 充分 | 每份独立扣配额；第 k 份 402 保留 1..k-1 |
| `auto_run=true` × N | 同上 | Workspace / Cloud | write + 启服 | 既有 `validateAutoRunPrerequisites` / OAuth / Git 身份 | ✅ 充分 | 不另开权限点；N 份云资源既有计费 |
| pending agent `context_pack.agent_models` | 内部 secret | Task / AIComment | write | `X-TaskAIComment-Internal-Secret` | ✅ 充分 | 不对外暴露新 public 写路径 |
| Idempotency-Key `${batch}:${i}` | 客户端 | 请求 | — | `claimTaskCreateDedup` | ✅ 充分 | i 对齐第 i 个所选模型 |

## 角色建模

不引入新角色或权限粒度。与「创建工作区任务」+「读取智能体资源配置」一致。

## IDOR / 越权

- 每份 POST 仍校验 `tenant_id` + `workspace_id` + 工作区访问。
- `personal_feature_params_config_id` 必须是当前用户可读配置（既有 feature-params 列表过滤）。禁止伪造他人个人配置 id 以套用密钥。
- 模型名不单独授权：绑定在所选配置的 `supported_models` 上。未在清单中的模型视为非法输入（400），不是新权限点。

## 滥用面

前端 cap 99 是产品上限。脚本仍可多次 Fork。缓解：既有任务帖配额与 auto_run 计费。本增量不新增后端硬顶。
