# 每服务独立日志路径配置 — 设计文档

**状态**: 待批准
**日期**: 2026-06-26
**触发**: 后续服务将单独部署，日志路径需要从服务自身配置中读取

## 1. 问题陈述

当前日志文件路径由 `conf/runAll.yaml` 中的 `logging.file_root` 全局统一控制：
```yaml
logging:
  file_root: ../logs
```
所有服务日志写入 `<file_root>/<service_name>.log`（如 `logs/task-auth.log`）。

**问题**：后续服务将独立部署，每个服务需要能独立配置自己的日志落盘路径。一个全局的 `file_root` 无法表达"不同服务日志写在不同位置"。

## 2. 核心约束

- **服务只写 stdout** — 服务自身不打开文件、不写日志文件。日志文件路径是日志收集层的配置
- **配置真实性** — 放在 `conf/<app>/config.yaml` 中的值应该是该服务相关方会读取的（服务自身、runAll、或独立部署的日志采集器）
- **向后兼容** — 未单独配置的服务应自动使用全局默认值

## 3. 三级优先级设计

| 优先级 | 来源 | 读取方 | 说明 |
|--------|------|--------|------|
| 1（最高）| `RUNALL_LOG_ROOT` 环境变量 | runAll, 脚本 | 运维覆盖，全局生效 |
| 2 | `conf/<app>/config.yaml` → `logging.log_file` | runAll (tee), 独立 Promtail | 服务自己的配置 |
| 3（默认）| `conf/runAll.yaml` → `logging.file_root` + `<name>.log` | runAll (tee) | 全局 fallback |

### 3.1 冲突解决

当服务配置了 `logging.log_file` 时：
- **runAll 本地编排**：tee 写入服务配置的路径，忽略全局 `file_root`
- **独立部署**：服务所在主机的 Promtail 从该路径读取日志
- **RUNALL_LOG_ROOT** 环境变量：仍然覆盖一切（运维紧急干预用）

## 4. 配置文件变更

### 4.1 `conf/<app>/config.yaml` — 每服务新增 logging 块

```yaml
# conf/auth/task-auth/config.yaml（示例）
host: "127.0.0.1"
port: 8005
health_path: /api/health/
logging:
  log_file: logs/task-auth/task-auth.log   # 相对于 monorepo 根目录
```

> **约定**：`log_file` 路径相对于 monorepo 根目录。绝对路径也支持（用于 `/var/log/` 等系统路径）。

### 4.2 `conf/runAll.yaml` — 保持全局默认

```yaml
logging:
  # 全局默认日志目录（仅当服务未在 conf/<app>/config.yaml 中配置 logging.log_file 时使用）
  file_root: ../logs   # fallback: <file_root>/<service_name>.log
```

### 4.3 `conf/logs/config.yaml` — 更新为文档/参考

```yaml
# 统一日志管理 — 参考文档
# 
# 日志路径优先级:
#   1. RUNALL_LOG_ROOT 环境变量（全局覆盖）
#   2. conf/<app>/config.yaml → logging.log_file（每服务独立配置）
#   3. conf/runAll.yaml → logging.file_root + <service_name>.log（全局默认）
#
# 本地编排时:
#   runAll 读取每服务配置 → tee 写入对应路径 → Promtail tail 所有路径 → Loki
#
# 独立部署时:
#   服务写 stdout → 本地 runAll/Vector/Promtail → 服务自身的 logging.log_file → Loki
```

## 5. runAll Go 代码变更

### 5.1 `config.go` — conf_app 读取新增 logging 字段

```go
// ConfAppConfig 从 conf/<app>/config.yaml 读取
type ConfAppConfig struct {
    Host    string        `yaml:"host"`
    Port    string        `yaml:"port"`
    Logging ConfAppLogging `yaml:"logging"`
}

type ConfAppLogging struct {
    LogFile string `yaml:"log_file"`  // 相对于项目根目录的日志文件路径
}
```

`resolveConfApp` 方法扩展为也返回 `LogFile`。

### 5.2 `FileServiceLogSink` — 支持每服务路径

```go
type FileServiceLogSink struct {
    mu           sync.Mutex
    rootDir      string                      // 全局默认根目录
    servicePaths map[string]string           // 每服务自定义路径
    files        map[string]*os.File
}

// SetServicePath 为指定服务设置自定义日志文件路径
func (s *FileServiceLogSink) SetServicePath(serviceName, path string) {
    s.servicePaths[serviceName] = path
}

func (s *FileServiceLogSink) fileForServiceLocked(serviceName string) (*os.File, error) {
    // 1. 优先使用服务自定义路径
    if customPath, ok := s.servicePaths[serviceName]; ok {
        return s.openFile(customPath)
    }
    // 2. 回退到全局默认
    path := filepath.Join(s.rootDir, serviceName+".log")
    return s.openFile(path)
}
```

### 5.3 `runner.go` — NewRunner 中注入每服务路径

```go
// 在 NewRunner 中，创建 FileServiceLogSink 后：
for _, svc := range cfg.Flatten() {
    logFile := resolveServiceLogFile(cfg, svc)  // 三级优先级
    if logFile != "" {
        fileSink.SetServicePath(svc.Name, logFile)
    }
}
```

## 6. Promtail 配置变更

### 6.1 本地编排（多服务）

Promtail 需要 tail 多个路径。更新 `promtail-local.yaml` 的 scrape_configs：

```yaml
scrape_configs:
  - job_name: runall-services
    static_configs:
      - targets: [localhost]
        labels:
          job: runall
          __path__: /var/log/runall/**/*.log   # 递归 glob
```

或通过 `file_sd_configs` 动态生成目标列表（从 `conf/<app>/config.yaml` 中读取所有 `logging.log_file`）。

### 6.2 独立部署（每服务一个 Promtail）

```yaml
# 服务专属 promtail 配置
scrape_configs:
  - job_name: task-auth
    static_configs:
      - targets: [localhost]
        labels:
          job: task-auth
          __path__: /var/log/task-auth/task-auth.log
```

## 7. 受影响的现有配置

需要为所有现有服务添加 `logging.log_file`（共 ~35 个服务）：

| 服务 | conf_app | 建议 log_file |
|------|----------|---------------|
| task-auth | auth/task-auth | `logs/task-auth/task-auth.log` |
| task-bill | billing/task-bill | `logs/task-bill/task-bill.log` |
| saas-backend | core/django | `logs/saas-backend/saas-backend.log` |
| ai-provider | ai/ai-provider | `logs/ai-provider/ai-provider.log` |
| task-agent-support | ai/task-agent-support | `logs/task-agent-support/agent.log` |
| task-ai-endpoint | ai/task-ai-endpoint | `logs/task-ai-endpoint/endpoint.log` |
| task-container-gateway | gateway/task-container-gateway | `logs/task-container-gateway/gateway.log` |
| task-gateway | gateway/task-gateway | `logs/task-gateway/task-gateway.log` |
| task-sse | gateway/task-sse | `logs/task-sse/task-sse.log` |
| taskFE | frontend/vue | `logs/taskFE/frontend.log` |
| git-oauth | (无 conf_app) | `logs/git-oauth/git-oauth.log` |
| value-stream | (无 conf_app) | `logs/value-stream/value-stream.log` |
| go-run-container | mock/mock-run-container | `logs/go-run-container/container.log` |
| go-relay | infra/relay-to-trae | `logs/go-relay/relay.log` |
| task-events-* (18个) | (无 conf_app) | `logs/task-events/<intent>.log` |
| docker-redis 等 infra | infra/* | `logs/infra/<name>.log` |

## 8. 实施步骤

1. **更新 `conf/logs/config.yaml`** — 改为参考文档，描述三级优先级
2. **更新每个 `conf/<app>/config.yaml`** — 为有 conf_app 的服务添加 `logging.log_file`
3. **更新 `conf/runAll.yaml`** — `file_root` 改为 fallback 语义
4. **runAll Go 变更**：
   - `config.go`: 扩展 ConfAppConfig 读取 `logging.log_file`
   - `FileServiceLogSink`: 支持 `SetServicePath`
   - `runner.go`: NewRunner 中注入每服务路径
5. **Promtail 配置**: 更新为递归 glob 或 file_sd
6. **更新脚本**: 传递多路径给 Promtail
7. **测试验证**

## 9. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Promtail 需要 tail 30+ 个不同目录 | 用 `**/*.log` 递归 glob 或 `file_sd_configs` 动态生成 |
| 配置文件大量重复 | 用脚本批量生成初始配置；格式统一 |
| 独立部署时路径可能冲突 | 使用服务名作为目录名前缀 |
