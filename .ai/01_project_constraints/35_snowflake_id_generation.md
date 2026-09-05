# 数据库 ID 字段雪花算法生成规范（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-26
- 最后修改：2026-07-26
- 维护者：Trae AI 团队

## 背景

本 monorepo 为分布式多服务架构，各服务独立部署并拥有各自数据库。传统自增主键在分布式环境下存在 ID 冲突、中心化依赖、无法离线生成等问题。**Snowflake（雪花算法）** 是 Twitter 开源的分布式唯一 ID 生成算法，能在无中心协调节点的前提下，生成全局唯一、趋势递增的 64 位整数 ID。

本仓库已在 Python（Django）和 Go 两侧落地了统一的 Snowflake 实现，所有**新建数据库表的主键 ID 字段**必须采用雪花算法生成。

## 核心规则（必须遵守）

### 1. 适用范围

- **所有新建数据库表**的主键 `id` 字段，无论所属服务（Django / Go / 其他）
- **所有新建服务**中的 ID 生成逻辑
- **存量表迁移**：对已有自增主键的表，在重构/迁移时**应**切换为 Snowflake ID；若暂不切换须在设计文档中说明原因与切换计划

### 2. 强制使用 Snowflake 的场景

| 场景 | 要求 | 说明 |
|------|------|------|
| 新建表主键 `id` | **必须** Snowflake | 禁止使用数据库自增（AUTO_INCREMENT / SERIAL / BIGSERIAL） |
| 新建服务 ID 生成 | **必须** 调用统一实现 | 禁止自行实现 Snowflake 变体或引入第三方库 |
| 外键关联字段 `*_id` | **必须** 为 `bigint`，存储 Snowflake ID 值 | 禁止外键字段使用 `varchar` 存储 UUID |
| 存量表迁移 | **应当** 切换 | 属于「迁移/重构」范畴，触发测试先行规则（见第 31 条） |
| 中间表 / 关联表 | **必须** Snowflake | 即使无独立实体语义，也须有 Snowflake 主键 |

### 3. 统一算法参数

所有语言的实现**必须**采用以下统一参数，确保跨服务 ID 可直接比较与关联：

| 参数 | 值 | 说明 |
|------|-----|------|
| 起始纪元 (epoch) | `1577836800000` | 2020-01-01 00:00:00 UTC（毫秒） |
| 时间戳位数 | 41 位 | 毫秒级，可用约 69 年 |
| 机器 ID 位数 | 10 位 | 支持最多 1024 个节点 |
| 序列号位数 | 12 位 | 每毫秒最多 4096 个 ID |
| 位排列 | `timestamp << 22 \| machine << 12 \| sequence` | 时间戳高位 → 机器ID中位 → 序列号低位 |
| 机器 ID 来源 | 环境变量 `MACHINE_ID` | 未设置时使用默认值（Python: 随机，Go: `7`）；部署时须为每个实例分配唯一 `MACHINE_ID` |

### 4. 统一实现

#### Python（Django / 侧车）

**唯一实现**：`task2app/Saas_project/core/utils/snowflake.py` → `SnowflakeGenerator`

```python
from core.utils.snowflake import generate_snowflake_id

# Model 保存前生成 ID
class Order(models.Model):
    id = models.BigIntegerField(primary_key=True)

    def save(self, *args, **kwargs):
        if not self.id:
            self.id = generate_snowflake_id()
        super().save(*args, **kwargs)
```

- 禁止各 app 复制或重新实现 Snowflake 生成器
- 禁止使用 Python `uuid` 模块或 `random` 生成 ID
- Provider 侧实现 `task2app/Saas_Ai_Provider/apps/marketplace/utils/snowflake.py` 应与核心实现保持一致；若发现分歧须以核心实现为准修正

#### Go

**唯一实现**（OPT-20260726-018）：`shareLib/snowflake/snowflake.go`（包级 `GenerateID()` / `GenerateIDString()`）

```go
import "snowflake"

func CreateOrder(req CreateOrderRequest) (*Order, error) {
    id := snowflake.GenerateID()         // int64
    idStr := snowflake.GenerateIDString() // string
    // ...
}
```

- **所有 Go 服务必须引用 `shareLib/snowflake/` 统一实现**，禁止内嵌或复制本地 snowflake 文件
- 已迁移服务：`taskEvents`、`taskTenantService`、`taskCloudService` — 已删除本地 snowflake 实现，统一 import `"snowflake"`
- 新 Go 服务须在 `go.mod` 添加 `replace snowflake => ../shareLib/snowflake` 并直接 import

### 5. 与 ID 字符串传输规范的配合

本规则与 [ID 字段字符串传输规范](../03_technical_implementation/11_id_field_string_transit.md) **联合生效**：

- **本规则** 规定 ID **如何生成**（Snowflake 算法 → `int64`）
- **传输规范** 规定 ID **如何传递**（进程内与进程间一律 `string`，仅入库时转 `int64`）

两者不矛盾：`snowflake.GenerateID()` 返回 `int64`，在 Repository 边界立即转为 `string` 向上层返回；写库时在 ORM/SQL 绑定参数紧前一步转回 `int64`。

## 禁止事项

- 新建表使用数据库自增主键（`AUTO_INCREMENT` / `SERIAL` / `BIGSERIAL` / `IDENTITY`）
- 使用 UUID（`uuid.uuid4()` / `uuid_generate_v4()` / `gen_random_uuid()`）作为主键
- 使用 `random.randint()` 或 `Math.random()` 生成 ID
- 自行实现 Snowflake 变体而不复用项目统一实现
- 引入第三方 Snowflake 库（如 `pysnowflake`、`bwmarrin/snowflake`）替代项目统一实现
- 修改 epoch 或其他算法参数而不更新本文档

## 例外

以下场景可豁免本规则，但须在设计文档或代码注释中说明原因：

1. **第三方系统 ID 映射表**：存储外部系统（如阿里云、GitLab）返回的 ID 时，直接使用对方 ID 类型
2. **Django 内置表**：`django_migrations`、`django_content_type` 等框架自管表不受此限
3. **临时表 / 缓存表**：生命周期短（< 1 天）且不参与跨服务关联的表
4. **合规或性能硬需求**：经架构评审确认的特殊场景

## 与相关规则的关系

| 规则 | 关系 | 说明 |
|------|------|------|
| [ID 字段字符串传输规范](../03_technical_implementation/11_id_field_string_transit.md) | **联合生效** | 本规则管生成，传输规范管传递 |
| [外键禁止与业务层关联](../03_technical_implementation/06_style_guide.md) | 配合 | 外键 ID 字段亦为 Snowflake ID |
| [动态外键双字段规范](../03_technical_implementation/06_style_guide.md) | 配合 | 双字段中的 ID 段须为 Snowflake ID |
| [新增服务/接口优先落 Go](./20_go_service_first_apis.md) | 配合 | 新 Go 服务须 import `snowflake`（`shareLib/snowflake/`） |
| [迁移/重构——测试先行](./30_migration_refactoring_test_first.md) | 触发 | 存量表切换 Snowflake 时触发测试先行规则 |

## 部署要求

- 每个服务实例必须配置唯一的 `MACHINE_ID` 环境变量（0–1023）
- Kubernetes 部署可通过 StatefulSet 副本序号或 Downward API 注入
- Docker Compose 部署须在 `environment` 或 `env_file` 中显式指定
- `runAll` 启动的本机服务默认使用固定 `MACHINE_ID=7`（开发环境可接受）

## 变更日志

- 2026-07-26：版本 1.0.0 - 初版：确立 Snowflake 为数据库 ID 字段唯一生成算法，定义统一参数与实现引用
