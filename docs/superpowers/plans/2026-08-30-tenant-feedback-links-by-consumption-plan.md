# Implementation Plan: 意见与建议链接

> Design / permission / value stream / NFR / DDD as of 2026-08-30

## Task 1: Domain visibility + HTTPS

- [ ] `feedback_domain.go` + `feedback_domain_test.go`：空阈值可见、99 vs ≥100、AND、嵌套
- [ ] `feedback_https.go` + test：https ok；http/javascript/relative 拒绝

## Task 2: DDL

- [ ] `dataMigrate/taskBill/074_feedback_link_groups.sql` 四表 + 幂等表 + 四种 kind seed
- [ ] `dataMigrate/taskAuth/052_nav_feedback_page.sql` page/region/members + 三角色绑定 + `feedback:view`

## Task 3: Store + consumption + handlers

- [ ] `feedback_store.go` CRUD 整组替换
- [ ] `feedback_consumption.go` 四种投影；缺数据=0
- [ ] `feedback_handlers.go` 超管 + 租户；RequireRegionView；IsPlatformStaff
- [ ] `events.go` 三 topic；POST/PUT/DELETE 发布
- [ ] `handlers.go` 挂路由
- [ ] `openapi.yaml` 同步
- [ ] `feedback_handlers_test.go` T1–T12

## Task 4: Frontend

- [ ] `tenantConsoleNav.js` + test：`nav.feedback`
- [ ] `TenantConsoleFeedbackNav.vue` + unit tests F1–F4、F6
- [ ] `Sidebar.vue` 引用组件（行数 ≤500）
- [ ] `SystemAdminFeedbackLinks.vue` + test F5；sidebar + `adminRoutes.js`

## Task 5: Review / ship

- [ ] 单测绿；commit submodule 再 meta；push origin main；登记精准重启 task-bill + taskFE
