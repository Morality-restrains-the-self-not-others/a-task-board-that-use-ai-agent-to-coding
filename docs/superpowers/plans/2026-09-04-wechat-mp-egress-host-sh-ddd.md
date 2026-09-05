# DDD — wechat-mp-egress-host-sh

无新聚合。既有 `WechatMPFollowTicket` 仍由 taskAuth 持有。

新增基础设施适配器：`wechatmpegress.Server`（端口出站），不改变领域事件契约。

业务意图 → 事件：见设计文档（无新事件）。
