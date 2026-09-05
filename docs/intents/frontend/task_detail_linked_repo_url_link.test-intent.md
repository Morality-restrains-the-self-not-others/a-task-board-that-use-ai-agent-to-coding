# 测试意图：任务详情关联仓库地址可跳转

## 测试目标

证明任务详情关联项目中的 http(s) 仓库 URL 渲染为真实外链，且非 http(s) 引用不会变成 `javascript:` 链接。

## 测试分层

| 层 | 文件 | 覆盖 |
|----|------|------|
| 组件（Vue） | `taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsViewMode.test.js` | 只读态 href / target / rel；非 http(s) 纯文本 |
| 组件（Vue） | `taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsEditMode.test.js` | 编辑态 http(s) 外链 |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 只读态 `https://github.com/demo/repo-a.git` | `a[data-testid=task-linked-repo-url]` href 相同，`target=_blank`，`rel` 含 `noopener` |
| T2 | 只读态 `javascript:alert(1)` 与 `git@…` | 无 `task-linked-repo-url`；无 `a[href^=javascript:]` |
| T3 | 编辑态 `https://gitlab.daydaymoney.com/org/ram-work` | 同上外链属性 |

## 通过标准

上述测例全绿。
