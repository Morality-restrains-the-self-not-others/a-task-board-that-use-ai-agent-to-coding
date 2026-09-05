# Value Stream: Mock authorization_id 云平台查库防护

> Derived from design: `docs/superpowers/specs/2026-05-31-mock-authorization-id-cloud-lookup-guard-design.md`

## Value Summary

Relay/mock 容器任务在读取历史配置、查运行状态、停 VM 时，不再因 `authorization_id='mock-auth'` 触发 500，仍能返回可用的配置快照或明确的业务错误。

## Related Value Streams

- **cloud-integration** (`cloud-compute`): 修改 — compute API 读路径需容忍非数值 authorization_id
- **relay-stop-stale-container-endpoint-invalidation**: 依赖 — mock 会话写入 `mock-auth` 的历史记录是本修复的输入源

## End-to-End Flow

[UI 打开任务详情 / 查上次配置] → [GET previous-server-config] → [读 CloudServerConfigHistory] → [解析 authorization_id] → [数值 PK 才查 CloudPlatformAuthorization] → [200 + server_config]

## Value Increments

### Increment 1: Helper + previous-server-config (Thin Slice)
**Value to user:** 查上次配置不再 500  
**Scope:** `authorization_lookup` helper + refactor `get_previous_server_config` + 单元/服务测试  
**Depends on:** nothing

### Increment 2: stop-vm 防护
**Value to user:** mock 任务停 VM 返回业务错误而非 500  
**Scope:** `stop_vm.py` + 测试  
**Depends on:** Increment 1

### Increment 3: runtime-status + start_vm 复用路径
**Value to user:** 其余 compute 读/写路径一致  
**Scope:** `get_server_runtime_status.py`, `start_vm.py` + 测试  
**Depends on:** Increment 1
