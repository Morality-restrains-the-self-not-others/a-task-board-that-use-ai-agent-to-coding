# Plan: 评论 CSC 云平台元数据不变量

- **状态**: 实现已完成（/goal 收口）
- **设计**: `docs/superpowers/specs/2026-08-14-comment-csc-cloud-meta-invariant-design.md`

## 任务清单

- [x] T1–T5 ensure/resolve 单测（拒 mock、克隆模板、保留覆盖、只补空字段）
- [x] `fillCommentCSCIllegalCloudMeta` 按字段补齐；ensure 禁止默认 mock
- [x] 020 SQL `CASE WHEN` 空/mock 才写模板
- [x] Workbench：真实 `i-` 拼阿里云 URL；`mock-` 仍拒绝
- [x] attach / ingress(comment_id) / internal lookup 走 heal
- [x] 意图文档：`docs/intents/backend/cloud/comment_csc_cloud_meta_invariant.intent.md`
- [x] 现网：020 已应用；目标行已是 aliyun+青岛+cpa；task-cloud-service 16:13 编译重启
- [ ] 公网点 Workbench（OPT-20260814-016 / BROWSER）

## 事件契约

无新 Kafka。补齐为例外（同库投影）。
