# Value Stream: runAll 显式生命周期命令

> Derived from design: `docs/superpowers/specs/2026-05-31-runall-explicit-lifecycle-commands-design.md`

## Value Summary

开发者在 runAll UI 上点击启停/重启/编译时，行为与 YAML 中显式脚本命令一致，且可在服务目录内手工复现；关闭本组在单项失败时仍尽力停掉其余服务。

## Related Value Streams

- **runall-cascade-lifecycle**：**扩展** — 单服务 stop/start 改为 `stop_command`/`start_command`；组级 stop 改为 best-effort。
- **runall-docker-infra-split**：**修改** — stop 不再依赖 runner 推断，须 YAML 显式 `stop_command`。
- **docker-infra-no-repull**：**对齐** — `run.sh stop` 由配置引用，非硬编码。

## End-to-End Flow

[开发者打开 :9999] → [点击启动/关闭/重启/编译] → [runner 执行对应 YAML 命令] → [健康检查/状态更新] → [Docker/进程真实启停]

组关闭：[关闭本组] → [逆拓扑逐服务 stop_command] → [失败仍继续] → [聚合错误可选返回]

## Value Increments

### Increment 1: 配置 + strict + infrastructure（Thin Slice）
**Value to user:** docker-redis/kafka/ai-monitor UI 关闭真实停容器/栈  
**Scope:** `start_command`/`stop_command`/`launch_mode`；strict 校验；runner stop 主路径；infrastructure YAML 齐全  
**Depends on:** nothing

### Increment 2: stopGroup best-effort + restart 先停后启
**Value to user:** 组关闭更可靠；重启语义可预期  
**Scope:** `stopGroup` 聚合错误；`restartService` 编排  
**Depends on:** Increment 1

### Increment 3: platform + domain-events 全量 YAML 与 stop 脚本
**Value to user:** 所有服务满足 strict，本地 runAll 可启动  
**Scope:** 各 `run.sh` / lifecycle 脚本；`runAll.yaml` 全服务三条命令  
**Depends on:** Increment 1–2

### Increment 4: 删除推断与 `command` 字段
**Value to user:** 无隐式魔法，文档即契约  
**Scope:** 移除 `composeStopShellCommand` 主路径、`resolveBuildCommand` 推断  
**Depends on:** Increment 3
