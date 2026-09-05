# 实施计划: 移除远程同步，统一 INFRA_HOST

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-05-remove-remote-sync-unify-infra-host-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-05-remove-remote-sync-unify-infra-host-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-05-remove-remote-sync-unify-infra-host-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-05-remove-remote-sync-unify-infra-host-ddd.md`
>
> 执行模式: subagent-driven-development（6 增量，每增量独立子代理）

## Increment 1: 配置拆分 — 基础设施各服务独立 conf_app

- [ ] 1.1 创建 `conf/infra/redis/config.yaml` — host: 127.0.0.1, port: 6379
- [ ] 1.2 创建 `conf/infra/kafka/config.yaml` — host: 127.0.0.1, kafkaUiPort: 18080
- [ ] 1.3 创建 `conf/infra/portainer/config.yaml` — host: 127.0.0.1, port: 9000
- [ ] 1.4 创建 `conf/infra/ai-monitor/config.yaml` — host: 127.0.0.1, grafana: 3000
- [ ] 1.5 `conf/runAll.yaml` — 基础设施 5 个服务添加 `conf_app` 字段（如 `conf_app: "infra/redis"`）
- [ ] 1.6 `conf/runAll.yaml` — 基础设施服务的 start_command 改为本地 run.sh（如 `bash dockerInfra/redis/run.sh start`）
- [ ] 1.7 `conf/runAll.yaml` — 基础设施服务的 stop_command 改为本地 run.sh（如 `bash dockerInfra/redis/run.sh stop`）
- [ ] 1.8 `conf/runAll.yaml` — 删除 `remote_docker` 块
- [ ] 1.9 删除 `conf/infra/docker-infra/config.yaml`
- [ ] 1.10 `runAll/src/config.go` — 从 Config 结构体移除 `RemoteDocker RemoteDocker` 字段
- [ ] 1.11 `make test` — 验证 config.go 编译通过

## Increment 2: 统一模板变量 — ${HOST} → ${INFRA_HOST}

- [ ] 2.1 `conf/runAll.yaml` — 所有 `${HOST}` 替换为 `${INFRA_HOST}`（git-oauth, 16 个 task-events-*, value-stream, go-relay）
- [ ] 2.2 `runAll/src/config.go` — `resolveConfApps` 移除 `${HOST}` 替换代码块
- [ ] 2.3 `runAll/src/config.go` — `resolveConfApps` 新增统一 `${INFRA_HOST}` 替换逻辑：
  - 有 conf_app → 自身 host
  - 无 conf_app → 首个 conf_app host（兜底）
  - Observability 顶层 URL → ai-monitor host 或首个 conf_app host
- [ ] 2.4 `runAll/src/config.go` — `LoadConfig` 移除对 `resolveInfraHostTemplates()` 的调用
- [ ] 2.5 `runAll/src/config.go` — `fillDefaults` 移除对 `resolveInfraHostTemplates()` 的调用
- [ ] 2.6 `runAll/src/config_test.go` — 更新测试：移除 `TestResolveConfApps_InfraHostTemplateResolution`
- [ ] 2.7 `runAll/src/config_test.go` — 新增测试：`TestResolveConfApps_UnifiedInfraHost` — 验证 ${INFRA_HOST} 从各 conf_app 解析
- [ ] 2.8 `runAll/src/config_test.go` — 新增测试：`TestResolveConfApps_InfraHostFallback` — 验证无 conf_app 服务的兜底
- [ ] 2.9 `runAll/src/config_test.go` — 新增测试：`TestResolveConfApps_NoHostVariable` — grep `${HOST}` 返回空
- [ ] 2.10 `make test` — config 测试全绿

## Increment 3: 删除远程同步 — 脚本 + Go 代码

- [ ] 3.1 删除 `scripts/runall-remote-docker.sh`
- [ ] 3.2 删除 `runAll/src/remote_docker.go`
- [ ] 3.3 删除 `runAll/src/remote_docker_test.go`
- [ ] 3.4 `runAll/src/runner.go` — 移除 `RemoteSyncEnabled()` 条件分支调用
- [ ] 3.5 `runAll/src/conf_sync.go` — 移除 `SyncConfReplicaToRemote()` 函数和 `confReplicaRemoteSyncLabel` 常量
- [ ] 3.6 `runAll/src/lifecycle_exec.go` — 从 `buildServiceEnv()` 移除 `RUNALL_INFRA_HOST` 环境变量设置
- [ ] 3.7 编译验证：`cd runAll && go build ./...`
- [ ] 3.8 `make test` — 全部 Go 测试编译 + 通过

## Increment 4: UI + API 清理

- [ ] 4.1 `runAll/src/ui.go` — 移除 `/api/remote-docker/sync` 端点和 `handleRemoteDockerSyncAction`
- [ ] 4.2 `runAll/src/ui.go` — 移除 `/api/conf/sync-remote` 端点和 `handleConfReplicaRemoteSyncAction`
- [ ] 4.3 `runAll/src/ui.go` — 从 observability API 移除 `infra_host` 和 `portainer_url` 字段
- [ ] 4.4 `runAll/src/ui.go` — 从 status payload 移除 `ManagementURL` 字段和 `syncable` 字段
- [ ] 4.5 `runAll/src/ui.go` — 移除 `ServiceStatusPayload.ManagementURL` 结构体字段
- [ ] 4.6 `runAll/src/status.html` — 移除 Portainer 链接渲染逻辑（`portLink` + Portainer bar）
- [ ] 4.7 删除 `runAll/src/domain/infra_management_panel_url_value_object.go`
- [ ] 4.8 删除 `runAll/src/domain/infra_management_panel_url_value_object_test.go`
- [ ] 4.9 删除 `runAll/src/domain/tcp_probe_endpoint_value_object.go`
- [ ] 4.10 删除 `runAll/src/domain/tcp_probe_endpoint_value_object_test.go`
- [ ] 4.11 `runAll/src/ui_test.go` — 移除 `TestAPIObservability_IncludesPortainerURL`
- [ ] 4.12 `runAll/src/ui_test.go` — 移除 `TestAPIRemoteDockerSyncService_*`
- [ ] 4.13 `runAll/src/ui_test.go` — 新增测试验证 `/api/remote-docker/sync` 返回 404
- [ ] 4.14 `make test` — UI 测试全绿

## Increment 5: 辅助脚本 + 监控 + 测试清理

- [ ] 5.1 `scripts/docker-env.sh` — 移除 runAll 远程编排提示
- [ ] 5.2 `scripts/docker-tunnel-remote.sh` — 移除 runAll 健康检查注释
- [ ] 5.3 `scripts/docker-status.sh` — 移除 runAll SSH 隧道注释
- [ ] 5.4 `scripts/docker-setup.sh` — 移除 runAll 远程构建说明
- [ ] 5.5 `scripts/docker-desktop-helper.sh` — 移除 runAll 远程堆栈引用
- [ ] 5.6 `AiMonitor/prometheus/file_sd/runall-health-targets.json` — 更新目标 IP 为 127.0.0.1
- [ ] 5.7 `AiMonitor/scripts/generate_prometheus_from_runall.py` — 移除远程 IP 生成逻辑
- [ ] 5.8 `runAll/playwright/tests/` — 审查并更新 runall-* E2E 测试中的远程堆栈选择器
- [ ] 5.9 `valueStream/` — 运行 `go test ./...` 验证 value-stream.yaml 解析
- [ ] 5.10 `grep -r "runall-remote-docker\|RemoteDocker\|remote_docker.host\|SyncRemoteDocker\|SyncServiceToRemote\|PortainerURL"` 确认仅在文档中匹配

## Increment 6: 端到端验证

- [ ] 6.1 `cd runAll && go test ./... -count=1` — 全部 Go 测试通过
- [ ] 6.2 `cd valueStream && go test ./...` — 价值流测试通过
- [ ] 6.3 手动验证：Docker Desktop 已运行，`dockerInfra/redis/run.sh start` 成功
- [ ] 6.4 手动验证：`dockerInfra/redis/run.sh stop` 成功清理
- [ ] 6.5 验证 `conf/runAll.yaml` YAML 语法：`python3 -c "import yaml; yaml.safe_load(open('conf/runAll.yaml'))"`
- [ ] 6.6 验证无 `${HOST}` 残留：`grep '\${HOST}' conf/runAll.yaml` 返回空
- [ ] 6.7 验证无 `remote_docker` 残留：`grep 'remote_docker' conf/runAll.yaml` 返回空
- [ ] 6.8 运行 runAll Web UI：`cd runAll && ./bin/runAll --config ../conf/runAll.yaml` — 页面加载无错误

## 执行顺序

```
Incr 1 ──→ Incr 2 ──→ Incr 3 ──→ Incr 4 ──→ Incr 5 ──→ Incr 6
(配置)    (变量)    (删除)    (UI)     (周边)    (验证)
```

每个增量内任务基本可并行，但建议顺序执行以保证每步编译通过。

## 关键验证命令

```bash
# Go 编译
cd runAll && go build ./...

# Go 测试
cd runAll && go test ./... -count=1

# 价值流
cd valueStream && go test ./...

# 死代码检查
grep -r "runall-remote-docker\|RemoteDocker\|remote_docker\.host\|SyncRemoteDocker\|SyncConfReplicaToRemote\|SyncServiceToRemote\|PortainerURL\|BuildInfraManagementPanelURL\|ResolveServiceManagementURL" \
  --include="*.go" --include="*.sh" --include="*.yaml" --include="*.html" \
  runAll/src/ scripts/ conf/ AiMonitor/ | grep -v "_test.go" | grep -v "docs/"

# 变量残留检查
grep '\${HOST}' conf/runAll.yaml          # 应返回空
grep 'remote_docker' conf/runAll.yaml     # 应返回空
```
