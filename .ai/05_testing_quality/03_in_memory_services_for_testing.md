# 测试时使用内存实现（依赖注入规范）

## 基本信息
- 版本：1.0.0
- 创建日期：2026-03-19
- 最后修改：2026-03-19
- 维护者：Trae AI 团队

## 规则分类

### 核心规则

#### 外部依赖动态注入与测试时内存实现
- **描述**：后端对 Kafka、Redis、数据库、邮件、云厂商 SDK 等外部依赖应采用**依赖注入**方式，支持在运行测试时切换为内存/Mock 实现，以减少测试对外部服务的依赖、缩短测试时间、便于 CI 环境执行
- **适用场景**：所有涉及外部依赖的后端开发、单元测试编写、测试环境配置
- **优先级**：高
- **规则**：
  - **抽象接口隔离**：将 Kafka、Redis、邮件、云厂商等基础设施封装为抽象接口（如 `IEventPublisher`、`IPubSubClient`、`IEmailSender` 等），业务层通过接口调用，不直接依赖具体实现
  - **服务注册表**：通过 `core/services/registry` 等注册表根据配置返回真实实现或内存实现；通过 `USE_IN_MEMORY_SERVICES`（环境变量或 `settings.USE_IN_MEMORY_SERVICES`）控制切换
  - **测试配置**：使用 `saas_project.settings_test` 或 `DJANGO_SETTINGS_MODULE=saas_project.settings_test` 运行测试时，自动启用内存实现；数据库使用 `:memory:` SQLite
  - **邮件**：使用 `core.services.email_sender.send_mail` 替代 `django.core.mail.send_mail`，测试时使用 `InMemoryEmailSender`
  - **云厂商 SDK**：通过 `cloud.providers.factory.get_provider` 获取云提供商；测试时返回 `MockCloudProvider`，不调用真实阿里云等 API
  - **事件发布**：使用 `core.kafka.send_event` / `send_message`，测试时使用 `InMemoryEventPublisher`
  - **Pub/Sub**：使用 `core.redis.redis_client` 的代理或 `get_pubsub_client()`，测试时使用 `InMemoryPubSubClient`
  - **断言能力**：内存实现应提供 `get_events()`、`get_sent_emails()` 等方法，便于测试中断言已发送的事件或邮件
- **参考文档**：`Saas_project/docs/testing/IN_MEMORY_SERVICES.md`

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 > 最佳实践 > 风格指南
  2. 文件级规则 > 目录级规则 > 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志
- 2026-03-19：版本 1.0.0 - 新增测试时使用内存实现（依赖注入）规范
