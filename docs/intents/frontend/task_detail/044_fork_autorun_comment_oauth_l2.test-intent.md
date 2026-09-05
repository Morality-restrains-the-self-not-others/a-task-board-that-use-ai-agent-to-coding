# 测试意图：Fork 自动运行评论继承源任务同用户 Git OAuth L2

- **日期**: 2026-09-02
- **对应意图**: [044_fork_autorun_comment_oauth_l2.intent.md](./044_fork_autorun_comment_oauth_l2.intent.md)

| ID | 场景 | 前置 | 动作 | 期望 |
|----|------|------|------|------|
| T1 | 同用户源 L2 | 源评论 GitLab L2；Fork 任务 fork_from=源 | `applyForkSourceCommentL2SeedToIdentities` | 新选择被 stamp `oauth_gitsite` |
| T2 | 他人源 L2 | 源评论作者 ≠ 操作者 | 同上 | 不 stamp |
| T3 | 非 Fork | fork_from 空 | 同上 | 不 stamp |
| T4 | 已有 L2 | 新选择已有 oauth_gitsite | 同上 | 不覆盖 remote |
| T5 | 站点级源行 | 源 `{repo_url:"", oauth_gitsite}` | 同上 | 按 gitsite stamp |
| T6 | 事件 | T1 成功 | 捕获 publish | `via=fork_source_comment_l2_seed`；键含 comment/user/gitsite |
| T7 | snapshot 补种 | Fork 评论 JSON 无 oauth | `loadSnapshotRepoIdentities` | 返回含 oauth；DB 已 UPDATE |
| T8 | ensure 路径 | `ensureAutoRunAtComment` Fork 任务 | 创建评论 | JSON 含 oauth_gitsite |
| T9 | 祖父任务 L2 | 中间 Fork 评论无 L2，祖父有 | `applyForkSourceCommentL2SeedToIdentities` | 沿 fork_from 链 stamp |
| T10 | snapshot 隐式 gitsite | 项目 L2 查询失败且源评论无 L2 | `loadSnapshotRepoIdentities` | 仍 stamp `oauth_gitsite` 为仓库 host |

可执行：`cd taskTaskService && go test ./src -count=1 -run 'TestApplyForkSourceCommentL2Seed|TestLoadSnapshotRepoIdentities_|TestEnsureAutoRunAtComment_SeedsFromForkSource'`
