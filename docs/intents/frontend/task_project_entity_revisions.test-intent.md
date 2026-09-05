# 测试意图：任务/项目内容历史 UI

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 任务详情渲染入口 | `data-testid=entity-revision-open` 可见 |
| F2 | 打开列表面板 | 请求 GET revisions；行 `entity-revision-row` |
| F3 | 点选版本 | 只读区显示该 title/description |
| F4 | 空列表 | `entity-revision-empty` 含「尚无历史版本」 |
| F5 | 加载失败 | 错误节点含 `data-traceId` |
| F6 | 项目详情对称 | 同 F1–F4 |
| F7 | 未打开不请求 | 无后台 GET 轮询 |
| F8 | 加载中可收起 | `entity-revision-close` 可关上面板；入口不得因 GET 保持 disabled |
| F9 | 单击展开双击收起 | 与下拉菜单规范一致 |
| F10 | 点击面板外收起 | document click 在面板外时关闭 |

可执行：`taskFE/app/src/components/entity-revision/*.test.js`；`TaskDetail` / `ProjectDetail` 相关 unit 测。
