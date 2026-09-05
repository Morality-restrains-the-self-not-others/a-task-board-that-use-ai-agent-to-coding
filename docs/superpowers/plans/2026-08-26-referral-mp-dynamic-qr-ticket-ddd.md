# DDD：推荐资格服务号动态 scene 码

## 限界上下文

- **Identity (taskAuth)**：动态码、回调、票、mp 别名。
- **Referral (taskReferral)**：申请资格；只读是否已有 mp 别名（不变）。

## 聚合

### FollowTicket（新）

- 根：`temp_id`（Snowflake = 微信 scene_str）。
- 不变式：一张未过期 pending 票对应一个 `user_id`；conflict 不修改 WechatIdentity。
- 命令：`IssueFollowQR(userID)`、`MatchScene(tempID, openID, unionID)`。

### WechatIdentity（已有）

- 票命中且占用检查通过才 `BindMpOpenID(ticket.user_id, openID, unionID)`。

### MpSubscribePending（保留）

- 无 scene 的 subscribe 旁路。

## 领域事件

| 事件 | 何时 | 键 |
|------|------|-----|
| WECHAT_MP_SUBSCRIBED bound | 票绑定成功 | mp openid |
| WECHAT_MP_SUBSCRIBED conflict | 票冲突 | temp_id |
| WECHAT_MP_SUBSCRIBED pending | 无 scene 未命中用户 | mp openid |
| WECHAT_IDENTITY_CONFLICT | unionId/openid 属他人 | owner + openid |

## 领域纯函数

- `WechatMPIsFollowScanEvent` — subscribe 或 SCAN。
- `WechatMPSceneTempID` — 去掉 `qrscene_`。
- `WechatMPSubscribeCreatesUser` — 恒 false。

## 端口

- `WechatMPQrcodeCreatePort`：QR_STR_SCENE。
- `WechatUserInfoPort`：openid → unionid。
- `WechatMPCallbackVerifier`：签名 / 解密。
