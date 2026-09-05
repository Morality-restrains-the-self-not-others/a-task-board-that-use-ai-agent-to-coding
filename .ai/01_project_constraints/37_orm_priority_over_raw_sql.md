# ORM 优先使用规范（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-27
- 最后修改：2026-07-27
- 维护者：Trae AI 团队

## 规则概述

所有数据库操作**必须优先使用 ORM 库**（Django ORM / GORM 等），禁止在业务代码中直接编写原始 SQL 语句，确保数据库操作的类型安全、可维护性与 SQL 注入防护。

## 适用范围

**monorepo 内全部服务**，包括但不限于：

| 技术栈 | ORM 库 | 说明 |
|--------|--------|------|
| Python / Django | Django ORM（`models.Model`、`QuerySet`） | 默认查询构造器 |
| Go | GORM（`gorm.io/gorm`） | Go 服务标准 ORM |
| Go（通用工具） | `shareLib` 中约定的 ORM/数据访问层 | 按仓库共享约定 |

## 核心规则

### 1. 业务代码禁止裸 SQL

- **禁止**在业务逻辑（领域服务、应用服务、视图/控制器、事件处理器等）中直接编写 `raw()`、`execute()`、`QueryRaw()` 等原始 SQL 字符串
- **禁止**使用字符串拼接或模板构造 SQL 语句
- **禁止**在日志、配置或环境变量中内联 SQL 片段作为业务查询

### 2. CRUD 必须走 ORM

所有增删改查操作必须通过 ORM 的查询构造器完成：

- **查询**：`Model.objects.filter()` / `db.Where().Find()` 等链式 API
- **创建**：`Model.objects.create()` / `db.Create()` 
- **更新**：`Model.objects.update()` / `db.Updates()`
- **删除**：`Model.objects.delete()` / `db.Delete()`
- **聚合**：`Model.objects.aggregate()` / `db.Raw()` 仅在聚合函数 ORM 不直接支持时允许（见例外）

### 3. 关联查询走 ORM 预加载

- Django：使用 `select_related()`、`prefetch_related()` 处理关联，禁止手写 JOIN
- GORM：使用 `db.Preload()`、`db.Joins()`（ORM 级 JOIN）处理关联

### 4. 复杂查询优先用 ORM 能力

遇到复杂查询需求时，按以下优先级选择：

1. **ORM 原生能力**（`Q` 对象、`F` 表达式、`Subquery`、`Exists` / GORM `Scopes`、`Clauses`）
2. **ORM 扩展/插件**（如 `django-cte`、`django-sql-utils` 等，需评估后引入）
3. **只读视图 + ORM Model**（创建数据库视图，映射为只读 ORM Model 查询）
4. **仓储封装**（将原始 SQL 封装在仓储层，对外暴露类型安全的接口；见例外节）

## 例外（允许使用原始 SQL）

以下场景经评估后可降级使用原始 SQL，但**必须**满足封装与审计要求：

| 场景 | 条件 | 要求 |
|------|------|------|
| **数据库 Migration（DDL）** | 框架 Migration 系统不支持的 DDL（如自定义索引类型、物化视图、分区表） | 走 `migrations.RunSQL`，注明原因；不影响业务查询代码 |
| **聚合/分析查询** | ORM 无法生成的复杂聚合（窗口函数、递归 CTE、跨库联邦查询等） | 封装在仓储层单一方法内，返回类型安全的 DTO/Model；方法签名不暴露 SQL |
| **批量数据操作** | 万级以上记录的批量 upsert/update，ORM 逐行操作性能不可接受 | 封装在仓储层，使用参数化查询（禁止字符串拼接）；附带性能对比注释（ORM 版本耗时 vs 原始 SQL 耗时） |
| **数据库特性查询** | 特定数据库专有语法（如 PostgreSQL `JSONB` 高级操作符、MySQL `MATCH...AGAINST` 全文搜索），ORM 不支持 | 封装在仓储层；方法名明确表达语义（如 `search_by_fulltext`），不暴露 SQL |
| **只读报表/数据导出** | 跨大量表的复杂只读查询，构造 ORM 链过长且性能不达标 | 可创建数据库视图 + 映射只读 Model；或封装在仓储层返回原始数据结构 |
| **数据迁移脚本（dataMigrate）** | 批量数据清洗、修复、迁移（`dataMigrate/` 目录下的 SQL 脚本） | 适用本目录既有规范；见 [dataMigrate 目录规范](./34_data_migrate_directory_standard.md) |

## 封装要求（当必须使用原始 SQL 时）

1. **位置**：原始 SQL 必须封装在**仓储层**（Repository / DAO），不得泄漏到领域层或应用层
2. **参数化**：**必须**使用参数化查询（`%s` / `$1` 占位符），**禁止**字符串拼接用户输入
3. **返回类型**：方法返回域对象、DTO 或值对象，**禁止**返回裸 `dict` / `map[string]interface{}` 或游标
4. **注释**：方法上方须注释说明为何无法用 ORM 实现，附 ORM 尝试的简要过程或等效（伪）代码
5. **测试**：原始 SQL 路径必须有对应的单元/集成测试覆盖，验证正确性与 SQL 注入防护

## 与相关规则的联动

| 规则 | 关系 | 说明 |
|------|------|------|
| 第 21 条「单库/单表单服务数据所有权」 | 互补 | 所有权规则管"谁能访问哪张表"；本条管"访问时用什么方式" |
| 第 33 条「数据库 ID 雪花算法生成」 | 配合 | ID 仅在入库时转 DB 类型，ORM 负责该转换 |
| 「服务端性能规范」N+1 规避 | 增强 | N+1 规避依赖 ORM 的 `select_related` / `prefetch_related` 能力 |
| 「服务端 DDD 规范」基础设施隔离 | 一致 | 仓储层封装原始 SQL 与 DDD 基础设施隔离原则对齐 |
| 「DDD 测试原则」 | 一致 | ORM 查询更易于单元测试（无需真实数据库），原始 SQL 必须有集成测试 |
| [dataMigrate 目录规范](./34_data_migrate_directory_standard.md) | 补充 | dataMigrate 目录下的 SQL 脚本适用自身规范，不因本条禁止 |

## Agent 执行指引

当 Agent 编写或审查涉及数据库操作的代码时：

1. **新建查询**：优先使用 ORM 构造器，检查是否有等价 ORM 写法
2. **遇到 ORM 限制**：先尝试 F 表达式 / Subquery / CTE 等高级 ORM 特性，再评估是否命中例外
3. **被迫使用原始 SQL**：封装在仓储层，参数化，加注释说明原因，补测试
4. **存量代码**：发现裸 SQL 时评估是否可改为 ORM；若涉及功能修改（非纯迁移/数据修复），须同步重写为 ORM（或符合封装例外要求）
5. **Code Review**：原始 SQL 出现于业务层视为**阻塞项**（blocker），必须移至仓储层或改为 ORM

## 变更日志

- 2026-07-27：版本 1.0.0 - 初始版本；作为项目约束第 35 条生效
