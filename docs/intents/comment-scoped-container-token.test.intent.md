# Test Intent: 评论级容器令牌

| ID | 用例 | 期望 |
|---|---|---|
| T1 | `TestTokenService_IssueTokenRequiresCommentID` | 无 comment → COMMENT_ID_REQUIRED |
| T2 | `TestTokenService_TwoCommentsIndependentExchange` | 两评论各换票，互不 403 |
| T3 | `TestParseTokenInitPath` | 11 段路径解析出 comment |
| T4 | `TestBootstrapStartVmTokensIncludesCommentInPath` | Cloud init URL 含 `/comment/` |
| T5 | `saasInboundScope.test.mjs` | COMMENT_ID 写入 exchange/reachability 字段 |
| T6 | `saasTaskCloud.layerGraphPush.test.mjs` COMMENT_ID | layer-graph-push body 含 comment_id |
| T7 | `saasInboundScope.scan.test.mjs` | SaaS 出站源文件均经 postJson / withSaasInboundScope |
| T8 | `autoRunPrBackfill.test.mjs` COMMENT_ID | complete 出站 body 含 comment_id |
