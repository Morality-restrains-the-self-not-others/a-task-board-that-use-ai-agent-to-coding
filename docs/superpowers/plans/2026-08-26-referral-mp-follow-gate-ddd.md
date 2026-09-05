# DDD：推荐资格服务号关注闸门

## 限界上下文

- **Identity (taskAuth)**：微信身份、关注回调、follow-status。
- **Referral (taskReferral)**：申请资格；只读「是否已有 mp 别名」。

## 聚合

### WechatIdentity（已有）

- 不变式：`(app_key, openid)` 唯一；unionid 为跨应用真源。
- 新别名：`app_key=mp`，`app_id`=服务号/支付 AppID。
- 命令：`BindMpOpenIDFromSubscribe(unionID, openID, appID)` — 命中用户才写；否则 Pending。

### MpSubscribePending

- 实体：unionid（PK）、openid、app_id、created_at。
- 命令：`RecordUnmatchedSubscribe`、`ClaimForUser(userID, unionID)`。

## 领域事件

`WECHAT_MP_SUBSCRIBED` { user_id?, app_key, openid, unionid, outcome }

无 intent 消费者。发布在绑定或 pending 成功之后。

## 端口

- `WechatMPCallbackVerifier`：签名 / 解密。
- `WechatUserInfoPort`：openid → unionid（XML 无 UnionID 时）。
- `WechatIdentityRepository`：已有 SQL 适配。

## 领域层文件

- `taskAuth/domain/wechat_mp_subscribe.go` — 纯函数：签名材料排序、事件是否 subscribe、是否应建号（恒 false）。
