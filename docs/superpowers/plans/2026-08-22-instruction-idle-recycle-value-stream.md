# Value Stream: 指令完成后闲置回收

> Derived from design: `docs/superpowers/specs/2026-08-22-instruction-idle-recycle-design.md`

## Value Summary

租户按工作空间已有「闲置自动回收」分钟数，在评论容器指令交付成功后自动释放机器，避免容器仍注册时空转计费。

## Related Value Streams

- **workspace-machine-idle-policy**: extension — 原「卸载后 idle_since」保留；本流增加「指令已交付」闲置
- **task-detail-repo-clone-credentials-contract** / **bootstrap-task-detail-via-gateway**: extension — task-detail 增只读 `idle_recycle_minutes`
- **terminal-release / request-machine-release**: 复用释放主权路径

## End-to-End Flow

设置页已有 N 分钟 → 容器 bootstrap 拉 task-detail 得到 N → 执行指令 → 交付成功 → 标记 instruction_idle + 倒计时 → 无新指令到期 → L1 release（失败则 L2 timer；可选 L3 STS）→ 机器释放

## Value Increments

### Increment 1: task-detail 带闲置分钟（Thin Slice）

**Value to user:** 容器能读到与设置页相同的 N。
**Scope:** Cloud internal GET policy；Credential 组装字段。
**Business intents → events:** 纯查询，无事件。
**Depends on:** nothing

### Increment 2: 交付成功进入闲置 + 新指令取消

**Value to user:** 平台知道「本评论已闲置」；新指令打断旧 job。
**Scope:** CSC `instruction_idle_since`；heartbeat 标志；OSJS interrupt + timer 取消；事件 Marked/Cleared。
**Depends on:** Increment 1

### Increment 3: 到期释放（L1+L2）

**Value to user:** N 分钟无新指令则拆机（容器可达走 L1；容器已死走 L2）。
**Scope:** OSJS `request-machine-release`；recycle 扫描 instruction_idle（server_url 可非空）。
**Events:** `CLOUD_SERVER_STOPPED` reason=`instruction_idle`
**Depends on:** Increment 2

### Increment 4: 交付门闩 + 可选 L3 STS

**Value to user:** 提交失败不拆机；平台宕机且已交付时可选用锁实例 STS。
**Scope:** finalizeJob 失败不闲置；CPA `sts_release_role_arn` 空则无 STS。
**Depends on:** Increment 3

## YAML 配置

见 `conf/value-stream.yaml` stream `instruction-idle-recycle`。
