# Feature-Params 访问控制 — 角色权限分析

- **日期**: 2026-07-19
- **依赖**: `2026-07-19-feature-params-access-control-design.md`

## 角色矩阵

| 角色 | 公司 full GET/POST | 工作空间 full GET/POST | summary GET | 审计可见 |
|------|-------------------|------------------------|-------------|---------|
| 非成员 | 403 | 403 | 403 | 记 denied（若已鉴权） |
| 活跃成员（非设置 context） | 403 | 403 | 200 脱敏 | 记 summary/denied |
| 活跃成员 + company_settings | 200/写需 admin | — | 可用 | 记 full |
| 活跃成员 + workspace_settings + workspace 访问权 | — | 200/写需 ws/tenant admin | 可用 | 记 full |
| 租户 admin | 同成员 + POST 允许 | 同左 | 同左 | 同左 |

写权限（admin）规则保持不变，仅叠加 access_context。

## 威胁模型

| 威胁 | 缓解 |
|------|------|
| 伪造 Referer | 不采用 Referer 鉴权 |
| 伪造 Access-Context 头 | 任意成员仍可声称 settings；**真正机密靠「非设置场景改走 summary」+ 审计追责**。完整密钥对成员本就可在设置页看见，目标是阻断任务详情等无意/隐蔽拉取 |
| 脚本直接拉 full | 需显式 header；审计可追溯；后续可收紧为仅 admin full |

## 数据分类

- **机密**：`api_key`、`extra_env_vars[].value`、env_preview 中密钥类变量
- **可公开给成员（summary）**：provider 名、supported_models、默认模型、`llm_budget_enabled`、`budget_enabled` 标志
