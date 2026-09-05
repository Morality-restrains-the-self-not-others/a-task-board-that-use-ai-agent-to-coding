# 价值流：推荐资格服务号关注闸门

- **Date:** 2026-08-26
- **Increment:** 单增量交付（关注绑定 + 申请闸门 + UI）

## 触发到价值

```
已登录用户打开 /profile/referral/
  → 无资格且未绑服务号：看到二维码与说明
  → 微信扫码关注
  → 平台收到 subscribe，unionId 锁定用户并写入 mp openId
  → 用户点「我已关注」看到名称/简介表
  → 提交申请 → 审批/开通后分账可用服务号 openId
```

## 测试点（写入 value-stream 图）

| ID | 步骤 | 断言 |
|----|------|------|
| TP1 | 未绑定 | 无申请表，有 QR `data-testid=referral-mp-qr` |
| TP2 | 点「我已关注」仍未绑定 | 提示未检测到，表单仍隐藏；错误节点可有 data-traceId |
| TP3 | 已绑定 | 展示名称/简介表 |
| TP4 | subscribe + 已知 unionid | upsert mp 别名，不新建 user |
| TP5 | subscribe + 未知 unionid | pending 行；不建 user |
| TP6 | follow-status 消费 pending | 当前用户 unionid 匹配则绑定 |
| TP7 | apply 未绑定 | 400 `service_account_not_followed` |
| TP8 | 回调签名错误 | 非 200 echo / 非 success |

## 范围内 / 外

- 内：taskAuth 回调与身份、taskReferral 申请闸门、taskFE 闸门 UI、APISIX 路由、conf mp app。
- 外：动态 scene 码、模板消息、取关后撤销分账接收方（OPT）。
