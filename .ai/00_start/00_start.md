# 起点
1. 按照项目规范使用适当的工具和方法进行项目开发
2. 将所有的价值流绘制在一张 docs/flows/value-stream-test-integration.wsd 中, 并在其中添加所有的测试点，每个测试点对应一个测试用例。每次添加功能或者单元测试时对应的更新这张图。
3. 当用户输入功能意图指令时，先按金字塔结构维护 `docs/intents/` 下的功能意图与测试意图文档（同名异后缀），再执行代码实现与测试验证。服务端业务意图须在文档与实现中映射为向消息队列投递的对应业务事件（见 `.ai/08_prompt_management/01_intent_driven_development.md`）。
4. 当出现网络中断后半路恢复、用户仅发送「继续」等续接语境时：先读 [会话中断与「继续」请求](./01_session_continuation.md)，再执行任务。
5. 会话收尾时：将可执行优化建议编号写入 [`.learnings/OPTIMIZATION_TODOS.md`](../../.learnings/OPTIMIZATION_TODOS.md)，完成后标记 `completed`；细则见 [会话结束优化建议 Todo](../01_project_constraints/25_session_end_optimization_todo.md)。