# NFR 澄清: taskAuth OIDC Issuer Docker 网络可达性修复

> 输入:
> - 设计文档: `docs/specs/taskauth-oidc-issuer-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-24-taskauth-oidc-issuer-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L2 | OIDC issuer URL 正确路由，客户端与 Provider 双向校验不降级 |
| 可用性 | L2 | SSO 登录链路在 Docker 部署下可达（connection refused → 200 OK） |
| 容错机制 | L2 | issuerURL() 多层 fallback 确保配置缺失时不返回不可路由地址 |
| 可维护性 | L2 | 配置参数化（`${GITLAB_EXTERNAL_HOST}`），消除硬编码 localhost |

## 跳过声明

以下 NFR 类别与本修复不相关——这是一个配置 Bug 修复，不引入新数据流、新性能路径或新合规要求：

| 类别 | 跳过理由 |
|------|---------|
| 性能 | 无新增请求路径；OIDC 发现端点已存在，响应时间不变 |
| 可伸缩性 | 单实例 taskAuth + 单 GitLab 容器，无扩展场景 |
| 数据一致性 | 无新增数据写入；issuer 值为只读配置 |
| 可观测性 | 现有 trace/log 覆盖已有 OIDC 端点；无需新增 |
| 合规与隐私 | 不涉及数据本地化或行业合规 |
| 法律法规 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: OIDC Issuer URL Docker 可达 (Thin Slice)

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: OIDC issuer URL 必须与 GitLab 配置的 issuer 完全一致（OIDC 协议强制要求，否则身份令牌验证失败）
- **质量场景**: QS-01

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: `docker exec gitlab curl http://${GITLAB_EXTERNAL_HOST}:8003/.well-known/openid-configuration` 返回 200；完整 SSO 登录流程可完成
- **质量场景**: QS-02

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: 即使 `oidc.issuer` 配置项缺失或为空，issuerURL() 仍能从 GatewayPublicBase 提取正确的 host 构造可路由 issuer URL
- **质量场景**: QS-03

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: GitLab 容器重建、端口变更后，只需修改 `conf/git-service/config.yaml` 的 host 字段，OIDC issuer 自动跟随变化
- **质量场景**: QS-04

## 质量场景

### QS-01: OIDC issuer 一致性校验
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | GitLab (OIDC Client) 发起 OIDC 发现请求 |
| 刺激 | GET `/.well-known/openid-configuration` |
| 制品 | taskAuth OIDC 发现端点 |
| 环境 | 正常 |
| 响应 | 返回的 `issuer` 字段与 GitLab 配置的 `GITLAB_OIDC_ISSUER` 完全一致（协议级要求） |
| 响应度量 | `curl -s http://${HOST}:8003/.well-known/openid-configuration \| jq .issuer` 输出等于 `${GITLAB_OIDC_ISSUER}` |

### QS-02: 容器内 OIDC Provider 可达性
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | GitLab 容器内 OmniAuth 模块 |
| 刺激 | 发起 OIDC 发现的 TCP 连接 |
| 制品 | taskAuth OIDC 发现端点 (宿主机 :8003) |
| 环境 | GitLab 运行在 Docker 容器内 (172.21.0.2)，taskAuth 运行在宿主机 |
| 响应 | TCP 连接成功，返回 HTTP 200 + JSON 发现文档 |
| 响应度量 | `docker exec gitlab curl -sS -o /dev/null -w '%{http_code}' ${GITLAB_OIDC_ISSUER}/.well-known/openid-configuration` = `200` |

### QS-03: issuerURL() 多层 fallback 不返回不可路由地址
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | taskAuth 启动（配置可能不完整） |
| 刺激 | `oidc.issuer` 为空 + `GatewayPublicBase` 可用 |
| 制品 | `issuerURL()` 函数 (taskAuth/src/oidc_handlers.go) |
| 环境 | 配置缺失场景（如新环境部署） |
| 响应 | 从 `GatewayPublicBase` 提取 host，拼接 taskAuth 自身端口 (8003)，返回可路由 issuer URL |
| 响应度量 | 返回值不以 `0.0.0.0` 或 `127.0.0.1` 开头；返回的 host 与 `conf/gateway/task-gateway/config.yaml` 的 `publicBase` host 一致 |

### QS-04: 配置参数化 — 端口变更后 OIDC issuer 自动跟随
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 运维人员修改 `conf/git-service/config.yaml` 的 `host` 字段 |
| 刺激 | 将 host 从 `183.250.1.132` 改为 `192.168.1.100` |
| 制品 | `gitService/run.sh` → 导出 `GITLAB_OIDC_ISSUER` → `docker-compose.yml` 注入 |
| 环境 | 配置变更 |
| 响应 | GitLab 容器重建后 `GITLAB_OIDC_ISSUER` 自动使用新的 host |
| 响应度量 | 容器内 `echo $GITLAB_OIDC_ISSUER` = `http://192.168.1.100:8003` |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L2 — issuer 一致性 | OIDC Provider 的 `issuer` 是值对象，其不变式是「必须为可路由 URL」 | 无需新实体；`issuer` 值对象校验增强 |
| 容错 L2 — fallback 链 | `issuerURL()` 是领域服务，需要多层 fallback 策略 | 无需新仓储接口；逻辑在基础设施层 (config.go) |
| 可用性 L2 — Docker 网络可达 | 无新领域概念；可达性是部署/运维关注点 | DDD 建模无需处理（基础设施关注点） |

**结论**: 本修复不引入新领域实体或聚合。`issuer` URL 作为值对象已存在于 OIDC Provider 上下文中。代码变更集中在基础设施层 (config loading + env injection)。

## 权衡与边界

### 取舍
- **选择动态参数化** (`${GITLAB_EXTERNAL_HOST}` 占位符) 而非硬编码外部 IP——牺牲简单性换取可移植性
- **选择多层 fallback** (显式 issuer → GatewayPublicBase host 提取 → 127.0.0.1) 而非单一 fallback——增加代码复杂度换取容错性

### 明确不做什么
- **不引入服务发现**（如 Consul/DNS SRV）——当前部署规模不需要
- **不做 TLS**——OIDC issuer 仍为 `http://`，内部 Docker 网络通信不需要 TLS 终止
- **不修改 OIDC 协议交互逻辑**——仅修复 URL 配置，授权码流和 Token 交换逻辑不变

### 升级触发条件
- 当 taskAuth 需要支持多实例部署时 → issuer URL 须通过外部 LB/域名发布 → 升级为 `https://` + 正式 DNS
- 当 GitLab 部署到远程主机（非本机 Docker）时 → `GITLAB_OIDC_ISSUER` 须使用公网可达的域名
