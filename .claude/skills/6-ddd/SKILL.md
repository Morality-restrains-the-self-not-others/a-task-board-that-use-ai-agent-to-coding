---
name: 6-ddd
description: "Step 6 - DDD Domain Modeling: extract domain models from value stream increments. Identify bounded contexts, entities, value objects, aggregates, domain services, repository interfaces, and domain events. Consumes /5-nfr 领域模型影响 plus 路径分片键审视 and 幂等性审视 tables (idempotency keys must match the business duplicate boundary). Produces domain model files that serve as the contract for TDD."
---

# /6-ddd — DDD 领域建模

## 概述

消费价值流增量（来自 `/4-value-stream`）与 NFR 澄清决策（来自 `/5-nfr`），为即将交付的增量提取领域模型，
生成领域层代码骨架。领域层是后续 TDD 和实现的契约——

**NFR 澄清定义「做到什么程度」，领域模型定义「是什么」，TDD 验证「对不对」，Build 实现「怎么做」。**

**核心原则：领域层是纯业务逻辑，不依赖任何基础设施。高层模块（领域）不依赖低层模块（基础设施），两者都依赖抽象（端口接口）。**

## 依赖反转原则 (Dependency Inversion Principle)

本技能的核心交付物不仅是领域模型，更是**可替换基础设施的端口-适配器架构**。

### 什么是依赖反转

```
❌ 传统分层（上层依赖下层 —— 换库 = 改业务代码）:
  领域层 → ORM / Kafka SDK / 云 SDK / HTTP 客户端

✅ 依赖反转（下层依赖上层定义的抽象 —— 换库 = 新增适配器）:
  领域层 ←[实现]— 基础设施层
    ↓ 定义
  端口接口 (ABC)
    ↑ 实现
  适配器 A / 适配器 B / 适配器 C
```

### 端口-适配器（六边形）架构

```
                    ┌──────────────────────────────┐
                    │      领域层 (Domain)           │
                    │  Entity / VO / Aggregate      │
                    │  Domain Service (纯业务逻辑)    │
                    │  Port Interface (ABC 端口定义)  │
                    │  Domain Event (业务事件契约)    │
                    └──────────┬───────────────────┘
                               │ 实现
              ┌────────────────┼────────────────┐
              │                │                │
     ┌────────┴────────┐ ┌────┴─────┐ ┌────────┴────────┐
     │  PostgreSQL 适配器│ │Kafka 适配器│ │  AWS S3 适配器   │
     │ (实现 Repository) │ │(实现 Event│ │ (实现 Storage)  │
     │                  │ │  Bus)    │ │                 │
     └─────────────────┘ └──────────┘ └─────────────────┘
```

**关键约束：箭头永远指向领域层。** 基础设施适配器依赖领域层定义的端口接口，领域层不知道也不关心适配器的存在。切换数据库/消息队列/云服务商时，只需新增适配器实现同一端口接口，领域层代码**零改动**。

### 依赖库切换的成本阶梯

| 切换场景 | 无端口-适配器 | 有端口-适配器 |
|---------|-------------|-------------|
| MySQL → PostgreSQL | 改所有 DAO/Repository 实现 | 新建 PostgresAdapter，实现同一 Repository 接口 |
| Kafka → Redis Streams | 改所有生产者/消费者 | 新建 RedisStreamAdapter，实现同一 EventBus 接口 |
| 本地文件存储 → AWS S3 | 改所有文件操作代码 | 新建 S3StorageAdapter，实现同一 StoragePort 接口 |
| 内存缓存 → Redis | 改所有缓存调用 | 新建 RedisCacheAdapter，实现同一 CachePort 接口 |
| HTTP 直连 → gRPC | 改所有远程调用 | 新建 gRPCAdapter，实现同一 RemoteServicePort 接口 |

**目标：任何基础设施替换，领域层和应用层代码变更量为 0。**

## 在流水线中的位置

```
1-brainstorming → 2-role-permission → [3-worktrees] → 4-value-stream → 5-nfr → [6-ddd] → 7-plans → 8-build → 9-review → 10-ship
                                                                                ↑
                                                                          你在这里
```

权威编号见 `.ai/11_ai_development/03_superpowers_workflow.md`。DDD 在价值流和 NFR 澄清之后、计划之前执行。价值流确定了要交付什么、按什么顺序交付；
NFR 澄清确定了每个增量需要达到的质量等级；DDD 为这些增量建模领域，根据 NFR 决策选择
聚合粒度、一致性模型和架构模式，不做过度的全量设计。

## 双重角色

### 主要角色（步骤 6）：领域建模阶段

作为流水线的第 6 步，消费价值流增量和 NFR 澄清决策，产出：
- 领域模型文件（entity/VO/aggregate/port/event）
- 端口接口（ABC 抽象 —— 领域层定义"我需要什么"，基础设施层实现"怎么做"）
- 领域事件契约
- 应用服务骨架（编排领域服务，注入端口实现）

**关键输入文件：**
- `docs/superpowers/plans/YYYY-MM-DD-<topic>-value-stream.md` — 价值流增量
- `docs/superpowers/plans/YYYY-MM-DD-<topic>-nfr-clarification.md` — NFR 质量等级与领域模型影响

### 辅助角色（横切）：合规检查

在步骤 9（代码审查）和步骤 10（交付）时，本技能可被重新调用执行合规审计：
- 扫描领域层文件，检查是否有基础设施导入（ORM、Kafka、云 SDK）
- 验证聚合边界和事件流
- 验证端口-适配器架构完整性（每个端口有对应适配器，适配器不直连领域实体）
- 运行 `python runAll/scripts/ci/check_ddd_bdd_compliance.py`

审查发现问题 → 调整领域模型 → 再次通过本技能生成正确的骨架。

## 何时使用

**主要：**
- 价值流已批准，NFR 澄清已完成，需要为即将交付的增量建模领域
- 新功能涉及新的限界上下文或聚合
- 需要为外部依赖定义可替换的端口接口

**横切（审查/交付时调用）：**
- 代码审查发现领域层违规
- 交付前最后一轮合规检查

**前置条件：** 已完成价值流映射（`/4-value-stream`）和 NFR 澄清（`/5-nfr`），有价值流文档和 NFR 澄清文档。

**跳过条件：** 纯前端/UI 改动、配置变更、文档修改、或价值流增量不涉及新的业务概念。

## 执行清单

你必须为以下每一项创建任务并按顺序完成：

1. **读取价值流、NFR 澄清与设计文档** — 了解增量范围和 NFR 质量要求
2. **识别限界上下文** — 从增量中划分业务边界
3. **提取实体与值对象** — 识别有标识的对象和无标识的值
4. **定义聚合与聚合根** — 确定一致性边界
5. **设计领域服务** — 跨实体/跨聚合的业务逻辑
6. **定义端口接口** — ABC 抽象，涵盖仓储、事件总线、通知、存储、缓存等所有外部依赖
7. **识别领域事件** — 上下文间的通信契约
8. **设计应用服务** — 编排领域服务，通过构造函数注入端口实现，完成依赖组装
9. **生成领域模型文件** — 写入 `domain/` 目录
10. **自检** — 验证领域层无基础设施依赖，端口接口完整，依赖反转成立

## 领域建模流程

### 1. 读取价值流、NFR 澄清与设计文档

加载已批准的价值流文档（`docs/superpowers/plans/YYYY-MM-DD-<topic>-value-stream.md`）、
NFR 澄清文档（`docs/superpowers/plans/YYYY-MM-DD-<topic>-nfr-clarification.md`）
和设计文档。只关注当前要交付的增量，不建模未来增量。

**关键动作：** 阅读 NFR 澄清文档的「领域模型影响」表格，记录每个影响模型的 NFR 决策，
作为后续建模步骤的约束条件。例如：
- NFR 要求强一致性 L3 → 聚合不能拆分
- NFR 要求性能 L3 → 需考虑 CQRS 读写分离
- NFR 要求审计 L2 → 需增加 AuditEvent 领域事件
- NFR 要求可用性 L3 → 端口接口需增加降级/兜底方法
- **「路径分片键审视」表**：无分片 ID 且需伸缩 → 聚合根/契约补分片键；分片 ID 不适配 → 纠正聚合身份与仓储必参（元规则 43）
- **「幂等性审视」表**：副作用路径的幂等键须与业务重复边界同粒度；键过粗/缺失 → 命令与事件契约补键，仓储按该键去重；实体用状态转移而非 `increment()`（元规则 48）。Kafka 消费落地须走共享 `IdempotentDispatchService`（元规则 49 / ADR-0015）

### 2. 识别限界上下文 (Bounded Context)

从增量中识别语义边界。每个上下文内部模型一致，上下文间通过事件/接口通信。

**判断标准：**
- 相同的「用户」在不同场景是否含义不同？（如：下单用户 vs 物流用户 → 不同上下文）
- 业务规则是否在一个边界内自洽？
- 是否可以独立演进？
- 是否有独立的端口接口集？（每个上下文有自己的端口，不共享基础设施适配器细节）

**输出：** 列出所有限界上下文，每个一句话描述职责。

```
上下文清单：
- 用户认证上下文：注册、登录、密码管理
- 工作空间上下文：创建与管理工作空间
- 任务管理上下文：任务的创建、分配、执行
```

### 3. 提取实体 (Entity)

实体是有唯一标识、生命周期可变的对象。

**提取规则：**
- 有独立 ID（雪花算法生成的全局唯一 ID）
- 状态会随时间变化
- 通过 ID 判断相等性（而非属性）

**实体建模模板：**
```python
# domain/entities/{entity_name}.py
class EntityName:
    """领域实体描述"""

    def __init__(self, id: int, ..., created_at=None, updated_at=None):
        self.id = id
        self.created_at = created_at or datetime.now()
        self.updated_at = updated_at or datetime.now()

    # 领域行为方法（动词命名）
    def activate(self) -> None:
        """激活实体"""
        ...

    # 注意：领域实体不依赖任何外部库
    # 不使用 ORM 基类、不导入数据库驱动、不直接调用外部服务
```

**禁止：**
- 实体中禁止出现 ORM 导入（`models.Model`, `ForeignKey`, `OneToOneField` 等）
- 禁止直接调用外部服务
- 字段禁止 `null=True` 语义（除非有明确业务注释说明可为空的原因）

### 4. 提取值对象 (Value Object)

值对象无唯一标识，通过属性值判断相等，不可变。

**提取规则：**
- 没有独立 ID
- 不可变（创建后不修改）
- 通过全部属性值判断相等
- 自验证（构造时检查合法性）

**值对象建模模板：**
```python
# domain/value_objects/{name}.py
class ValueObjectName:
    """值对象描述"""

    def __init__(self, field1: str, field2: str):
        self._field1 = field1
        self._field2 = field2
        self._validate()

    def _validate(self) -> None:
        if not self._field1:
            raise ValueError("field1 不能为空")

    def __eq__(self, other):
        if not isinstance(other, ValueObjectName):
            return False
        return self._field1 == other._field1 and self._field2 == other._field2

    def __hash__(self):
        return hash((self._field1, self._field2))
```

### 5. 定义聚合 (Aggregate) 与聚合根 (Aggregate Root)

聚合是一组必须保持一致的实体和值对象。聚合根是外部访问聚合的唯一入口。

**判断标准：**
- 哪些对象必须同时保持一致？（→ 同一聚合）
- 哪个实体是其他对象的外部入口？（→ 聚合根）
- 聚合尽量小，一个聚合一个根

**输出：** 对每个限界上下文，列出聚合及聚合根。

```
用户认证上下文聚合：
- 聚合根: User
  - 值对象: PhoneNumber, EmailAddress
  - 关联: VerificationCode (独立实体)
```

### 6. 设计领域服务 (Domain Service)

当业务逻辑不属于任何单一实体时，归入领域服务。

**何时需要领域服务：**
- 操作涉及多个聚合
- 复杂的业务计算或规则
- 需要编排多个实体协作

**领域服务模板：**
```python
# domain/services/{service_name}.py
class DomainServiceName:
    """领域服务描述

    领域服务只依赖端口接口（ABC），不依赖具体实现。
    """

    def __init__(self, dependency_port: DependencyPort):
        """通过端口接口注入依赖

        Args:
            dependency_port: 端口接口的任意实现（基础设施适配器）
        """
        self._dependency_port = dependency_port

    def execute_operation(self, param: Entity) -> Result:
        """执行业务操作

        只调用端口接口定义的方法，不关心底层是哪个实现。
        """
        ...
```

**要求：** 领域服务通过构造函数注入端口接口（ABC），不直接实例化具体实现。领域服务的方法只操作领域对象和端口接口——不导入 ORM、HTTP 客户端、消息队列 SDK。

### 7. 定义端口接口 (Port Interfaces) ⭐ 依赖反转核心

端口接口是领域层定义「我需要什么能力」的抽象契约。基础设施层通过实现这些接口来提供能力。
**这是依赖反转的关键：领域层定义接口，基础设施层实现接口。**

#### 7.1 端口接口类型全景

对每个限界上下文，识别需要与外部世界交互的**所有**接触点，为每个接触点定义端口接口：

| 端口类型 | 目录 | 职责 | 典型适配器实现 |
|---------|------|------|-------------|
| **仓储端口** (RepositoryPort) | `domain/ports/repositories/` | 聚合持久化与查询 | PostgreSQL / MySQL / MongoDB / 内存 |
| **事件总线端口** (EventBusPort) | `domain/ports/event_bus/` | 领域事件发布与订阅 | Kafka / Redis Streams / RabbitMQ / 内存 Channel |
| **通知端口** (NotificationPort) | `domain/ports/notifications/` | 发送邮件/短信/推送通知 | SendGrid / AWS SES / 阿里云短信 / 控制台 Logger |
| **存储端口** (StoragePort) | `domain/ports/storage/` | 文件/对象存储 | AWS S3 / MinIO / 本地文件系统 / 阿里云 OSS |
| **缓存端口** (CachePort) | `domain/ports/cache/` | 临时数据缓存 | Redis / Memcached / 内存 Dict |
| **身份端口** (IdentityPort) | `domain/ports/identity/` | 用户认证与授权 | OIDC / LDAP / 本地 JWT / API Key |
| **外部服务端口** (ExternalServicePort) | `domain/ports/external/` | 第三方 API 调用防腐层 | 支付网关 / 地图服务 / AI 服务 |

**识别端口的方法：** 遍历领域服务的方法，找出所有需要「从外部获取数据」「向外部发送数据」「在外部存储数据」的操作——每个这样的外部依赖都是一个端口。

#### 7.2 仓储端口模板

```python
# domain/ports/repositories/{entity_name}_repository.py
from abc import ABC, abstractmethod
from typing import Optional

class EntityNameRepository(ABC):
    """实体仓储端口接口

    定义聚合持久化的抽象契约。实现者可以是 PostgreSQL、MySQL、MongoDB 或内存存储。
    领域层只依赖此接口，不关心具体实现。
    """

    @abstractmethod
    def save(self, entity: EntityName) -> EntityName:
        """保存聚合，返回最新状态"""
        pass

    @abstractmethod
    def find_by_id(self, entity_id: int) -> Optional[EntityName]:
        """按 ID 查询聚合，不存在返回 None"""
        pass

    @abstractmethod
    def delete(self, entity: EntityName) -> None:
        """删除聚合"""
        pass
```

**规则：**
- 一个聚合根对应一个仓储端口
- 仓储只操作聚合根，子实体通过聚合根访问
- 禁止在端口接口中导入 ORM 或数据库驱动
- 查询方法返回领域实体，不是数据库行对象

#### 7.3 事件总线端口模板

```python
# domain/ports/event_bus/event_bus.py
from abc import ABC, abstractmethod
from typing import List
from domain.events.base import DomainEvent

class EventBusPort(ABC):
    """事件总线端口接口

    定义领域事件发布与订阅的抽象契约。
    实现者可以是 Kafka、Redis Streams、RabbitMQ 或内存 Channel。
    """

    @abstractmethod
    def publish(self, events: List[DomainEvent]) -> None:
        """发布领域事件

        领域服务调用此方法发布事件，不关心底层是 Kafka 还是内存队列。
        """
        pass

    @abstractmethod
    def subscribe(self, event_type: str, handler) -> None:
        """订阅领域事件

        Args:
            event_type: 事件类型标识
            handler: 事件处理回调，签名为 (DomainEvent) -> None
        """
        pass
```

#### 7.4 通知端口模板

```python
# domain/ports/notifications/notification.py
from abc import ABC, abstractmethod
from dataclasses import dataclass

@dataclass(frozen=True)
class NotificationMessage:
    """通知消息值对象 —— 领域层定义的通用通知模型"""
    recipient: str       # 接收者标识（邮箱/手机号/用户ID）
    subject: str
    body: str
    channel: str         # "email" | "sms" | "push"

class NotificationPort(ABC):
    """通知端口接口

    定义消息通知的抽象契约。
    实现者可以是 SendGrid、AWS SES、阿里云短信等任意通知服务。
    """

    @abstractmethod
    def send(self, message: NotificationMessage) -> bool:
        """发送通知，返回是否发送成功"""
        pass
```

#### 7.5 存储端口模板

```python
# domain/ports/storage/file_storage.py
from abc import ABC, abstractmethod
from typing import BinaryIO, Optional

class FileStoragePort(ABC):
    """文件存储端口接口

    定义文件/对象存储的抽象契约。
    实现者可以是 AWS S3、MinIO、本地文件系统、阿里云 OSS 等。
    """

    @abstractmethod
    def upload(self, key: str, data: bytes, content_type: str = None) -> str:
        """上传文件，返回可访问的 URL 或 key"""
        pass

    @abstractmethod
    def download(self, key: str) -> Optional[bytes]:
        """下载文件，不存在返回 None"""
        pass

    @abstractmethod
    def delete(self, key: str) -> bool:
        """删除文件，返回是否成功"""
        pass

    @abstractmethod
    def exists(self, key: str) -> bool:
        """检查文件是否存在"""
        pass
```

#### 7.6 缓存端口模板

```python
# domain/ports/cache/cache.py
from abc import ABC, abstractmethod
from typing import Optional, Any

class CachePort(ABC):
    """缓存端口接口

    定义临时数据缓存的抽象契约。
    实现者可以是 Redis、Memcached 或内存字典。
    """

    @abstractmethod
    def get(self, key: str) -> Optional[Any]:
        """获取缓存值"""
        pass

    @abstractmethod
    def set(self, key: str, value: Any, ttl_seconds: int = 300) -> None:
        """设置缓存值"""
        pass

    @abstractmethod
    def delete(self, key: str) -> None:
        """删除缓存"""
        pass
```

#### 7.7 端口接口设计原则

1. **接口属于领域层**：端口接口和领域实体放在同一 `domain/` 目录下
2. **接口小而专注**：一个端口只做一件事（遵循接口隔离原则 ISP）
3. **方法返回领域对象**：不返回数据库行、HTTP Response、SDK 原生对象
4. **不暴露实现细节**：`find_by_id` 不叫 `select_by_primary_key`，`publish` 不叫 `kafka_produce`
5. **每个端口有至少两种可替换实现**：至少要有一个生产实现和一个 mock/stub 实现用于测试

### 8. 识别领域事件 (Domain Event)

领域事件表示已发生的业务事实，用于上下文间解耦通信。

**硬门禁 — 业务意图 → 事件 → 消息队列：**

每当价值流/设计中出现一个会改变系统事实或触发跨边界副作用的**业务意图**，必须定义对应领域事件，并由应用服务在意图成功后经 `EventBusPort` 投递到消息队列。禁止「有意图、无事件、同步直调副作用」。

| 步骤 | 产出 |
|------|------|
| 1. 列出业务意图 | 来自设计文档「意图 → 事件对照」或价值流增量 |
| 2. 为每个意图命名事件 | 过去式：`UserRegistered`, `TaskCompleted` |
| 3. 定义事件契约 | `domain/events/` 不可变数据类 |
| 4. 在聚合/应用服务挂载 | 状态变更后 `publish`；适配器写入 MQ |

细则：`.ai/08_prompt_management/01_intent_driven_development.md`。

**识别规则：**
- 以过去式命名（`UserRegistered`, `TaskCompleted`）
- 包含事件发生的时间戳
- 携带必要的数据（不是整个实体）
- **与业务意图一一对应（默认）**；复合意图可拆多个事件；纯查询意图可无事件（须标注）

**领域事件模板：**
```python
# domain/events/{event_name}.py
from dataclasses import dataclass
from datetime import datetime

@dataclass(frozen=True)
class EventName:
    """领域事件描述 — 对应业务意图: <意图简述>"""
    entity_id: int
    occurred_at: datetime
    # 事件携带的数据字段
```

### 9. 设计应用服务 (Application Service)

应用服务是领域层的出入口，负责编排领域服务和端口接口。它不包含业务逻辑，
只做流程编排和事务边界控制。

**应用服务模板：**
```python
# application/services/{service_name}.py
class ApplicationServiceName:
    """应用服务 —— 编排领域服务，不含业务逻辑

    这是依赖注入（DI）的组装点：
    所有端口实现（具体的适配器）在此注入到领域服务中。
    """

    def __init__(
        self,
        entity_repo: EntityNameRepository,      # 端口接口类型，不是具体实现
        event_bus: EventBusPort,                 # 端口接口类型
        notification: NotificationPort,          # 端口接口类型
        storage: FileStoragePort = None,         # 可选端口
    ):
        self._repo = entity_repo
        self._event_bus = event_bus
        self._notification = notification
        self._storage = storage
        # 用端口接口组装领域服务
        self._domain_service = DomainServiceName(
            dependency_port=entity_repo
        )

    def handle_create_entity(self, command: CreateEntityCommand) -> Entity:
        """处理创建实体命令

        编排流程：
        1. 调用领域服务执行业务逻辑
        2. 通过仓储端口持久化
        3. 通过事件总线端口发布领域事件
        4. 通过通知端口发送通知
        """
        # 1. 领域逻辑
        entity = self._domain_service.create(command)

        # 2. 持久化（通过端口，不关心底层是什么数据库）
        saved = self._repo.save(entity)

        # 3. 发布事件（通过端口，不关心底层是什么消息队列）
        self._event_bus.publish(entity.domain_events)

        # 4. 通知（通过端口，不关心底层是什么通知服务）
        self._notification.send(NotificationMessage(
            recipient=entity.owner_email,
            subject="Entity Created",
            body=f"Entity {entity.id} has been created.",
            channel="email"
        ))

        return saved
```

#### 依赖注入与组装 (Composition Root)

在应用入口（如 API handler、CLI command、cron job）中组装依赖：

```python
# composition_root.py 或直接在入口文件中

# 选择基础设施适配器（可通过配置切换）
if config.STORAGE_BACKEND == "s3":
    storage_adapter = S3StorageAdapter(bucket=config.S3_BUCKET)
elif config.STORAGE_BACKEND == "minio":
    storage_adapter = MinioStorageAdapter(endpoint=config.MINIO_ENDPOINT)
else:
    storage_adapter = LocalFileStorageAdapter(base_dir="/data/files")

if config.QUEUE_BACKEND == "kafka":
    event_bus_adapter = KafkaEventBusAdapter(brokers=config.KAFKA_BROKERS)
else:
    event_bus_adapter = RedisStreamEventBusAdapter(redis_url=config.REDIS_URL)

if config.DB_BACKEND == "postgresql":
    repo_adapter = PostgresEntityRepository(db_session)
else:
    repo_adapter = MongoEntityRepository(mongo_collection)

# 组装应用服务 —— 注入适配器
app_service = ApplicationServiceName(
    entity_repo=repo_adapter,
    event_bus=event_bus_adapter,
    notification=SendGridNotificationAdapter(api_key=config.SENDGRID_KEY),
    storage=storage_adapter,
)

# 处理请求
result = app_service.handle_create_entity(command)
```

**切换依赖库时，只需修改组装代码，领域层和应用层零改动。**

### 10. 生成领域模型文件

在对应模块下创建端口-适配器目录结构：

```
{module}/
├── domain/                          # 领域层（不依赖任何基础设施）
│   ├── __init__.py
│   ├── entities/                    # 实体
│   │   ├── __init__.py
│   │   └── {entity_name}.py
│   ├── value_objects/               # 值对象
│   │   ├── __init__.py
│   │   └── {vo_name}.py
│   ├── aggregates/                  # 聚合（可选，聚合根多的场景）
│   │   ├── __init__.py
│   │   └── {aggregate_name}.py
│   ├── services/                    # 领域服务
│   │   ├── __init__.py
│   │   └── {service_name}.py
│   ├── ports/                       # 端口接口（ABC 抽象）⭐ 依赖反转核心
│   │   ├── __init__.py
│   │   ├── repositories/            # 仓储端口
│   │   │   ├── __init__.py
│   │   │   └── {entity_name}_repository.py
│   │   ├── event_bus/               # 事件总线端口
│   │   │   ├── __init__.py
│   │   │   └── event_bus.py
│   │   ├── notifications/           # 通知端口
│   │   │   ├── __init__.py
│   │   │   └── notification.py
│   │   ├── storage/                 # 存储端口
│   │   │   ├── __init__.py
│   │   │   └── file_storage.py
│   │   ├── cache/                   # 缓存端口
│   │   │   ├── __init__.py
│   │   │   └── cache.py
│   │   └── identity/                # 身份端口
│   │       ├── __init__.py
│   │       └── identity.py
│   └── events/                      # 领域事件
│       ├── __init__.py
│       └── {event_name}.py
│
├── application/                     # 应用层（编排，不包含业务逻辑）
│   ├── __init__.py
│   └── services/
│       ├── __init__.py
│       └── {application_service_name}.py
│
├── infrastructure/                  # 基础设施层（适配器实现）
│   ├── __init__.py
│   ├── adapters/
│   │   ├── persistence/             # 持久化适配器
│   │   │   ├── __init__.py
│   │   │   ├── postgres_{entity}_repository.py
│   │   │   └── mongo_{entity}_repository.py
│   │   ├── messaging/               # 消息适配器
│   │   │   ├── __init__.py
│   │   │   ├── kafka_event_bus.py
│   │   │   └── redis_stream_event_bus.py
│   │   ├── notifications/           # 通知适配器
│   │   │   ├── __init__.py
│   │   │   ├── sendgrid_adapter.py
│   │   │   └── aws_ses_adapter.py
│   │   ├── storage/                 # 存储适配器
│   │   │   ├── __init__.py
│   │   │   ├── s3_storage.py
│   │   │   ├── minio_storage.py
│   │   │   └── local_file_storage.py
│   │   └── cache/                   # 缓存适配器
│   │       ├── __init__.py
│   │       ├── redis_cache.py
│   │       └── memory_cache.py
│   └── composition_root.py          # 依赖组装（DI 容器/工厂）
│
└── interfaces/                      # 接口层（API/CLI/Web）
    ├── __init__.py
    ├── api/
    │   └── {resource}_controller.py
    └── cli/
        └── {command}.py
```

**分层依赖规则（严格单向）：**

```
interfaces → application → domain ← infrastructure
     ↓            ↓           ↑           ↑
   API/CLI    编排服务    端口接口   适配器实现
```

- `interfaces` 依赖 `application`
- `application` 依赖 `domain`（端口接口）
- `infrastructure` 依赖 `domain`（实现端口接口）
- `domain` **不依赖任何层**
- `infrastructure` 之间不相互依赖（适配器 A 不知道适配器 B 的存在）

## 自检清单 (Hard Gate)

领域建模完成后，逐项确认：

### 领域纯净性
- [ ] 所有实体/值对象/端口接口/事件文件位于 `domain/` 目录下
- [ ] 领域层无任何 ORM 导入（`django.db.models`, `ForeignKey` 等）
- [ ] 领域层无任何外部服务 SDK 导入（Kafka, 云 SDK, HTTP 客户端等）
- [ ] 领域层无任何数据库字段声明（无 `null=True`, `max_length` 等持久化细节）

### 端口接口完整性
- [ ] 所有外部依赖都有对应的端口接口（ABC 抽象）
- [ ] 仓储端口接口只操作聚合根
- [ ] 端口接口方法返回领域对象，不返回基础设施原生类型
- [ ] 端口接口命名不暴露实现细节（`publish` 而非 `kafka_produce`）
- [ ] 每个端口接口有至少一种 mock/stub 适配器（用于测试）

### 依赖反转验证 ⭐
- [ ] **场景验证：切换数据库（如 PostgreSQL → MySQL）**，只需新增适配器实现仓储端口，领域层/应用层代码零改动
- [ ] **场景验证：切换消息队列（如 Kafka → Redis Streams）**，只需新增适配器实现事件总线端口，领域层/应用层代码零改动
- [ ] **场景验证：切换云服务商（如 AWS → 阿里云）**，只需新增对应适配器，领域层/应用层代码零改动
- [ ] 所有端口接口由领域层定义（`domain/ports/`），不由基础设施层定义
- [ ] 基础设施适配器文件导入 `domain.ports.xxx`，领域层不导入 `infrastructure`

### 领域模型质量
- [ ] 实体 ID 使用雪花算法或 UUID
- [ ] 领域事件以过去式命名
- [ ] **每个服务端业务意图都有对应领域事件契约，并经事件总线端口投递 MQ（或书面标注无事件例外）**
- [ ] 每个聚合根有对应的仓储端口
- [ ] 副作用命令/事件消费按 NFR「幂等性审视」表的键去重（禁止默认 `company_id`/`user_id`）
- [ ] 领域服务通过构造函数注入端口接口
- [ ] 文件名即类名（snake_case），一个文件一个类
- [ ] 应用服务不包含业务逻辑，只做编排

### 架构分层
- [ ] `domain/` 无 `import infrastructure` 或 `import application`
- [ ] `application/` 无 `import infrastructure`（只依赖 `domain`）
- [ ] `infrastructure/` 导入 `domain.ports` 并实现其接口
- [ ] 依赖注入在 `composition_root.py` 或入口文件中完成

**任何一项未通过 → 修复后再进入计划阶段。**

## 横切支持：审查与交付时的合规检查

当 `/9-review` 或 `/10-ship` 调用本技能时，执行合规审计模式：

```bash
python runAll/scripts/ci/check_ddd_bdd_compliance.py
```

此统一入口运行：
- Python DDD/BDD 合规：`task2app/scripts/ci/check_ddd_bdd_compliance.py`
- Go DDD 合规：`valueStream/scripts/ci/check_go_ddd_compliance.py`（针对 `valueStream/domain`）

审计要点：
- 领域文件是否导入了基础设施（ORM、Kafka、云 SDK）
- 聚合边界是否正确（无跨聚合直接调用）
- 事件流是否通过事件/接口解耦（非直接调用）
- 端口接口是否被正确实现（适配器实现了完整的 ABC 接口）
- 分层是否正确（领域层 → 应用层 → 基础设施层 → 接口层）
- **依赖反转是否成立**：适配器依赖端口接口，端口接口不依赖适配器

审查发现领域模型问题时的回退路径：

```
8-review 发现问题 → 重新调用 5-ddd 修正领域模型 → 更新 6-plans → 7-build 修复实现
```

## 与其他技能的衔接

| 上游 | 下游 |
|------|------|
| brainstorming 产出设计文档 | DDD 将设计转化为领域模型 |
| value-stream 产出价值增量 | DDD 只为当前增量建模 |
| nfr 产出质量等级和模型影响 | DDD 据此选择聚合粒度、一致性模型、架构模式 |

**DDD 完成后 → 调用 `/7-plans`。** 计划阶段使用领域模型（实体、端口接口、领域事件）作为制定任务清单的契约。

## 项目特定约束

- 关联在业务层维护，禁止 ForeignKey/OneToOne/ManyToMany
- 动态外键用双字段：`content_type_id` + `object_id`
- 数字类型 ID 使用雪花算法
- 遵循 GOF 23 种设计模式
- 函数式编程：一个文件一个类/函数
- 文件命名：`{职能}_{角色}` 模板

## 常见错误

| 错误 | 正确做法 |
|------|---------|
| 在实体中导入 Django Model | 实体是纯 Python 类 |
| 端口接口包含 SQL/ORM 代码 | 端口接口是 ABC，实现在基础设施适配器 |
| 领域服务直接调用外部 API | 通过端口接口注入，适配器实现在基础设施层 |
| 值对象有 setter | 值对象不可变 |
| 聚合过大包含太多实体 | 拆分为小聚合，通过 ID 引用 |
| 跳过领域事件 | 上下文间通信必须用事件解耦 |
| 在领域层导入 `requests`/`boto3` | 领域层无基础设施依赖，通过端口接口隔离 |
| 只在领域层定义仓储端口，忽略其他外部依赖 | 所有外部依赖（通知、存储、缓存、消息队列）都需端口接口 |
| 端口接口定义在基础设施层 | 端口接口属于领域层（`domain/ports/`），由领域定义"我要什么" |
| 应用服务包含业务逻辑 | 应用服务只做编排，业务逻辑在领域服务和实体中 |
| 适配器直接依赖另一个适配器 | 适配器之间不直接依赖，通过端口接口或领域事件通信 |
| 切换依赖库时需要改动领域层 | 正确的端口-适配器架构下，切换库只需新增适配器，领域层零改动 |
| 用 `company_id`/`user_id` 作事件消费幂等键 | 键须与单次业务意图同粒度（`event_id` / 业务单号 / `task_id`），见 NFR「幂等性审视」 |
| 实体方法 `increment()` 作为可重试命令 | 改为 `activate()` / `mark_as(status)` 状态转移 |

## 完成信号

领域模型文件和端口接口已生成，依赖反转验证通过。
此时可调用 `/7-plans` 开始制定实施计划。

> "领域模型已生成到 `{module}/domain/` 目录。端口接口（{列出端口类型}）已定义。依赖反转验证通过：切换任意基础设施实现时，领域层和应用层代码零改动。请审查领域模型和端口接口，确认后我们将进入计划阶段。"

## 完成后 — 下一步选择

领域模型生成并自检通过后，使用 `AskUserQuestion` 工具让用户一键选择下一步：

```
header: "下一步"
question: "领域建模已完成。下一步做什么？"
multiSelect: false
options:
  1. label: "实施计划 (推荐)"
     description: "将领域模型和价值流转为可验证的 checkbox 任务清单"
  2. label: "重新 DDD 建模"
     description: "调整聚合边界、限界上下文、端口接口或领域事件"
```

- 用户选 1 → 调用 `/7-plans`
- 用户选 2 → 重新执行本技能（DDD 建模）
