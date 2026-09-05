# Value Stream: 每服务独立日志路径配置

> Derived from design: `docs/design/per-service-log-config-design.md`

## Value Summary

运维/SRE 可以为每个服务独立配置日志文件路径，支持后续独立部署；开发者在本地编排时，runAll 自动读取每服务配置并 tee 到对应路径。

## Related Value Streams

Greenfield — no existing value streams for logging configuration.

## End-to-End Flow

[运维配置 conf/<app>/config.yaml → logging.log_file] → [runAll 启动时读取 conf_app] → [tee 写入每服务路径] → [Promtail tail 多路径] → [Loki 聚合] → [用户可在 Grafana 查看每服务日志]

## Value Increments

### Increment 1: Go 层支持 + 1-2 服务 PoC（Thin Slice）
**Value to user:** 开发/运维可以为 task-auth 和 saas-backend 独立配置日志路径
**Scope:**
- `config.go`: ConfAppConfig 新增 `logging.log_file` 读取 → `resolveServiceLogFile()`
- `FileServiceLogSink`: 新增 `SetServicePath()` + 每服务路径 map
- `runner.go`: NewRunner 中注入服务路径
- `conf/auth/task-auth/config.yaml`: 新增 `logging.log_file`
- `conf/core/django/config.yaml`: 新增 `logging.log_file`
- `conf/logs/config.yaml`: 更新为文档/参考
- `conf/runAll.yaml`: `file_root` 注释说明 fallback 语义
- 已有测试保持通过
**Depends on:** nothing

### Increment 2: 全量服务配置覆盖 + Promtail 多路径
**Value to user:** 所有 30+ 服务都有独立的日志路径配置，Promtail 正确采集
**Scope:**
- 为所有剩余有 conf_app 的服务添加 `logging.log_file`
- 无 conf_app 的服务：在 `conf/runAll.yaml` 的 service 定义中内联 logging 配置（或保持 fallback）
- Promtail 配置：更新 scrape_configs 为递归 glob `**/*.log`
- `runall-local-promtail.sh`: 支持多路径传参
- `AiMonitor/run.sh`: 同步更新
**Depends on:** Increment 1

### Increment 3: 独立部署 Promtail 模板 + 文档
**Value to user:** 新服务独立部署时可直接复用 Promtail 模板
**Scope:**
- 模板文件：`conf/logs/promtail-service-template.yaml`
- 文档：`docs/runbooks/per-service-log-deploy.md`
- runAll 配置验证：启动时检查引用的 log_file 父目录是否存在，自动创建
**Depends on:** Increment 2
