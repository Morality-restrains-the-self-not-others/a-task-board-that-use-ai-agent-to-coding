---
name: deprecation-and-migration
description: 废弃与迁移管理。适用于替换旧系统/API/库、下线功能、合并重复实现、数据库 schema 变更（expand/contract 模式）、规划新系统生命周期。覆盖 Strangler、Adapter、Feature Flag 模式。
source: adapted from addyosmani/agent-skills
---

# 废弃与迁移（Deprecation & Migration）

## 概述

代码是负债，不是资产。每行代码都有持续维护成本 — 修复 bug、更新依赖、打安全补丁、新人上手。废弃是移除不再物有所值的代码的纪律，迁移是安全地将用户从旧系统移到新系统的过程。

大多数工程组织擅长构建东西。很少擅长移除它们。此技能解决这个差距。

## 核心原则

### 代码即负债

每行代码都在持续产生成本。当同样的功能可用更少代码、更低复杂度、更好抽象提供时——旧代码就应该消失。

### Hyrum's Law 使移除困难

有足够多的用户时，每个可观察行为都被依赖——包括 bug、时序怪异、未文档化的副作用。这就是为什么废弃需要主动迁移，而不只是公告。

### 废弃规划从设计时开始

构建新东西时问自己："三年后我们怎么移除它？"设计时考虑到清洁接口、feature flag、最小化暴露面的系统，比到处泄漏实现细节的系统更容易废弃。

## Compulsory vs Advisory 废弃

| 类型 | 何时使用 | 机制 |
|------|---------|------|
| **Advisory** | 迁移可选，旧系统稳定 | 警告、文档、引导。用户自主时间线迁移 |
| **Compulsory** | 旧系统有安全问题、阻塞进展、维护成本不可持续 | 硬期限。日期 X 前移除。必须提供迁移工具 |

**默认用 advisory。** 只在安全风险或维护成本证明强制迁移合理时才用 compulsory。强制废弃需要提供迁移工具、文档和支持。

## 迁移流程

### Step 1: 构建替代品

替代品必须：
- 覆盖旧系统所有关键用例
- 有文档和迁移指南
- 在生产环境验证（不只是"理论上更好"）

### Step 2: 公告并文档化

```markdown
## 废弃通知: OldService

**状态:** 自 2026-08-01 起废弃
**替代品:** NewService (详见迁移指南)
**移除日期:** Advisory — 暂无硬期限
**原因:** OldService 需要手动扩容、缺乏可观测性

### 迁移指南
1. 将 `import "old-service/client"` 替换为 `import "new-service/client"`
2. 更新配置 (见下方示例)
3. 运行迁移验证脚本
```

### Step 3: 增量迁移

消费者逐个迁移，不一锅端：

```
1. 识别与废弃系统的所有接触点
2. 更新为使用替代品
3. 验证行为一致（测试、集成检查）
4. 移除对旧系统的引用
5. 确认无回归
```

**The Churn Rule:** 如果你拥有被废弃的基础设施，你负责迁移你的用户——或提供不需要迁移的向后兼容更新。

### Step 4: 移除旧系统

仅在**所有**消费者都迁移后：

```
1. 验证零活跃使用（指标、日志、依赖分析）
2. 移除代码
3. 移除相关测试、文档、配置
4. 移除废弃通知
5. 庆祝——移除代码是一种成就
```

## 迁移模式

### Strangler Pattern（绞杀者模式）

新旧系统并行运行。流量从旧到新增量路由：

```
Phase 1: 新系统 0%，旧系统 100%
Phase 2: 新系统 10% (canary)
Phase 3: 新系统 50%
Phase 4: 新系统 100%，旧系统空闲
Phase 5: 移除旧系统
```

### Adapter Pattern（适配器模式）

创建适配器，将旧接口的调用翻译到新实现：

```go
// Adapter: 旧接口，新实现
type LegacyTaskService struct {
    newService *NewTaskService
}

func (s *LegacyTaskService) GetTask(id int) (*OldTask, error) {
    task, err := s.newService.FindByID(strconv.Itoa(id))
    if err != nil {
        return nil, err
    }
    return s.toOldFormat(task), nil
}
```

### Feature Flag 迁移

```go
func getTaskService(userID string) TaskService {
    if featureFlags.IsEnabled("new-task-service", userID) {
        return NewTaskService{}
    }
    return LegacyTaskService{}
}
```

### 数据库 Schema 迁移（Expand/Contract）

Schema 变更是最危险的迁移——数据是唯一无法通过回滚部署来撤销的。**绝不在原地修改列。** 以加法阶段迁移，使新旧代码在每个步骤都有效：

```
EXPAND ────────→ MIGRATE ────────→ CONTRACT
添加新列，可为空，   填充既有行，     一旦没有代码读取旧列，
与旧列并存           应用中双写新旧   在之后的独立部署中删除它
```

**示例 — 重命名 `name` 为 `full_name`：**

1. **Expand.** 添加 `full_name` 为 nullable。部署。（旧代码忽略它；无事发生）
2. **Dual-write.** 应用每次 insert/update 时写入 `name` 和 `full_name`。部署。
3. **Backfill.** 分批复制 `name → full_name` 到既有行（分批，不锁表）
4. **Switch reads.** 应用切换到读 `full_name`，继续双写。部署并观察。
5. **Contract.** 停止写 `name`，然后在**独立的后续部署**中删除该列

**规则：**
- **加法优先，破坏性最后且独立。** 添加（新 nullable 列、新表、新索引）在任何部署中安全；删除和重命名在**没有代码引用旧形态后**才部署
- **每个迁移有测试过的 down 路径。** 不能回滚的迁移就是不能撤销的部署
- **分批 backfill，不在热路径。** 单条 `UPDATE` 数百万行锁表；分块+限流
- **大索引不加锁构建**（Postgres: `CREATE INDEX CONCURRENTLY`）
- **风险切换用 feature flag 与代码解耦**

## 僵尸代码

僵尸代码是无人所有但人人依赖的代码。迹象：
- 超过 6 个月无提交但有活跃消费者
- 未分配的维护人或团队
- 无人修复的失败测试
- 带有已知漏洞且无人更新的依赖

**响应：** 要么分配 owner 并正常维护，要么以具体迁移计划废弃它。

## 常见借口

| 借口 | 现实 |
|------|------|
| "还能用，为什么移除？" | 无人维护的工作代码默默积累安全债务和复杂度 |
| "将来可能有人需要" | 如果被需要，可以重建。留着"以防万一"比重建更贵 |
| "迁移太贵了" | 比较 2-3 年的迁移成本 vs 持续维护成本。迁移通常长期更便宜 |
| "就原地重命名列，就一行" | 部署期间旧代码和新代码同时运行 — 其中一个会查询已不存在的列。Expand/contract，绝不原地重命名 |
| "我会在同一迁移中加列和删旧列" | 把安全添加和破坏性删除耦合了。删除在各自部署中进行 |

## 红旗

- 已废弃但没有替代品的系统
- 废弃公告没有迁移工具或文档
- "软"废弃维持 advisory 数年无进展
- 无 owner 且有活跃消费者
- Schema 变更和依赖它的代码在同一部署中发版
- 列原地重命名或删除而非 expand/contract
- 合并的迁移没有测试过的 down 路径
