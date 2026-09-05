# Review: 评论 CSC 云平台元数据不变量

- **日期**: 2026-08-14
- **结论**: 通过（无 Critical）

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | 按字段补齐；合法覆盖保留；新建拒 mock。现网目标行已是 aliyun+青岛+cpa |
| Readability | fill/heal 命名与注释对齐不变量 |
| Architecture | 无新组件；v80 评论权威上补不变量 |
| Security | 无新 path；internal 仍要 secret；SQL 参数化 |
| Performance | 每请求最多一次模板读 + upsert |

## 调用链

- resolveScoped / ensure / attach / ingress(comment_id) / internal lookup 均 heal
- PATCH server-config 仍只打任务级（既有）

## Intent→Event

同库投影例外已写在意图文档。
