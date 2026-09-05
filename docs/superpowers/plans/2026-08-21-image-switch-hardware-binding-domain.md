# 切换镜像 × 硬件硬拦截 — 领域模型

## Bounded Contexts

- **Project Configuration**（taskProjectService）：项目默认镜像与运行模版
- **Cloud Provisioning**（taskCloudService）：start-vm 宿主机镜像解析
- **Task Collaboration**（taskFE + taskTaskService）：任务默认镜像、评论启动

## Aggregates

- **Project**（根）：`container_image_id` + `server_run_template`
  - 不变量：变更非空镜像时，生效模版完整且实例架构 ∈ 镜像架构集合
  - 清空镜像或清空模版 `{}`：允许（未设置）
- **InstalledContainerImage**（cloud 所有）：`id`, `tenant_id`, `target_architectures`
- **StartVmRequest**：实例类型 vs runtime env architecture（既有）

## Value Objects

- `CpuArchitecture`：`x86_64` | `arm64`
- `InstanceType`：云厂商规格字符串；`inferInstanceArchitecture` 映射到 VO

## Domain Services

- `EnsureImageTemplateCompatible(imageID, template, lookup)` → error | ok
- 无新领域事件（同步配置写 / 校验失败不改变事实）

## Ports

- `InstalledImageLookup`：已有 cloud internal lookup（适配器 `cloud_client.go`）

## 事件对照例外

见设计文档表：无新 MQ 事件。
