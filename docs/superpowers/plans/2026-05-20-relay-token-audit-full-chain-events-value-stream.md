# Value Stream: relayToTrae Token 审计全链路事件化

> Derived from design: `docs/superpowers/specs/2026-05-20-relay-token-audit-events-design.md`

## Value Summary
当 relay 链路出现 401 或代理异常时，研发可在分钟级定位“尝试、成功、失败”发生点，快速确认是否旧 token 或下游响应异常导致故障。

## End-to-End Flow
用户触发 `刷新状态/启动` -> Django 代理发起 relay 调用并记录 `attempted` -> relay 返回 -> Django 记录 `succeeded` 或 `failed` -> 状态上报接口继续记录 `status_push_ok/401` -> 研发查询时间线并定位问题根因。

## Value Stages
- 触发阶段：用户在任务详情页点击 `刷新状态` 或 `启动`。
- 代理编排阶段：Django 计算 access token、构造 payload。
- 审计写入阶段：按 `attempted -> succeeded/failed` 追加事件。
- 状态上报阶段：`status-push` 校验 token 并记录成功/401 事件。
- 交付阶段：时间线查询返回完整链路，支持排障。

## Wait / Dependency Points
- 依赖 `CloudServerConfig` 的当前 token 与过期时间判断。
- 依赖 relay HTTP 返回（异常、4xx/5xx、2xx 非法响应体）决定失败分类。
- 依赖审计表可写（写失败不阻断主链路，但会影响观测完整度）。

## Value Increments

### Increment 1: Register 全链路最薄切片（Thin Slice）
**Value to user:** 可看到 `relay_register_attempted` 与最终 `relay_register_succeeded/failed`，并区分异常类别。  
**Scope:** `relay_to_trae_register` 增加三态事件；覆盖异常、HTTP>=400、2xx 非法响应体。  
**Depends on:** 现有审计表与写入应用服务（无新增 schema）。

### Increment 2: Start 全链路补齐
**Value to user:** `启动` 链路具备与 `register` 一致的全链路审计语义。  
**Scope:** `relay_to_trae_start` 增加 `attempted/succeeded/failed`，并落 `error_code/error_detail`。  
**Depends on:** Increment 1（事件语义与测试模式复用）。

### Increment 3: 时间线消费语义对齐（核心可用性）
**Value to user:** 查询时间线时可以直接按阶段理解链路，不再把“成功事件”误读为“调用已成功”。  
**Scope:** 时间线 API/调用方按三态事件展示；兼容旧 `relay_register/relay_start` 事件（视为 succeeded）。  
**Depends on:** Increment 1-2。

### Increment 4: 失败分类增强与运维可观测（Enhancement）
**Value to user:** 失败事件可快速分流（网络异常/下游错误/响应体异常），定位路径更短。  
**Scope:** 统一 `error_code` 枚举与截断策略，补充回归与清理策略验证。  
**Depends on:** Increment 1-3。

