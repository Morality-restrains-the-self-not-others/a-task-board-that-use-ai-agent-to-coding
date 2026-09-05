# Value Stream: translate-branch-title Go native

```
用户输入中文标题 → CreateTaskModal debounce
  → POST …/translate-branch-title/
  → gateway → taskProjectService
  → [无中文] sanitize → 200
  → [含中文] fanyi_agent chat/completions → sanitize → 200
  → 工作分支名拼装
```

最小切片：去掉 Django hop；契约与前端零改。
