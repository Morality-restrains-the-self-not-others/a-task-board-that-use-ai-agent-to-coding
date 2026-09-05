<!-- markdownlint-disable MD013 MD060 -->
# 测试意图：GitLab 连接页流量超额阻断提示

## 测试分层

- `taskFE/app/src/views/WorkspaceSettingsGitlabConnection.test.js`
- `taskFE/app/src/composables/useGitlabResourcePurchase.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | currentTrafficGb=0，active | 可见 `gitlab-traffic-quota-blocked` |
| T2 | currentTrafficGb=5，used=1，allowed=true | 无阻断条 |
| T3 | 阻断条购买链接含 tenant billing orders | `<a href>` |

## 通过标准

T1–T3 全绿。
