# [运行时] 导航栏任务搜索：粘贴云机器名 `task-task_<id>` 搜不到

## 基本信息

- 版本：1.1.0
- 案例编号：FE-20260722-0076
- 录入日期：2026-07-22
- 最后更新：2026-07-22
- 录入人：Cursor Agent

## 现象

- 页面：租户导航栏任务搜索（`input#navbar-task-search-input`）
- 从阿里云控制台复制机器名称 `task-task_13838391081043321882` →「无匹配任务」
- 输入后 6 位 `321882` → 能搜到同一任务

## 环境与上下文

- 前端：`NavbarTaskSearch.vue` → `GET /api/tenant/{tid}/tasks/search/?q=`
- 任务主键：`genID("task")` → `task_<digits>`
- 云机器名：历史 `RunInstances.InstanceName = "task-" + task_id` → 叠成 `task-task_<digits>`
- 对账：`taskCloudService` orphan reconcile 按 InstanceName DescribeInstances

## 根因

1. **命名叠前缀**：id 已是 `task_*`，再拼 `task-` 得到控制台可见的 `task-task_*`。
2. 搜索对 `q` 做 id/title LIKE；`task-task_*` 不是库内 id 子串 → 0 命中；短数字仍是后缀故能命中。

## 修复

1. **搜索归一化**（兼容已存在机器名）：`task-task_` → `task_`（前后端）。
2. **命名源头调整**（新开机器）：`InstanceName = task_id`（不再拼 `task-`）。
   - `taskEvents/internal/cloud/aliyun/instance_name.go`
   - Django `start_vm.py` 同步
   - orphan 对账按 canonical + legacy 双名 Describe

## 验收

```bash
cd taskTaskService/src && go test -count=1 -run 'TestNormalizeTaskSearchQuery|TestSearchTasksAcceptsServicePrefixedTaskID' .
cd taskEvents && go test -count=1 ./internal/cloud/aliyun/ -run 'TestECSInstanceName'
cd taskCloudService/src && go test -count=1 -run 'TestReconcileOrphan|TestECSInstanceNames' .
# 新启动机器：控制台 InstanceName 应等于 task_id；旧机器名仍可被搜索归一化命中
```

## 预防

- ECS InstanceName **必须等于** 规范 `task_id`；禁止再拼服务名或 `task-` 前缀。
- 反查/对账须兼容 legacy `task-{task_id}`，直至旧实例淘汰。
- 同类问题：评论容器名 `task_<id>_cmt_<cmt>` 见 `95_navbar_task_search_comment_container_miss.md`。
