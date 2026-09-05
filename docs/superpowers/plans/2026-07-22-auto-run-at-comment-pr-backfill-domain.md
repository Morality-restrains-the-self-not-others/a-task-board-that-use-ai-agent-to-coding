# auto_run @ 评论 PR 回填 — DDD

## 限界上下文

| BC | 实体/对象 | 职责 |
|----|-----------|------|
| Task | HumanComment + ImageMention | 合成 auto_run @ 评论 |
| AI Comment | ContainerAgentComment | parent 挂载；assistant_response 承载 PR |
| Container Runtime | AutoRunJob + Delivery | 执行与交付；回填调用 |

## CRG→BC

unavailable；边界与既有 @mention / auto_run 一致，新增 `at_mention_run.source` 区分触发源。

## 领域事件

| 意图 | 事件 | 说明 |
|------|------|------|
| 合成 auto_run @ 评论 | （证据豁免）不发 `TASK_COMMENT_IMAGE_MENTIONED` | 启服已由 auto_run 路径负责；误发会导致双 VM |
| PR 回填 | 无新 MQ | 副作用：Agent complete HTTP |
