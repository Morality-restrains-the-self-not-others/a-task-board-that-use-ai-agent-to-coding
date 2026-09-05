# 任务详情陈旧仓库地址与 relay 启动设计

日期：2026-05-27  
状态：已实现

## 问题

任务 `TaskProject.repo_address` 保存创建时的仓库 URL（如 `http://localhost:8012/example-user/somanyad`），项目 `ProjectRepo` 已更新为当前 URL（如 `http://localhost:8012/ljy/somanyad`）。关联项目面板展示项目当前地址，但 relay 直接启动后 `onlineServiceJS` 克隆仍使用陈旧地址，导致 git exit 128。

## 方案

### 后端

1. 新增 `projects/repo_url_resolution.py`：按仓库名解析陈旧 URL，容器 `task-detail` / `repo-clone-credentials` 统一返回 `ProjectRepo` 当前 URL。
2. `TodoSerializer._projects_api_representation` 增加 `stored_repo_address`、`project_repo_url`、`repo_address_mismatch` 供前端检测。

### 前端

1. 「直接启动」面板展示不一致横幅，默认禁用「启动」。
2. 用户可选「更新任务仓库地址」（PATCH projects）或「不更新，继续启动」。
3. 关联项目卡片显示「仓库地址已变更」徽章。

## 测试

- `test_fetch_container_task_detail_resolves_stale_task_project_repo_address`
- `TaskDetail.relay-to-trae-stale-repo-address.playwright.test.js`
