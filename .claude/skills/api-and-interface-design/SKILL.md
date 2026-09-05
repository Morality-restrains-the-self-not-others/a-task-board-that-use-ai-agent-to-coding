---
name: api-and-interface-design
description: 稳定 API 与接口设计。适用于新增 API 端点、定义模块边界、微服务间契约设计，以及任何跨服务/跨模块的接口变更。遵循 Hyrum's Law 与向后兼容原则。
source: adapted from addyosmani/agent-skills
---

# API 与接口设计

## 概述

为 20+ 微服务架构设计的 API 契约规范。每个端点、每个模块边界、每个服务间调用都是一个契约——设计时就必须考虑稳定性、可扩展性和兼容性。

## 核心原则

### Hyrum's Law（海勒姆定律）

> 当 API 的用户数量足够多时，系统的**所有可观察行为**都会被某个人依赖——无论你在契约中承诺了什么。

这意味着：每个公开行为——包括未文档化的怪癖、错误消息文本、时序、排序——一旦被用户依赖，就成了事实契约。

**实践含义：**
- **有意识地控制暴露面**：每个可观察行为都是潜在的承诺
- **不泄漏实现细节**：用户能观察到的，他们就会依赖
- **设计时就要规划废弃路径**：参见 `deprecation-and-migration` 技能
- **仅靠测试不够**：即使契约测试完美通过，依赖未文档化行为的真实用户仍会受影响

### One-Version Rule（单版本规则）

避免强制消费者在同一个依赖的多个版本之间做选择。为只有一个版本存在的世界设计——**扩展而非分叉**。

### 1. 契约优先

在实现之前定义接口。契约就是 spec，实现在后。

```go
// Go: 定义 API 契约
type TaskAPI struct {
    CreateTask(ctx context.Context, input CreateTaskInput) (*Task, error)
    ListTasks(ctx context.Context, params ListTasksParams) (*PaginatedResult[Task], error)
    GetTask(ctx context.Context, id string) (*Task, error)
    UpdateTask(ctx context.Context, id string, input UpdateTaskInput) (*Task, error)
    DeleteTask(ctx context.Context, id string) error  // 幂等
}
```

### 2. 一致的错误语义

项目的 API URL 遵循 `/api/${serviceName}/${funcName}/key/value/...` 模式。每个服务必须有**唯一且一致的**错误格式：

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Email is required",
    "details": {}
  }
}
```

HTTP 状态码映射：
| 状态码 | 含义 |
|--------|------|
| 400 | 客户端发送了无效数据 |
| 401 | 未认证 |
| 403 | 已认证但未授权 |
| 404 | 资源未找到 |
| 409 | 冲突（重复、版本不匹配） |
| 422 | 校验失败（语义无效） |
| 500 | 服务器错误（**绝不暴露内部细节**） |

### 3. 边界验证

信任内部代码，在外部输入进入系统的边界处验证：

```go
// Go: 在 HTTP handler 边界验证
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var input CreateTaskInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        respondError(w, http.StatusBadRequest, "INVALID_JSON", "Failed to parse request body")
        return
    }
    if err := h.validator.Struct(input); err != nil {
        respondError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
        return
    }
    // 验证通过后，内部代码信任类型安全
    task, err := h.service.CreateTask(r.Context(), input)
    // ...
}
```

**验证归属地**：API handler（用户输入）→ 表单提交（用户输入）→ 第三方 API 响应解析（**始终视为不信任数据**）→ 环境变量加载（配置）

**不需要验证的地方**：共享类型契约的内部函数之间、已被验证代码调用的工具函数、刚从自己数据库取出的数据

### 4. 加法优于修改

不破坏既有消费者的前提下扩展接口：

```go
// Good: 新增可选字段
type CreateTaskInput struct {
    Title       string   `json:"title" validate:"required,min=1,max=200"`
    Description *string  `json:"description,omitempty"` // 可选
    Priority    *string  `json:"priority,omitempty"`    // 后加，可选
    Labels      []string `json:"labels,omitempty"`      // 后加，可选
}

// Bad: 修改既有字段类型或删除字段
type CreateTaskInput struct {
    Title       string `json:"title"`
    // Description string `json:"description"` // ❌ 删除—破坏既有消费者
    Priority    int    `json:"priority"`       // ❌ 类型改变
}
```

### 5. 可预测的命名

| 模式 | 约定 | 示例 |
|------|------|------|
| URL 路径 | `/api/${service}/${func}/key/value/...` | `/api/taskTaskService/createTask/` |
| 查询参数 | camelCase | `?sortBy=createdAt&pageSize=20` |
| 响应字段 | camelCase | `{ "createdAt": ..., "taskId": ... }` |
| 布尔字段 | is/has/can 前缀 | `isCompleted`, `hasAttachments` |
| 枚举值 | UPPER_SNAKE | `"IN_PROGRESS"`, `"COMPLETED"` |

## REST 资源设计

```
GET    /api/taskTaskService/listTasks/page/1/size/20        → 列表（含分页）
POST   /api/taskTaskService/createTask                       → 创建
GET    /api/taskTaskService/getTask/id/xxx                   → 获取单个
PATCH  /api/taskTaskService/updateTask/id/xxx                → 部分更新
DELETE /api/taskTaskService/deleteTask/id/xxx                → 删除（幂等）
```

## 分页

所有列表端点必须有分页：

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "pageSize": 20,
    "totalItems": 142,
    "totalPages": 8
  }
}
```

## 常见借口 vs 现实

| 借口 | 现实 |
|------|------|
| "以后再写 API 文档" | 类型签名就是文档。先定义它们 |
| "现在不需要分页" | 当有人有 100+ 条数据时就需要了。从一开始就加 |
| "没人用那个未文档化的行为" | Hyrum's Law：如果可观察，就有人依赖 |
| "内部 API 不需要契约" | 内部消费者仍然是消费者。契约防止耦合，实现并行开发 |

## 红旗

- 端点返回形状随条件变化
- 跨端点错误格式不一致
- 验证散落在内部代码中而非集中在边界
- 对既有字段的破坏性变更（类型改变、删除）
- 列表端点无分页
- URL 中出现动词（`/api/createTask`, `/api/getUsers`）
- 第三方 API 响应未经验证直接使用

## 验证

API 设计完成后确认：
- [ ] 每个端点有类型化的输入/输出 schema
- [ ] 错误响应遵循统一格式
- [ ] 验证仅在系统边界执行
- [ ] 列表端点支持分页
- [ ] 新增字段是可选的加法（向后兼容）
- [ ] 命名遵循一致的跨端点约定
- [ ] API 文档/类型与实现一起提交
