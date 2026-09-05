# 设计：默认服务器启动配置迁入机器节点策略

- 日期：2026-07-15
- 状态：已采用（goal-mode 自动采纳）
- 类型：前端设置入口迁移（无新增后端 API）

## 1. 目标与成功标准

将项目页「快速应用默认模版」下拉的**管理面**（即租户级「默认服务器启动配置」）从「云平台绑定」页迁到「工作空间管理 → 机器节点策略」中。

| # | 成功标准 |
|---|----------|
| S1 | `/settings/task-panel/` →「机器节点」模态内可打开「默认服务器启动配置」并完成读写 |
| S2 | `/settings/cloud-platform/` 授权行不再提供「默认服务器启动配置」入口 |
| S3 | 项目页「快速应用默认模版」仍消费同一 `server-config-default` API；空态/错误文案指向「工作空间管理 → 机器节点」 |
| S4 | 既有 Playwright：机器策略模态 + 默认配置表单链路更新并通过 |

## 2. 背景

- 消费面：`ProjectRunTemplatePanel` → `GET .../cloud/server-config-default/`
- 管理面现状：`WorkspaceSettingsCloudPlatform` → `CloudPlatformAuthorizationRow` → `SetDefaultConfigModal`
- 机器策略：`WorkspaceSettingsTaskPanel` → `WorkspaceMachinePolicyModal`（已加载 CPA 列表）

## 3. 方案对比（已选 A）

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 机器策略模态内嵌入口** | 在策略模态增加「默认服务器启动配置」分区，按授权列出并打开现有 `SetDefaultConfigModal`；云平台行移除按钮 | **采用**：改动面小、与「启用的机器节点」同屏、复用表单 |
| B. task-panel 独立页签 | 新设置子页 | 过重，与现有行级「机器节点」交互不一致 |
| C. 双入口并存 | 云平台保留 + 机器策略新增 | 违背「一并过来」的搬迁语义 |

## 4. 设计细节

### 4.1 UI

在 `WorkspaceMachinePolicyModal`：

1. 加宽容器（`max-w-2xl`），说明文案补充「及默认服务器启动配置」。
2. 新增分区「默认服务器启动配置」：
   - 列出租户云平台授权（与上方启用列表同源 `authorizations`）
   - 每行：授权标签 + 按钮「配置默认启动」（`data-alias="open-default-server-config"`）
   - 无授权时提示先去「云平台绑定」添加授权
3. 嵌套打开 `SetDefaultConfigModal`（`z-index` 高于策略模态，现有组件 `z-50` 可改为策略 `z-40` / 配置 `z-50`，或配置层再叠一层）
4. 策略「保存」与默认配置保存**解耦**：配置模态自管 POST，不强制先保存策略

### 4.2 移除

- `CloudPlatformAuthorizationRow`：删除「默认服务器启动配置」按钮与 `set-default` emit
- `WorkspaceSettingsCloudPlatform`：移除 `SetDefaultConfigModal` 挂载与 `openSetDefaultModal`

### 4.3 文案

`ProjectRunTemplatePanel` 及错误映射：

- 「工作空间设置 → 云平台」→「工作空间管理 → 机器节点」

### 4.4 API / 权限 / 架构

- **无新接口**；仍用 `server-config-default` CRUD 与既有 CPA API
- **无权限模型变更**
- **无架构图变更**（`docs/architecture/` 无 current 基线需更新；纯前端入口）

## 5. 测试

- 单元：机器策略模态含默认配置分区（若有 Vue test）；文案断言
- Playwright：将 `WorkspaceSettingsCloudPlatform.set-default-config-reload` 入口改为 task-panel → 机器节点 → 配置默认启动；扩展 `WorkspaceSettings.machine-policy-modal` 断言分区可见

## 6. 非目标

- 不改变 `server-config-default` 数据结构或项目 `server_run_template` 存储
- 不把默认配置改为「按工作空间隔离」（仍为租户级授权维度）
