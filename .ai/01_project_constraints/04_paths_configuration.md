# 路径配置规范

## 基本信息
- 版本：1.0.1
- 创建日期：2026-03-19
- 最后修改：2026-07-22
- 维护者：Trae AI 团队

> **关联**：`paths.conf` 管业务路径；各服务 **`conf/<app>/` YAML 的读取边界**见 [29_service_own_conf_directory_only_via_sync.md](./29_service_own_conf_directory_only_via_sync.md)（禁止运行时直读他服务 conf）；**人工可改运行时/编排参数的编辑落点**见 [47_conf_app_human_editable_config_ssot.md](./47_conf_app_human_editable_config_ssot.md)。

## 规则分类

### 核心规则（一级分类）
> 路径配置的统一管理，避免多模块分散定义导致的不一致

#### 路径配置集中化（二级分类）

##### paths.conf 单一配置源
- **描述**：所有路径的设置必须仅集中在项目根目录的一个 `paths.conf` 文件中进行约定。该文件作为项目路径的唯一配置源，其他模块不得在自身代码中硬编码或独立推导路径。配置格式为：**1 个项目根目录（REPO_ROOT，绝对路径）+ N 个相对于根目录的路径**。
- **适用场景**：所有涉及文件路径、目录路径的配置，包括但不限于：日志目录、静态资源目录、数据库文件路径、配置文件路径、临时文件目录等
- **优先级**：高
- **规则**：
  - REPO_ROOT 为项目根目录，须使用绝对路径；迁移或克隆项目时仅需修改此处
  - 其余路径使用相对于 REPO_ROOT 的路径，paths_loader 会自动解析为绝对路径
  - 所有需引用路径的模块，必须从 `paths.conf` 或项目提供的路径加载模块获取路径
  - 禁止在各模块中使用 `os.path.dirname(__file__)`、`Path(__file__).parent` 等方式自行推导与配置相关的路径
  - `paths.conf` 位于项目根目录，即与 `Saas_project`、`front_project` 等子项目同级
  - 完整的路径定义列表请查看项目根目录的 `paths.conf` 文件
  - **monorepo SDK**：`SDK_DIR`、`SDK_ALIYUN_PYTHON`、`SDK_ECS_PYTHON` 等键指向 `task2app` 同级的 `sdk/` 目录（见 `07_aliyun_sdk_usage.md`）

##### 路径获取方式
- **描述**：模块需要路径时，应通过统一的路径加载接口获取，不得重复实现路径解析逻辑
- **适用场景**：Django 配置、日志配置、脚本初始化、服务启动等所有需要路径的场景
- **优先级**：高
- **规则**：
  - 使用 `Saas_project/core/paths_loader` 模块获取路径，例如：`from core.paths_loader import get, get_path, log_dir, repo_root`
  - 加载模块负责解析 `paths.conf` 并导出所需路径变量
  - 路径加载应支持在项目根目录下通过查找 `paths.conf` 自动定位
  - `paths.conf` 使用 INI 格式，`[paths]` 段下：`REPO_ROOT` 为绝对路径，其余键值为相对于 `REPO_ROOT` 的路径

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  - 本规范 > 各模块原有路径推导逻辑
  - 统一迁移时，优先保证 `paths.conf` 作为唯一配置源

## 变更日志
- 2026-08-12：关联元规则 47（人工可改配置 SSOT 在 conf/<app>/）
- 2026-06-01：补充 monorepo `sdk/` 路径键说明（与 `07_aliyun_sdk_usage.md` 对齐）
- 2026-03-19：版本 1.0.0 - 新增路径配置规范，要求所有路径集中在 paths.conf
