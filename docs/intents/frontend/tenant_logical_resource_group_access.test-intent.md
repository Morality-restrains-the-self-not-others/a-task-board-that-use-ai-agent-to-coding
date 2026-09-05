# 测试意图：逻辑资源组访问管理（v72）

## TI-1 侧栏入口

- Given 公司租户且当前用户可管理访问（如 `member:manage` 或管理 region）
- When 打开人员管理子菜单
- Then 可见「访问管理」并导航至 `/people/access/`

## TI-2 配置树语义

- Given 系统已种子化 page 与子 ui_region
- When 打开访问管理配置面板
- Then 展示 **page → region** 树；勾选默认落在 **region**（A1）；整页勾选可写 page 行

## TI-3 保存后 PDP

- Given 管理员为成员授予 `region:people.access.edit`
- When 该成员刷新权限
- Then PDP 含 `region:people.access.edit`；若其所属 page 下至少一 region 命中则含对应 `page:*`
- And **不含**因本授予而展开的旧粗码（如 `project:view`）（B2）

## TI-4 后端 Enforce

- Given 用户无目标 region
- When 调用挂载 `RequireRegion(该 region)` 的 API
- Then 返回 403；与前端隐藏无关

## TI-5 侧栏 page 过滤

- Given 用户无任何 `page:nav.image_market` 下 region
- When 渲染侧栏
- Then 「镜像市场」不渲染

## TI-6 tenant_admin

- Given 用户角色为 `tenant_admin`
- When 拉取权限
- Then 拥有全部系统 page/region（设计约定）
- And 在访问管理选中该主体时，右侧勾选为目录全部 page + ui_region（与 PDP 全权限一致展示）

## TI-7 与业务资源组隔离

- Given 存在 `tenant_resource_group_assignment` 业务绑定
- When 配置访问管理
- Then UI/API **不**读写该业务表作页面 ACL
