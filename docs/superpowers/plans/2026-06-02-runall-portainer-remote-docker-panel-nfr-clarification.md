# NFR 澄清: runAll Portainer 远程 Docker 管理面板

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-02-runall-portainer-remote-docker-panel-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-02-runall-portainer-remote-docker-panel-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | UI 链接加载 < 1s；Portainer 由独立服务承担 |
| 可用性 | L1 | Portainer skip 失败，不阻塞 redis/kafka |
| 安全性 | L2 | LAN only；Portainer admin 密码；socket 权限仅限 dev CPU |
| 可维护性 | L2 | 镜像 pin + pull_policy if_not_present |
| 可观测性 | L1 | runAll HTTP 探活 `/api/status` |
| 可伸缩性 | L0 | 单开发者单 CPU |
| 数据一致性 | L0 | 无业务数据 |
| 合规 | L0 | Portainer CE zlib 许可 |

## 质量场景

### QS-01: Portainer 可选不阻塞 infra
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L1 |
| 刺激源 | 开发者点击「全部启动」 |
| 刺激 | docker-portainer 启动失败 |
| 制品 | runAll infrastructure 组 |
| 环境 | 正常 |
| 响应 | docker-redis/kafka 仍继续启动；UI 显示 portainer skipped |
| 响应度量 | docker-redis TCP 120s 内 healthy |

### QS-02: TCP 端口可点击
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者查看 runAll UI |
| 刺激 | docker-redis healthy |
| 制品 | status.html 端口列 |
| 环境 | remote_docker.host 已配置 |
| 响应 | 显示 `172.20.10.7:6379` 可点击链接 |
| 响应度量 | Go ui_test + 嵌入 HTML 断言 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| L1 可用性 skip | Portainer URL 可选 | InfraManagementPanelURL 允许空值 |
| L2 安全 LAN | URL 仅 http host:port | URL VO 校验 scheme/host |

## 权衡与边界

### 明确不做什么
- 不在 runAll 内嵌容器 CRUD
- 不做 Portainer 自动初始化 admin

### 升级触发条件
- 多开发者共 CPU → Portainer RBAC L3

## 跳过声明
- 可伸缩性、数据一致性、合规（除 license 已确认）：开发工具特性，无业务 NFR。
