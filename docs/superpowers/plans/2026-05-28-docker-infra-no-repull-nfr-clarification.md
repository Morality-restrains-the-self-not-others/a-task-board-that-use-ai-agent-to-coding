# NFR 澄清: docker-infra 禁止重复拉镜像

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-28-docker-infra-no-repull-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-28-docker-infra-no-repull-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 镜像已齐时启动/重启 ≤30s 至 8080 healthy |
| 可用性 | L2 | 本地 dev 栈可重复启停，不因 pull 卡死 |
| 可维护性 | L2 | 与 AiMonitor run.sh 模式一致 |
| 安全性 | L0 | 不涉及 |
| 可伸缩性 | L0 | 单开发者 Docker Desktop |
| 数据一致性 | L0 | 无业务数据 |

## 质量场景

### QS-01: 镜像已齐重启
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 开发者 |
| 刺激 | UI 重启 docker-infra，四类镜像均已 inspect 成功 |
| 制品 | run-infra.sh + runAll runner |
| 环境 | 正常，Docker Desktop 运行 |
| 响应 | stderr 无 `Pulling`/`Downloading`；120s 内 8080 OK |
| 响应度量 | 日志 grep + curl 8080 |

### QS-02: 首次缺镜像
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激 | 本地无 cp-kafka 镜像，UI 启动 |
| 响应 | 脚本执行 compose pull 一次后 up |
| 响应度量 | 最终 healthy；仅首次出现 Pulling |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| L2 性能启停 | 生命周期脚本与 runner 解耦 | 脚本承担 StackLifecycle；runner 仅探测 URL |
| L0 安全 | 无 | 跳过实体建模 |

## 权衡与边界

### 明确不做什么
- 不做镜像缓存代理/离线 registry
- 不重构全部 Docker 托管服务

### 升级触发条件
- 多环境镜像版本矩阵 → 引入 `.env` 或 manifest 文件

## 跳过声明
- 安全性、合规、可伸缩性：本地 dev 基础设施，不适用。
