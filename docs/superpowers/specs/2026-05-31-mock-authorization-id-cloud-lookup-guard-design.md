# Mock authorization_id 云平台查库防护 — 设计文档

**日期**: 2026-05-31  
**状态**: 待批准  
**触发**: `get_previous_server_config` 在 `authorization_id='mock-auth'` 时 500（`Field 'id' expected a number but got 'mock-auth'`）

---

## 1. 问题陈述

`CloudServerConfig` / `CloudServerConfigHistory.authorization_id` 是 **CharField**，用于存储：

| 值类型 | 示例 | 来源 |
|--------|------|------|
| 真实云授权 PK（字符串形式） | `"1847293847293847"` | Kafka 启动 VM、用户选授权 |
| Mock/Relay 占位符 | `"mock-auth"` | `runtime_session_lifecycle`、`mock_run_container` |
| 系统探针标识 | `"sysadmin-container-probe"` | system admin 容器探针 |

多处服务在查 `CloudPlatformAuthorization` 时，直接把该字符串当作 ORM `id=` 过滤条件。Django 对 Integer/Snowflake PK 做 `int(value)` 预处理，非数值字符串会 **抛 ValueError**，被外层 `except` 捕获后变成 500。

已复现路径：`GET .../previous-server-config?task_id=...`，历史记录来自 mock/relay 会话。

**局部修复已落地**（未提交）：`get_previous_server_config.py` 在 `isdigit()` 时才查库；`tests/test_get_previous_server_config.py` 已通过。

---

## 2. 目标与非目标

### 目标

- Mock/relay/探针等非数值 `authorization_id` 不再导致 500。
- 真实数值 PK 行为不变（仍能查授权、查价、停机等）。
- 抽取单一解析/查库入口，避免同类 bug 重复出现。
- 每个受影响 API 至少一条回归测试。

### 非目标

- 不改 `authorization_id` 字段类型（仍为 CharField）。
- 不改 mock 写入 `"mock-auth"` 的现有约定。
- 不重构 `resolve_cloud_auth_for_tenant` 的全局授权路由（OAuth vs AccessKey）。
- 不统一 mock 检测策略（如 `instance_id.startswith('mock-')`）——仅聚焦 **查 CloudPlatformAuthorization 前的 PK 校验**。

---

## 3. 领域概念清单（供 DDD 输入）

| 概念 | 边界上下文 | 说明 |
|------|-----------|------|
| **AuthorizationReference** | 云平台与资源 | CharField 上的授权引用；可能是 PK 或语义占位符 |
| **CloudPlatformAuthorization** | 云平台与资源 | 租户云账号授权聚合根（Snowflake PK） |
| **RuntimeSession** | 容器/Relay | mock/relay 本地运行会话，使用占位 authorization |
| **ServerConfigHistory** | 云平台与资源 | 任务级配置历史，含 authorization_id 快照 |

**领域事件**：无新增事件；属读路径/停机等操作的防御性修复。

---

## 4. 价值流影响分析

### 受影响现有流

| 流 | 步骤 | 影响 |
|----|------|------|
| `cloud-integration` | `cloud-compute` | `previous-server-config`、停机等 compute API |
| `container-relay`（relay 相关步骤） | `relay-stop-*`、mock 容器启动 | 写入 `mock-auth` 的历史被读取时不应 500 |

### 字段

- `saas-backend.cloud_cloudserverconfig.authorization_id` — 读路径，无 schema 变更
- `saas-backend.cloud_cloudserverconfighistory.authorization_id` — 读路径，无 schema 变更
- `saas-backend.cloud_cloudplatformauthorization.id` — 仅当 authorization_id 为数值 PK 时查询

### 测试

| 文件 | 动作 |
|------|------|
| `tests/test_get_previous_server_config.py` | 新增（已有） |
| `tests/test_stop_vm*.py` 或新建 | mock authorization 停 VM 不 500 |
| `tests/test_server_runtime_status*.py` 或扩展 | mock authorization 查状态不 500 |
| `cloud/tests/test_cloud_compute.py` | 可选：集成 previous-server-config |

### 新流

不需要新 value stream；作为 `cloud-integration` / relay 容器的 **增量加固**。

---

## 5. 方案对比

### 方案 A（推荐）：共享 helper + 调用点替换

新增 `cloud/services/authorization_lookup.py`：

```python
def parse_cloud_platform_authorization_pk(authorization_id: str | None) -> int | None:
    raw = str(authorization_id or "").strip()
    return int(raw) if raw.isdigit() else None

def get_cloud_platform_authorization(
    authorization_id, *, company_id, platform_type=None
) -> CloudPlatformAuthorization | None:
    pk = parse_cloud_platform_authorization_pk(authorization_id)
    if pk is None:
        return None
    qs = CloudPlatformAuthorization.objects.filter(id=pk, company_id=company_id)
    if platform_type:
        qs = qs.filter(platform_type=platform_type)
    return qs.first()
```

**优点**：最小 diff、行为集中、易测。  
**缺点**：需逐个替换调用点。

### 方案 B：DB 层约束 mock 值不写 history

在 `runtime_session_lifecycle` 写空字符串而非 `"mock-auth"`。

**缺点**：破坏 MinLengthValidator(4)；且 CharField 语义仍是「引用」，无法从根上消除误用 PK 的风险。

### 方案 C：authorization_id 改 FK

Migration 将 CharField 改为 Nullable FK。

**缺点**：mock/探针占位符无法建模；迁移成本高；超出本次范围。

**推荐：方案 A**

---

## 6. 详细设计

### 6.1 共享 helper

- 路径：`task2app/Saas_project/cloud/services/authorization_lookup.py`
- 导出：`parse_cloud_platform_authorization_pk`、`get_cloud_platform_authorization`
- 不在 helper 内查 OAuthToken（与现有 `resolve_cloud_auth_for_tenant` 职责分离；本次 mock 场景无需 OAuth）

### 6.2 替换调用点（Phase 1 必须）

| 文件 | 当前行为 | 修复后 |
|------|---------|--------|
| `get_previous_server_config.py` | 已 inline isdigit | 改用 helper（去重） |
| `stop_vm.py` | `filter(id=authorization_id)` 可能 ValueError | helper；无授权时保持现有「未配置云平台授权」分支 |
| `get_server_runtime_status.py` | mock instance 有 early return，但非 mock instance + mock-auth 仍可能炸 | 在查 auth 前用 helper；无 auth 返回 400「未找到云平台授权信息」（与现有一致） |
| `start_vm.py`（`_reuse_running_instance` 等） | `filter(id=server_config.authorization_id)` | helper；无 auth 时已有「跳过复用」日志 |

### 6.3 调用点（Phase 2 可选，本计划不含）

以下文件同样模式，但当前 mock 路径未触发；可在 follow-up 批量替换：

- `get_occupied_cidr_blocks.py`
- `cloud/functions/network/get_cloud_*.py`
- `core/middleware.py`

### 6.4 错误处理契约

| API | authorization_id 非数值 PK | 无 CloudPlatformAuthorization |
|-----|---------------------------|------------------------------|
| `previous-server-config` | 200，返回 history，无 price/availability | 同左 |
| `stop-vm` | 走「未配置云平台授权」错误 SSE + 4xx | 不变 |
| `server-runtime-status` | 400 未找到授权（非 mock instance） | 不变 |
| start_vm 复用 | 跳过复用，继续创建流程 | 不变 |

**原则**：非数值 ID **不得** 抛 ValueError；应视为「无云授权」。

### 6.5 测试策略

1. **单元**：`tests/test_authorization_lookup.py` — `parse_*` 与 `get_*` 边界（`mock-auth`、空、数值 PK、错误 tenant）。
2. **服务**：
   - `test_get_previous_server_config.py`（已有）
   - `test_stop_vm_mock_authorization_id.py` — history/config 带 mock-auth，stop 不 500
   - 扩展 runtime status 测试 — config 带 mock-auth + 非 mock instance_id 时 400 而非 500

---

## 7. 验收标准

- [ ] `pytest tests/test_get_previous_server_config.py tests/test_authorization_lookup.py` 通过
- [ ] mock-auth 场景下 stop-vm / server-runtime-status 无 500
- [ ] 真实数值 authorization_id 回归测试仍绿
- [ ] 日志中不再出现 `Field 'id' expected a number but got 'mock-auth'`

---

## 8. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Snowflake ID 超大数 | Python `int()` 可处理；`isdigit()` 对正整数 PK 足够 |
| 漏改调用点 | grep `filter(id=.*authorization` 清单 + Phase 2 backlog |
| stop-vm mock 语义 | mock 实例通常无真实 VM；返回「未配置授权」符合预期 |

---

## 9. 实施增量（供价值流/计划引用）

1. **Increment 1**：helper + 单元测试  
2. **Increment 2**：`get_previous_server_config` 改用 helper（ refactor inline）  
3. **Increment 3**：`stop_vm` + 测试  
4. **Increment 4**：`get_server_runtime_status` + `start_vm` 复用路径 + 测试  

预估规模：~4 文件改动 + 3 测试文件，无 migration。
