# Plan: edit-run auto delivery + PR comment

## Tasks

- [x] Gateway：createJob 带 `edit_run_delivery` + 可选 parent/image；单测
- [x] onlineServiceJS：持久化 `edit_run_delivery`；完成钩子触发 force 交付
- [x] onlineServiceJS：按 parent 创建 Agent + PR 回填文案；单测
- [x] taskAIComment：public create 接受 X-Access-Token；单测
- [x] 前端：edit-run 传 `parent_comment_id` + `installed_image_id`
- [x] 意图勾选 + SPA build（若前端改）
