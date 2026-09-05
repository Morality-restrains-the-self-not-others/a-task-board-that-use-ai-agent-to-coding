# 实施计划: 每服务独立日志路径配置

> 输入:
> - 设计文档: `docs/design/per-service-log-config-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-26-per-service-log-config-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-26-per-service-log-config-nfr-clarification.md`
> - 领域模型: `docs/superpowers/plans/2026-06-26-per-service-log-config-ddd-model.md`

## 增量 1: Go 层支持 + PoC 服务配置（Thin Slice）

### Phase 1: 领域层

- [ ] **T1.1** 创建 `runAll/src/domain/service_log_path_resolution.go`
  - `ServiceLogPathResolution` 值对象：封装三级优先级路径解析
  - `Resolve()` 方法：`RUNALL_LOG_ROOT` → `conf_app logging.log_file` → `file_root/<name>.log`
  - 纯函数，无外部依赖
  - 测试: `runAll/src/domain/service_log_path_resolution_test.go`

- [ ] **T1.2** 创建 `runAll/src/domain/log_file_path_value_object.go`
  - `LogFilePath` 值对象：验证非空路径
  - `IsAbsolute()` 判断方法
  - 测试: `runAll/src/domain/log_file_path_value_object_test.go`

### Phase 2: 基础设施层

- [ ] **T1.3** 扩展 `runAll/src/infrastructure/file_service_log_sink.go`
  - 新增 `servicePaths map[string]string` 字段
  - 新增 `SetServicePath(name, path string)` 方法
  - 修改 `fileForServiceLocked()`：先查 `servicePaths`，再 fallback 到 `rootDir + name + ".log"`
  - 在 `NewFileServiceLogSink` 中初始化 map
  - 测试: 更新 `runAll/src/infrastructure/file_service_log_sink_test.go`

- [ ] **T1.4** 扩展 `runAll/src/config.go` — conf_app 读取 logging 块
  - `ConfAppConfig` 结构体新增 `Logging ConfAppLogging` 字段
  - `ConfAppLogging` 结构体：`LogFile string \`yaml:"log_file"\``
  - 新增 `resolveServiceLogFile(cfg *Config, svc Service) string` 函数
  - 测试: 更新 `runAll/src/config_test.go`

### Phase 3: Runner 集成

- [ ] **T1.5** 修改 `runAll/src/runner.go` — NewRunner 注入每服务路径
  - 在 `NewRunner` 创建 FileServiceLogSink 后，遍历所有服务
  - 调用 `resolveServiceLogFile()` 获取每服务路径
  - 调用 `fileSink.SetServicePath()` 注册自定义路径
  - 保留向后兼容：未配置的服务走 fallback

### Phase 4: 配置文件 — PoC 服务

- [ ] **T1.6** 更新 `conf/auth/task-auth/config.yaml`
  - 新增 `logging.log_file: logs/task-auth/task-auth.log`

- [ ] **T1.7** 更新 `conf/core/django/config.yaml`
  - 新增 `logging.log_file: logs/saas-backend/saas-backend.log`

- [ ] **T1.8** 更新 `conf/logs/config.yaml`
  - 改为文档/参考格式
  - 描述三级优先级
  - 列出所有服务的建议 log_file 路径

- [ ] **T1.9** 更新 `conf/runAll.yaml`
  - `file_root: ../logs` 注释说明 fallback 语义
  - 添加 "仅当服务未单独配置时使用" 注释

### Phase 5: 构建与验证

- [ ] **T1.10** 构建与测试
  ```bash
  cd runAll && go build -o bin/runAll ./src/
  cd runAll/src && go test ./... -count=1
  ```
  - 验证: 全部测试通过，二进制构建成功

---

## 增量 2: 全量服务配置覆盖 + Promtail 多路径

- [ ] **T2.1** 为所有有 conf_app 的服务添加 `logging.log_file`
  - `conf/billing/task-bill/config.yaml` → `logs/task-bill/task-bill.log`
  - `conf/ai/ai-provider/config.yaml` → `logs/ai-provider/ai-provider.log`
  - `conf/ai/task-agent-support/config.yaml` → `logs/task-agent-support/agent.log`
  - `conf/ai/task-ai-endpoint/config.yaml` → `logs/task-ai-endpoint/endpoint.log`
  - `conf/gateway/task-container-gateway/config.yaml` → `logs/task-container-gateway/gateway.log`
  - `conf/gateway/task-gateway/config.yaml` → `logs/task-gateway/task-gateway.log`
  - `conf/gateway/task-sse/config.yaml` → `logs/task-sse/task-sse.log`
  - `conf/frontend/vue/config.yaml` → `logs/taskFE/frontend.log`
  - `conf/mock/mock-run-container/config.yaml` → `logs/go-run-container/container.log`
  - `conf/infra/relay-to-trae/config.yaml` → `logs/go-relay/relay.log`
  - `conf/infra/redis/config.yaml` → `logs/infra/docker-redis.log`
  - `conf/infra/kafka/config.yaml` → `logs/infra/docker-kafka.log`
  - `conf/infra/ai-monitor/config.yaml` → `logs/infra/ai-monitor.log`

- [ ] **T2.2** 更新 Promtail 配置 — 递归 glob
  - `AiMonitor/promtail/promtail-local.yaml`：scrape_configs 改为 `__path__: /var/log/runall/**/*.log`

- [ ] **T2.3** 更新脚本
  - `runAll/scripts/runall-local-promtail.sh`：已验证支持相对路径解析

- [ ] **T2.4** 构建与回归测试
  ```bash
  cd runAll && go build -o bin/runAll ./src/
  cd runAll/src && go test ./... -count=1 -timeout 120s
  ```

---

## 增量 3: 独立部署 Promtail 模板 + 文档

- [ ] **T3.1** 创建 `conf/logs/promtail-service-template.yaml`
  - 每服务 Promtail 配置模板
  - `__path__` 指向服务自身的 `logging.log_file`

- [ ] **T3.2** 创建 `docs/runbooks/per-service-log-deploy.md`
  - 独立部署指南
  - Promtail 配置说明

- [ ] **T3.3** runAll 启动时自动创建日志父目录
  - 在 `SetServicePath` 中调用 `os.MkdirAll(filepath.Dir(path), 0755)`

---

## 验证命令

```bash
# 构建
cd runAll && go build -o bin/runAll ./src/

# 测试
cd runAll/src && go test ./... -count=1 -timeout 120s

# 验证配置
python3 -c "import yaml; yaml.safe_load(open('conf/runAll.yaml'))"
python3 -c "import yaml; yaml.safe_load(open('conf/logs/config.yaml'))"

# 验证 YAML
python3 -c "import yaml; yaml.safe_load(open('conf/value-stream.yaml'))"
```
