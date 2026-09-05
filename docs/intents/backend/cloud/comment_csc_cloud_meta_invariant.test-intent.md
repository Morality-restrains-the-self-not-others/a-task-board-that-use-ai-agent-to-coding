# 测试意图：评论 CSC 云平台元数据不变量

## 测试目标

验证评论 CSC 不得持久化 mock/空 region/空授权；任务模板只补非法字段；评论级合法覆盖不被抹掉。

## 测试分层

- 单元：`taskCloudService/src/comment_csc_ensure_test.go`、`compute_scoped_csc_test.go`、`compute_workbench_link_test.go`
- 无新增 MQ 断言（补齐为同库投影，见功能意图例外）

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 模板 aliyun+青岛+cpa-1；无评论行 | ensure(comment) | 新建行 platform=aliyun、region=青岛、auth=cpa-1，不是 mock |
| T2 | 无任务模板或模板 platform 空 | ensure(comment) | error，库中无 mock 评论行 |
| T3 | 评论已是 aliyun+杭州+cpa-user；模板青岛+cpa-1 | ensure / resolve | 仍为杭州+cpa-user |
| T4 | 评论 mock、region/auth 空；模板合法 | resolve | 补齐三字段；instance/server_url 不变 |
| T5 | 评论 aliyun、region 空、auth=cpa-user | resolve | 只补 region；auth 仍为 cpa-user |
| T6 | T4 之后 GET workbench-link | — | 200，url 含真实 instance + 补齐后的 region |
| T7 | instance=`mock-abc` | workbench-link | 仍 400 Mock 拒绝 |

## 数据与环境

- `setupCloudTestDB`；不调用真实 Aliyun API

## 通过标准

`go test ./src -count=1 -run 'TestEnsureCommentCSC|TestResolveScoped|TestWorkbenchLink|TestFillCommentCSC'` 全绿。
