# 测试意图：系统管理 GitLab 区域带宽展示

## 测试目标

区域卡片必须标明是否带宽共享分区，并展示总带宽与剩余带宽（Mbps）。

## 测试分层

- 前端：`SystemAdminGitlabRegionCapacity.test.js`、`gitlabRegionCapacity.test.js`、`SystemAdminGitlabResources.bandwidth.test.js`
- 后端：`taskBill/src/gitlab_region_capacity_test.go`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | `bandwidth_shared=true` | 文案「带宽共享分区」 |
| F2 | `bandwidth_shared=false` | 文案「独立带宽」 |
| F3 | 总量 200、剩余 80 | 展示 200 Mbps 与剩余 80 Mbps |
| F4 | 未配置（0/0） | 展示 0，不抛错 |
| B1 | attachDeployInfo | JSON 含三字段；剩余被 clamp 到 [0, total] |
| B2 | clamp remaining | remaining>total → total；负数 → 0 |

## 通过标准

上述测例全绿；公网页腾讯上海一区卡片可见带宽共享标记与 Mbps 数字。
