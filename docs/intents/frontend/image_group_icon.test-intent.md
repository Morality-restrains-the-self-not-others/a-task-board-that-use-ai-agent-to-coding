# 测试意图：编辑镜像组弹窗图标

- **对应**: `docs/intents/frontend/image_group_icon.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| F1 | validateImageGroupForm 无 icon | 返回镜像组图标必填 |
| F2 | 超 512KB | 返回大小错误 |
| F3 | 已有 icon_file_key 无新文件 | 校验通过 |
