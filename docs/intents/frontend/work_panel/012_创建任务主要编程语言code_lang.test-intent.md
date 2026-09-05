# 012_创建任务主要编程语言 code_lang（测试意图）

## 自动化

- `task2app/front_project/app`：`codeLangOptions.unit.test.js`（默认值、去重、响应解析）
- `taskProjectService`：`TestCodeLangOptions*`（默认三值、PUT/GET、工作区隔离）
- `taskTaskService`：`TestCreateAndListTasks` 断言 create/list 的 `code_lang`

## 手工

1. 打开工作台创建任务，确认「主要编程语言」下拉含 go / rust / js。
2. 选 `go` 创建，任务详情/列表响应含 `"code_lang":"go"`。
3. 工作区设置 → 编程语言：改为仅 `go`/`python`，再开创建弹窗，选项与之一致。
