# 测试意图：Workbench 链接读取当前物理机 CSC

## 测试目标

验证 `GET compute/workbench-link` 与 runtime-status 使用同一套 CSC 解析：评论级实例可打开 Workbench；仅缺任务级 instance_id/region 不再误报。

## 测试分层

- 单元：`taskCloudService/src/compute_workbench_link_test.go`（httptest + SQLite 夹具）
- 前端单元：Workbench 失败路径写入 `serverRuntimeStatusTraceId`（`useServerConfigRuntimeFetch.test.js`）
- 无新增 MQ 断言（只读，见功能意图例外）

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 任务级 CSC instance_id/region 空；评论级 aliyun + i-cmt + cn-hangzhou | GET workbench-link?task_id= | 200，url 含 instanceId=i-cmt 与 regionId=cn-hangzhou |
| T2 | 任务级有 instance_id、region 空；评论级同 instance 有 region | GET workbench-link | 200，regionId 为回填地域 |
| T3 | 仅任务级 CSC 且 instance_id、region 皆空 | GET workbench-link | 400，文案含「缺少实例ID或地域」 |
| T4 | 任务级完整 aliyun 实例（既有） | GET workbench-link | 200（不回归） |
| T6 | 任务级 i-task + 评论 a/b 各有实例 | GET workbench-link?comment_id=cmt-b | 200，url 含 instanceId=i-comment-b，不含任务/评论 a |

## 数据与环境

- `setupCloudTestDB`；`cloud_server_configs` 双行（comment_id='' 与非空）
- 不调用真实 Aliyun API（只拼 ecs-workbench URL）

## 通过标准

`go test ./src -count=1 -run 'TestWorkbenchLink'` 全绿；前端相关 vitest 全绿。
