# 测试意图：租户购买 GitLab 资源须按行指定区域

## 覆盖

| 场景 | 测试 | 期望 |
|------|------|------|
| 磁盘缺 region | `TestCreateOrder_GitlabDiskRequiresRegion` | error 含 region |
| 非法 slug | `TestCreateOrder_GitlabDiskUnknownRegion` | `region not found` |
| 指定区域落订单行 | `TestCreateOrder_GitlabDiskWritesSelectedRegion` | item.region = slug |
| 前端磁盘 POST region | `OrderCreate.contract.test.js` | body 含 region |
| 磁盘/流量合卡同区 | `OrderCreate.contract.test.js` | 仅一个区域下拉；两行同一 region |
| 详情展示 | `OrderDetail.contract.test.js` | 可见 slug |
| 设置页入口 | `WorkspaceSettingsGitlabConnection.test.js` | href 含 `orders/create` |

## 命令

```bash
cd /tmp/ram-work/taskBill && go test ./src -count=1 -run 'CreateOrder_Gitlab'
cd /tmp/ram-work/taskFE/app && npx vitest run src/views/OrderCreate.contract.test.js src/views/OrderDetail.contract.test.js src/views/WorkspaceSettingsGitlabConnection.test.js
```
