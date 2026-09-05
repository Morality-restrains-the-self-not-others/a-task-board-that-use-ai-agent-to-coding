# 测试意图：购买页 GitLab 流量并入磁盘卡片并共用区域

## 测试目标

证明 VIP1 购买页 GitLab 流量位于磁盘卡片内、只保留一个区域下拉，且磁盘/流量订单行共用该区域。

## 测试分层

- 前端单元：`taskFE/app/src/views/OrderCreate.contract.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | VIP1 打开购买页 | `order-gitlab-resources` 同时含「GitLab 磁盘」与「GitLab 流量」 |
| T2 | 区域控件 | 仅一个 `order-gitlab-region`；`order-gitlab-traffic-region` 不存在 |
| T3 | 同时购买磁盘与流量 | POST 两行 region 均为所选 slug |
| T4 | 仅购买流量 | POST 仅 `gitlab_traffic`，region 为共用下拉值 |
| T5 | 未选区域即填流量 | 不 POST；文案「请先选择 GitLab 区域」 |

## 数据与环境

jsdom + mock `apiFetch`；membership `vip1`；regions 至少含一个启用 slug。

## 通过标准

T1–T5 全绿；既有「创建成功跳转」「磁盘 POST region」「施工留言」「防重放」「失败 data-traceId」不回归。

## 业务意图 → 事件对照（测试）

纯前端 payload 组装，不断言 MQ。
