# NFR 澄清: 每服务独立日志路径配置

> 输入:
> - 设计文档: `docs/design/per-service-log-config-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-26-per-service-log-config-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可维护性 | L2 | 向后兼容，三级优先级 fallback，配置变更不中断日志流 |
| 可观测性 | L2 | 每服务日志路径可查询，Promtail 正确采集所有路径 |
| 可用性 | L1 | 个别服务配置缺失不影响其他服务日志收集 |
| 可伸缩性 | L1 | 支持 35+ 服务，路径解析 O(1) 查找 |

## 跳过声明

| 类别 | 理由 |
|------|------|
| 性能 | 纯配置变更，不影响日志写入路径的性能（tee 逻辑本身不变） |
| 安全性 | 日志文件权限沿用现有 Unix 644 约定，无新增安全面 |
| 数据一致性 | 日志文件写入为本地文件系统操作，不涉及跨服务一致性 |
| 容错机制 | 现有 tee 容错不变；配置读取失败 fallback 到 L1 可用性 |
| 合规与隐私 | 日志内容策略不变，本次仅变更路径配置方式 |

## 逐增量 NFR 分析

### Increment 1: Go 层支持 + 1-2 服务 PoC

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**:
  - 100% 向后兼容：未配置 `logging.log_file` 的服务自动使用 `file_root/<name>.log`
  - 三级优先级正确解析：`RUNALL_LOG_ROOT` > `conf_app logging.log_file` > `file_root`
  - 新增配置字段不破坏现有 `conf/<app>/config.yaml` 的 YAML 解析

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: runAll 启动日志中可查看每个服务的日志文件路径（"service X → logs/path/file.log"）

#### NFR 类别: 可用性
- **等级**: L1 - 基础
- **量化目标**: 若某个 `conf/<app>/config.yaml` 格式错误，runAll 对其日志路径 fallback 到全局默认值，不影响其他服务

### Increment 2: 全量服务配置覆盖 + Promtail 多路径

#### NFR 类别: 可伸缩性
- **等级**: L1 - 基础
- **量化目标**: 35+ 服务的路径查找在 O(1) 完成（map 查找）；Promtail 递归 glob `**/*.log` 覆盖所有子目录

### Increment 3: 独立部署 Promtail 模板

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 模板文件可直接复制使用，仅需修改 service name 和 log path

## 质量场景

### QS-01: 服务无 logging 配置时 fallback
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 运维人员 |
| 刺激 | 在 conf/core/django/config.yaml 中不写 logging 块 |
| 制品 | runAll tee → FileServiceLogSink |
| 环境 | 正常启动 |
| 响应 | saas-backend 日志写入 `<file_root>/saas-backend.log` |
| 响应度量 | 日志文件在预期路径存在，runAll 启动日志显示 fallback 路径 |

### QS-02: 服务配置了自定义 log_file
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 运维人员 |
| 刺激 | 在 conf/auth/task-auth/config.yaml 中设置 `logging.log_file: logs/task-auth/auth.log` |
| 制品 | runAll tee → FileServiceLogSink |
| 环境 | 正常启动 |
| 响应 | task-auth 日志写入 `logs/task-auth/auth.log`，而非 `logs/task-auth.log` |
| 响应度量 | 日志文件在自定义路径存在；Promtail 能采集到 |

### QS-03: 环境变量覆盖一切
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 运维人员 |
| 刺激 | 设置 `RUNALL_LOG_ROOT=/var/log/emergency` |
| 制品 | runAll tee → FileServiceLogSink |
| 环境 | 紧急运维 |
| 响应 | 所有服务日志写入 `/var/log/emergency/<name>.log`，忽略 conf_app 和 file_root |
| 响应度量 | 日志文件在 `/var/log/emergency/` 下存在 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 三级优先级 (L2 可维护性) | `FileServiceLogSink` 需要路径解析策略 | 新增 `ServiceLogPathResolver` 值对象封装优先级逻辑 |
| 每服务路径 O(1) 查找 (L1 可伸缩性) | 路径存储用 `map[string]string` | 基础设施层使用 map 而非 list |
| 配置缺失不阻塞 (L1 可用性) | 路径解析返回 fallback 而非 error | `resolveServiceLogFile()` 永不为单个服务返回 error |

## 权衡与边界

### 取舍
- 选择在 `conf/<app>/config.yaml` 中新增 `logging` 块（而非独立日志配置文件），保持每服务单一配置入口
- Promtail 递归 glob `**/*.log` 替代逐路径配置，牺牲精确控制换取配置简洁

### 明确不做什么
- 不实现日志文件的动态重载（修改配置后需重启 runAll）
- 不支持 per-service 日志格式自定义（保持统一 JSON 格式）
- 不在 V1 实现日志文件轮转（由 Loki 保留策略管理）

### 升级触发条件
- 当单服务日志量超过 1GB/天 → 考虑 per-service 日志轮转策略
- 当服务数量超过 100 → 考虑 Promtail file_sd 动态发现替代静态 glob
- 当需要 per-tenant 日志隔离 → 引入日志路径模板变量（如 `{tenant_id}`）
