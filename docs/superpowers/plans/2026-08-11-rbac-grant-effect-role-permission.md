# Role-Permission — 资源组授予效果 v73

## 变更角色/权限面

| 主体 | 能力 | 门禁 |
|------|------|------|
| 租户管理员 | 全部 page/region operate | PDP 特判 |
| 自定义访问角色 | 按 grants effect | auth_role_resource_group.effect |
| 仅 view 主体 | 可见区块/读 API | HasRegionView |
| operate 主体 | 写按钮/写 API | HasRegionOperate |
| 访问管理保存者 | PUT resource-groups | people.access.save_actions **operate** |

## 审计结论

- 写路径已改为 `HasRegionOperate`（save_actions）
- 读目录仍 `HasRegionView`
- 无平行 ACL
