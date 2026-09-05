# NFR 澄清: 配置感知的 GitLab 仓库识别

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-30-config-aware-gitlab-repo-detection-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-30-config-aware-gitlab-repo-detection-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 分支预览 + catalog 加载 P95 ≤ 500ms（不含 GitLab 上游） |
| 安全性 | L3 | OAuth 仍经现有 gitOauth 链路，无新凭据存储 |
| 可用性 | L2 | catalog 加载失败降级启发式，页面不阻塞 |
| 可维护性 | L2 | 配置单源，新增 gitOauth 条目零代码 |
| 可伸缩性 | L0 | 不适用 |
| 数据一致性 | L0 | 只读查询，无写入 |

## 质量场景

### QS-01: 分支预览识别延迟

| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 项目详情页用户 |
| 刺激 | 点击分支列表预览（IP GitLab repo） |
| 制品 | GET /api/.../branches/ |
| 环境 | 正常负载 |
| 响应 | 200 + 鉴权提示或 branches |
| 响应度量 | 服务端处理 P95 ≤ 500ms（mock GitLab 上游） |

### QS-02: catalog 加载失败降级

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 前端 ProjectDetail |
| 刺激 | catalog API 503 |
| 制品 | resolveRepoOAuthProvider |
| 环境 | 降级 |
| 响应 | 启发式仍识别 github/gitlab 域名；页面可交互 |
| 响应度量 | 无 uncaught rejection；分支预览仍可用（后端已修复） |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| 配置单源 L2 | 复用 RepoUrl VO + git_oauth_providers | 新增 GitProviderDetectionService 封装识别 |
| 安全 L3 | 无新聚合 | 不引入凭据实体 |

## 权衡与边界

- 不做 catalog 持久化缓存（仅 session 内存）
- IP 站点不支持 _gitlab_session cookie 兜底（不变）
- 不做授权后自动刷新分支预览

## 跳过声明

- 可伸缩性/数据一致性：只读配置匹配，无分布式写入。
