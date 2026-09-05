# NFR — GitLab 同步已购区域选择

## 路径分片键审视

| 路径 | 分片键 | 判定 |
|------|--------|------|
| `/api/tenant/{tid}/billing/gitlab-resources/` | tenant_id | 合适 |
| `/api/projects/gitlab-remote-repos/tenant_id/{tid}/?gitlab_host=` | tenant_id + host | 合适 |
| `/tenant/{tid}/projects` UI | tenant_id | 合适 |

## 幂等性审视

| 路径 | 副作用 | 级别 | 说明 |
|------|--------|------|------|
| 列已购区域 / 列远程仓 | 无 | L0 | 纯查询 |
| 区域切换重载 | 无 | L0 | 前端状态重置 |
| 批量/合并建项目 | 有 | （既有） | 本次不改 |

## 其它

- 可用性 L2：列表失败展示错误 + data-traceId；空列表引导价格页
- 安全 L2：不信任前端 host 为授权边界（换票仍走 provider 匹配）；仅展示已购 URL
