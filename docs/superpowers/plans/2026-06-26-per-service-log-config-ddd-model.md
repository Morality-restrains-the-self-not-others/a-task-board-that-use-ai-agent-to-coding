# 领域模型: 每服务独立日志路径配置

> 输入:
> - 价值流: `docs/superpowers/plans/2026-06-26-per-service-log-config-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-26-per-service-log-config-nfr-clarification.md`
>
> 类型: 基础设施增强 — 轻量领域模型，无新业务实体

## 跳过声明

本次变更不涉及业务实体、聚合、仓储或领域事件。核心变更是为 `FileServiceLogSink` 引入
**每服务路径解析策略**。因此采用轻量领域模型，仅产出值对象和接口。

## 限界上下文

| 上下文 | 职责 | 现有/新增 |
|--------|------|----------|
| runAll 日志收集 | 服务 stdout tee → 文件落盘 → Promtail 采集 | 增强 |
| 服务配置管理 (conf_app) | conf/<app>/config.yaml 读取与解析 | 增强 |

## 新增领域概念

### 值对象: `LogFilePath`

```
LogFilePath
├── value: string (相对或绝对路径)
├── Validate(): 非空、不含非法字符
└── IsAbsolute(): bool
```

**职责**: 封装日志文件路径，确保路径合法性。

### 值对象: `ServiceLogPathResolution`

```
ServiceLogPathResolution
├── ServiceName: string
├── EnvOverride: string | nil    (RUNALL_LOG_ROOT)
├── ConfAppLogFile: string | nil (conf/<app>/config.yaml → logging.log_file)
├── GlobalFileRoot: string       (conf/runAll.yaml → logging.file_root)
│
└── Resolve(): LogFilePath
      1. EnvOverride != nil → <EnvOverride>/<ServiceName>.log
      2. ConfAppLogFile != nil → ConfAppLogFile
      3. Fallback → <GlobalFileRoot>/<ServiceName>.log
```

**职责**: 封装三级优先级路径解析逻辑。NFR 要求 L2 可维护性（100% 向后兼容）。

### 接口变更: `ServiceLogFileSink`

```
ServiceLogFileSink (现有接口)
├── 新增: SetServicePath(name string, path string)
├── 变更: fileForServiceLocked(name)
│     1. 检查 servicePaths map
│     2. Fallback 到 rootDir + name + ".log"
└── 不变: AppendLine, TruncateService, Close
```

**职责**: 日志文件落盘，现在支持每服务自定义路径。

## 文件结构

```
runAll/src/domain/
├── log_file_path_value_object.go           ← 新增
├── service_log_path_resolution.go          ← 新增
└── service_log_file_sink.go                ← 接口扩展

runAll/src/infrastructure/
└── file_service_log_sink.go                ← 实现变更
```

## 领域模型影响（来自 NFR）

| NFR 决策 | 模型影响 | 实现动作 |
|----------|---------|---------|
| L2 三级优先级 | `ServiceLogPathResolution` 封装优先级逻辑 | 值对象不可变，Resolve() 纯函数 |
| L1 O(1) 查找 | servicePaths 用 `map[string]string` | 基础设施层直接使用 Go map |
| L1 配置缺失不阻塞 | Resolve() 永不为单服务返回 error | Fallback 到 file_root 而非 error |

## 自检

- [x] 无 ORM 导入 — Go 纯类型，无持久化依赖
- [x] 无外部服务导入 — 路径解析为纯计算
- [x] 接口即契约 — `ServiceLogFileSink` 定义 AppendLine 签名
- [x] 值对象不可变 — `ServiceLogPathResolution` 构造后不可变
- [x] 文件名 = 类名 (snake_case)
