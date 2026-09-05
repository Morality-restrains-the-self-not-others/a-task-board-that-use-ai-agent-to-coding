# 测试意图：Chrome 插件使用说明

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_user_guide.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `UserGuide.listSectionIds` | 含 overview/login/float-create/element-pick/devtools-*/popup-extras |
| T2 | surface 过滤 | float 含 element-pick，不含 popup-extras |
| T3 | `renderCollapsibleHtml` / `renderFullGuideHtml` | 含 data-guide-surface 与章节标记 |
| T4 | USER_GUIDE.md | 文档化全部 section id |

## 运行

```bash
cd taskChromePlugin && npm test
```
