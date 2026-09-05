# 阿里云SDK使用规范

## 基本信息
- 版本：1.1.0
- 创建日期：2026-03-05
- 最后修改：2026-06-01
- 维护者：Trae AI 团队

## SDK 源码位置（monorepo）

阿里云 Python SDK 已迁入 monorepo，**不再**使用 `~/alibabacloud-python-sdk`。

| 用途 | paths.conf 键 | 相对 REPO_ROOT（task2app） | 绝对路径示例 |
|------|---------------|---------------------------|--------------|
| SDK 根目录 | `SDK_DIR` | `../sdk` | `…/ram-mount/sdk` |
| Python SDK 合集 | `SDK_ALIYUN_PYTHON` | `../sdk/alibabacloud-python-sdk` | 原 home 目录内容 |
| **ECS（云服务器）** Python 包 | `SDK_ECS_PYTHON` | `../sdk/alibabacloud-python-sdk/ecs-20140526` | RunInstances / DescribeInstances 等 |
| ECS Go SDK（参考） | `SDK_ECS_GO` | `../sdk/ecs-20140526` | Go 侧云 API 范例 |

路径真源：`task2app/paths.conf`；代码中通过 `core.paths_loader.get('SDK_ECS_PYTHON')` 获取，**禁止**硬编码 `~/alibabacloud-python-sdk`。

### 本地 editable 安装（开发环境）

在 `Saas_project` 虚拟环境中，从 paths.conf 解析出的绝对路径安装：

```bash
cd Saas_project
pip install -e "$(python -c "from core.paths_loader import get; print(get('SDK_ECS_PYTHON'))")"
# 常用依赖（若 requirements 未锁定）
pip install alibabacloud-tea-openapi alibabacloud-credentials alibabacloud-tea-util
```

等价于（本机 monorepo 布局下）：

```bash
pip install -e ../sdk/alibabacloud-python-sdk/ecs-20140526
```

### 调用云服务器 API 时的参考范例

编写或调试 **ECS 启停、镜像、交换机、安全组** 等逻辑时，按以下顺序查阅：

1. **本仓库业务实现**：`Saas_project/cloud/providers/aliyun/`（如 `instance/start_vm.py`、`network/vswitch.py`）
2. **Python SDK 包内 README**：`sdk/alibabacloud-python-sdk/ecs-20140526/README-CN.md`
3. **SDK 通用用法**：`sdk/alibabacloud-python-sdk/docs/0-Usage-CN.md`
4. **调试工具**：`Saas_project/cloud/providers/aliyun/debug.py` 的 `debug_aliyun_request`（见 `03_cloud_sdk_specifications.md`）

方法命名约定：新版 Tea SDK 使用 `{action}_with_options`（例如 `create_vswitch_with_options`），以 `SDK_ECS_PYTHON` 目录下 `alibabacloud_ecs20140526/client.py` 为准。

## 规则分类

### 核心规则（一级分类）
> 影响代码质量和安全性的关键规则，必须严格遵守

#### 阿里云SDK使用规范（二级分类）
##### 依赖管理规则
- 描述：使用阿里云 SDK 时，必须从 monorepo `sdk/` 目录（paths.conf → `SDK_ALIYUN_PYTHON` / `SDK_ECS_PYTHON`）安装依赖，确保使用仓库内指定版本
- 适用场景：所有使用阿里云 SDK 的项目（含云服务器 ECS、短信 dysmsapi 等）
- 优先级：高
- 规则：
  - 安装依赖时使用本地目录 editable 安装，例如 `pip install -e <SDK_ECS_PYTHON 绝对路径>`
  - 避免从公共 PyPI 随意升级 major 版本；若 PyPI 安装，须与 `sdk/` 内版本对齐并记录在 requirements
  - 在 requirements 或文档中注明 SDK 来源为 monorepo `sdk/alibabacloud-python-sdk`
  - **已废弃**：`~/alibabacloud-python-sdk`、`task2app/alibabacloud-python-sdk/` 软链或副本

##### 客户端初始化规范
- 描述：统一阿里云SDK客户端的初始化方式，确保配置正确且安全
- 适用场景：所有初始化阿里云SDK客户端的代码
- 优先级：高
- 规则：
  - 使用 Config 类进行客户端配置，包含必要的参数：access_key_id、access_key_secret、region_id
  - 访问密钥必须通过环境变量或配置文件获取，禁止硬编码在代码中
  - 根据业务需要设置合适的超时时间和重试策略
  - 参考示例：
    ```python
    from alibabacloud_tea_openapi.models import Config
    from alibabacloud_ecs20140526.client import Client
    
    config = Config(
        access_key_id=os.environ.get('ALIBABA_CLOUD_ACCESS_KEY_ID'),
        access_key_secret=os.environ.get('ALIBABA_CLOUD_ACCESS_KEY_SECRET'),
        region_id='cn-hangzhou'
    )
    client = Client(config)
    ```

##### 日志记录规范
- 描述：调用阿里云SDK时必须记录相关信息，便于问题排查和审计
- 适用场景：所有调用阿里云SDK的代码
- 优先级：高
- 规则：
  - 记录业务Action、AccessKey和RequestId到日志中
  - 使用结构化日志格式，包含时间戳、操作类型、状态等信息
  - 日志级别根据操作重要性设置，一般操作使用INFO级别，错误使用ERROR级别

##### 响应处理规范
- 描述：统一处理阿里云SDK的响应，确保数据格式一致且符合前端要求
- 适用场景：所有处理阿里云SDK响应的代码
- 优先级：高
- 规则：
  - 对于SDK方法调用的返回，直接转为JSON格式传到前端
  - 将SDK返回的RequestId放入HTTP响应头中返回
  - 提取响应中的核心数据，避免返回原始响应对象
  - 处理响应中的错误码和错误信息，转换为统一的错误格式

### 最佳实践（一级分类）
> 提升开发效率和代码可维护性的建议

#### 阿里云SDK使用规范（二级分类）
##### 错误处理最佳实践
- 描述：合理处理阿里云SDK的错误，确保系统稳定性
- 适用场景：所有调用阿里云SDK的代码
- 优先级：中
- 规则：
  - 使用try-except捕获SDK可能抛出的异常
  - 对不同类型的错误进行分类处理，如网络错误、认证错误、业务错误等
  - 实现适当的重试机制，特别是对于网络临时故障
  - 记录详细的错误信息，便于问题排查

##### 性能优化建议
- 描述：优化阿里云SDK的使用，提高性能和可靠性
- 适用场景：所有调用阿里云SDK的代码
- 优先级：中
- 规则：
  - 对于频繁调用的API，考虑使用连接池和缓存
  - 合理设置超时时间，避免请求长时间阻塞
  - 对于大数据量操作，使用批量API或异步操作
  - 监控SDK调用的性能指标，如响应时间、成功率等

##### 代码组织建议
- 描述：合理组织阿里云SDK相关代码，提高可维护性
- 适用场景：所有使用阿里云SDK的项目
- 优先级：中
- 规则：
  - 将SDK客户端初始化逻辑封装为单例或工厂方法
  - 按功能模块组织SDK相关代码，如ecs.py、oss.py等
  - 抽象通用的SDK调用逻辑，减少重复代码
  - 编写单元测试，确保SDK调用的正确性
  - 新增云 API 调用前，先在 `SDK_ECS_PYTHON` 源码中确认 Request/Client 方法名

### 风格指南（一级分类）
> 统一代码风格和格式的规范

#### 阿里云SDK使用规范（二级分类）
##### 命名规范
- 描述：统一阿里云SDK相关代码的命名风格
- 适用场景：所有使用阿里云SDK的代码
- 优先级：低
- 规则：
  - 客户端实例命名使用 `{service}_client` 格式，如 `ecs_client`、`oss_client`
  - 配置对象命名使用 `{service}_config` 格式，如 `ecs_config`
  - 请求对象命名使用 `{operation}_request` 格式，如 `describe_instances_request`
  - 响应对象命名使用 `{operation}_response` 格式，如 `describe_instances_response`

##### 代码格式规范
- 描述：统一阿里云SDK相关代码的格式风格
- 适用场景：所有使用阿里云SDK的代码
- 优先级：低
- 规则：
  - 遵循项目的代码格式规范，如PEP 8
  - 合理使用空白行和缩进，提高代码可读性
  - 为复杂的SDK调用添加注释，说明参数含义和业务逻辑；注释中引用 SDK 时使用 `SDK_ECS_PYTHON` 相对 monorepo 路径，勿写 `~/alibabacloud-python-sdk`
  - 保持代码行长度适中，避免过长的行

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 > 最佳实践 > 风格指南
  2. 文件级规则 > 目录级规则 > 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志
- 2026-06-01：版本 1.1.0 - SDK 迁入 monorepo `sdk/`；paths.conf 新增 SDK_* 键；废弃 ~/alibabacloud-python-sdk
- 2026-03-05：版本 1.0.0 - 初始版本，添加阿里云SDK使用规范
