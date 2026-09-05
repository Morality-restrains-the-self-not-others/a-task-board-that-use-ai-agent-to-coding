# 测试意图：Chrome 插件相邻兄弟多选

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_sibling_range_pick.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `collectContiguousSiblings` 同父两端 | 返回含端点的有序连续切片 |
| T2 | 不同父 | 返回 `null` |
| T3 | 同 tag 区间 CSS path | 含 `:nth-of-type(n+…):nth-of-type(-n+…)` |
| T4 | 混合 tag | 退化为 `:nth-child` 区间 |
| T5 | `snapshotSiblingRange` 单元素 | 与 `snapshotElement` 等价字段（label/cssPath） |
| T6 | `formatElementAdjustmentBlock` multi | 含「兄弟区间」与 `×N` 标签 |
| T7 | `unionClientRects` | 并集宽高正确 |

## 运行

```bash
cd taskChromePlugin && npm test
```
