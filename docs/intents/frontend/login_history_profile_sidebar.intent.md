# 功能意图：账号中心侧边栏可查看登录历史

## 背景与目标

用户在个人资料侧边栏进入「登录历史」，看到每次成功登录的时间、入口、IP。超管在用户列表可打开同一数据结构的管理页。

## 范围与边界

- 范围内：`UserCenterSidebar` 导航、用户列表页、超管用户行真实链接、空态/错误 `data-traceId`。
- 范围外：地图、设备管理、强制下线。

## 验收标准

1. 侧边栏有「登录历史」，`activeMenu=login-history` 高亮。
2. 页面请求 `GET /api/auth/login-history/`，展示 IP 与入口标签。
3. 空列表文案「暂无登录记录」。
4. 请求失败节点带 `data-traceId`。
5. 超管用户行「登录历史」为真实 `href`，禁止 `@click.prevent` + `router.push`。
