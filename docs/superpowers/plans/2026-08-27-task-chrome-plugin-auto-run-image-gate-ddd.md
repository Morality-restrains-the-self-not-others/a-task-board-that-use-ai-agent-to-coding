# DDD：Chrome 插件自动运行与已安装镜像挂钩

- **Date:** 2026-08-27
- **Bounded context:** taskChromePlugin（前端扩展，非后端 BC）

## 模型

| 类型 | 名称 | 说明 |
|------|------|------|
| 既有 VO | `ProjectAutoRunCapability` | `projectAllowsAutoRun(project)` |
| 既有 VO | `AutoRunControlState` | `{ enabled, checked, hint }`；本增量增加前置 `hasInstalledImage` |
| 新增 VO | `ImageFieldAppearance` | `{ requiredMarkVisible, requiredMarkText, emptyOptionLabel }` |
| 领域服务 | `resolveAutoRunControlState` | 项目 + 镜像是否已选 + 勾选偏好 |
| 领域服务 | `resolveImageFieldAppearance` | 项目是否允许自动运行 → 镜像标签/空选项 |
| 领域服务 | `validateAutoRunRequiresImage` | `auto_run` 且无镜像 → 提交拦截文案 |
| 既有不变量 | 服务端 `AUTO_RUN_IMAGE_REQUIRED` | 客户端门禁不得弱于该不变量 |

## 事件

| 业务意图 | 领域事件 | MQ | 例外 |
|----------|----------|-----|------|
| UI 挂钩镜像与自动运行 | （无） | — | **书面例外**：纯客户端；创建任务仍走既有 TASK 创建事件 |

## 端口

无新仓储。出站仍为既有 GET 镜像 + POST 创建。
