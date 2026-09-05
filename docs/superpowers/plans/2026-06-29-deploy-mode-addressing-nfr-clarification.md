# NFR 澄清: DEPLOY_MODE 寻址模式切换

> 输入:
> - 设计文档: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-29-deploy-mode-addressing-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L0 | 不适用 — 配置解析在启动时完成，不影响请求路径延迟 |
| 可伸缩性 | L0 | 不适用 — 无数据增长或流量扩展关注点 |
| 可用性 | L1 | 基础 — `base.yaml` 是启动必需文件，缺失则启动失败并明确报错 |
| 安全性 | L1 | 基础 — domain 模式为 TLS 提供寻址基础，但 TLS 终止本身不在本范围内 |
| 数据一致性 | L0 | 不适用 — 无数据存储或跨服务事务 |
| 容错机制 | L1 | 基础 — 配置格式错误时启动失败并给出精确错误信息 |
| 可观测性 | L1 | 基础 — 记录当前 DEPLOY_MODE 和解析后的寻址方案 |
| 合规与隐私 | L0 | 不适用 — 无数据处理或存储变更 |
| 可维护性 | L2 | 标准 — 向后兼容 DEPLOY_MODE=local；模板变量统一管理寻址 |

## 逐增量 NFR 分析

### Increment 1: Python 配置解析核心

**NFR 类别: 可维护性**
- **等级**: L2 - 标准
- **量化目标**: 默认 `DEPLOY_MODE=local` 下所有现有测试通过，无回归
- **说明**: `base.yaml` 是寻址的单一真相源，所有寻址字段必须经 `resolve_template_vars()` 解析

**NFR 类别: 可观测性**
- **等级**: L1 - 基础
- **量化目标**: 启动时输出 `DEPLOY_MODE=<mode>` 和 resolved 寻址表

### Increment 2: 网关路由生成适配

**NFR 类别: 容错机制**
- **等级**: L1 - 基础
- **量化目标**: `base.yaml` 格式错误时 `routes-to-apisix.py` 立即失败并给出精确的解析错误信息
- **说明**: 不吞错、不静默跳过；配置有问题就阻止启动

**NFR 类别: 可维护性**
- **等级**: L2 - 标准
- **量化目标**: `DEPLOY_MODE=local` 生成的 `apisix.yaml` 与当前逐位一致

### Increment 3: 服务配置迁移

**NFR 类别: 安全性**
- **等级**: L1 - 基础
- **量化目标**: domain 模式下 `publicBase` 为 `https://daydaymoney.com`（HTTPS），OIDC issuer 同理
- **说明**: 不强制 TLS 终止（网关层职责），但配置输出为 HTTPS URL

**NFR 类别: 可维护性**
- **等级**: L2 - 标准
- **量化目标**: 所有迁移字段在 `DEPLOY_MODE=local` 下解析为与当前硬编码值相同的 IP:端口

## 质量场景

### QS-01: base.yaml 缺失时启动失败
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L1 |
| 刺激源 | 部署环境缺少 `conf/base.yaml` 文件 |
| 刺激 | `load_base_yaml()` 调用时文件不存在 |
| 制品 | `conf_lib.load_base_yaml()` |
| 环境 | 服务启动 / 配置加载 |
| 响应 | 抛出 `FileNotFoundError` 并给出清晰错误信息：`conf/base.yaml not found — this file is required for service addressing` |
| 响应度量 | 进程以非零退出码终止；错误信息包含文件路径 |

### QS-02: 向后兼容性验证
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者未设置 DEPLOY_MODE 环境变量 |
| 刺激 | 系统启动，`conf_loader.py` 加载所有服务配置 |
| 制品 | `load_app_config()` → `resolve_template_vars()` |
| 环境 | 正常启动 |
| 响应 | 所有 L1/L2 字段解析值与当前硬编码 IP 一致 |
| 响应度量 | `DEPLOY_MODE=local` 下 `conf-read.py snapshot-json` 输出与当前无差异（除新增 `_addressing` 字段） |

### QS-03: 配置格式错误时精确报错
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L1 |
| 刺激源 | `base.yaml` YAML 格式损坏或缺少必需字段 |
| 刺激 | 配置加载 |
| 制品 | `load_base_yaml()` |
| 环境 | 启动时 |
| 响应 | 抛出具体异常，指明哪个字段缺失/格式错误 |
| 响应度量 | 错误信息包含字段名和期望格式；不输出笼统的"配置错误" |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 可用性 L1 — base.yaml 必需 | `DeployMode` / `AddressingScheme` 值对象无默认构造路径 | DDD 步骤建模时不提供 `default()` 工厂方法，必须从 base.yaml 构造 |
| 可维护性 L2 — 向后兼容 | 模板变量解析为幂等纯函数 | 解析函数标记为 pure function |

## 权衡与边界

### 取舍
- 选择运行时解析而非编译时生成：配置模板变量在进程启动时解析，同一构建产物可在不同环境切换 DEPLOY_MODE
- 选择渐进迁移而非一次性替换：L1 对外字段优先迁移到模板变量，L2 内部地址保持 `127.0.0.1` 不变，L3 绑定地址不变

### 明确不做什么
- 不实现 OAuth provider `redirect_uri` 自动同步（domain 切换时需手动更新远程 GitLab/GitHub OAuth app 配置）
- 不迁移 Go runAll config.go 的 `${INFRA_HOST}` 解析逻辑
- 不实现 domain 模式下服务间调用走域名（始终走 `127.0.0.1`，跨节点通过环境变量覆盖）
- 不做动态热切换 `DEPLOY_MODE`（需重启服务）

### 升级触发条件
- 当需要支持多域名 → base.yaml 扩展为多域名映射
- 当跨节点部署成为常态 → L2 内部寻址升级为服务发现机制

## 跳过声明

- **性能**: 跳过。配置解析在启动时一次性完成，不在请求热路径上
- **可伸缩性**: 跳过。无数据增长或流量扩展关注点
- **数据一致性**: 跳过。无数据存储或跨服务事务
- **合规与隐私**: 跳过。无数据处理或存储变更
