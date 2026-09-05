# Step 4 — Value Stream：访问管理

```
管理员进入人员管理
  → 打开「访问管理」
  → 选择成员/小组
  → 勾选菜单（映射权限码）
  → 保存
      → taskAuth 创建/更新自定义角色 (+ RoleChanged)
      → taskTenant 绑定 member/group role (+ TenantRoleChanged + membership_rev)
      → 目标用户下次请求 PDP 重算 X-Tenant-Perms
  → 目标用户侧栏/页面按新权限码收敛
```

测试点：TI-1…TI-6（见 intents）。
