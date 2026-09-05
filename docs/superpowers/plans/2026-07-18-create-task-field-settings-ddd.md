# DDD 摘记：创建任务可选字段显隐

**聚合**：WorkspaceCreateTaskFieldSettings（workspace_id 为标识）  
**值对象**：FieldVisibilityMap（已知键 → bool）  
**仓储**：SQLite `workspace_create_task_field_settings`（taskProjectService 所有）  
**应用服务**：GET/PUT handlers（无领域事件；配置 CRUD 书面例外，见设计 §2）

不跨服务写任务表；消费方为 Vue CreateTaskModal（读模型）。
