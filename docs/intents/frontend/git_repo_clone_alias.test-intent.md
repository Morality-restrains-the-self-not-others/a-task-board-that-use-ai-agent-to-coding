# 测试意图：Git 仓库克隆别名

## 覆盖点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 创建项目仅填 URL | `git_repos` 可仍为 string[]；克隆目录 = URL 推导 |
| T2 | 创建项目 URL + 别名 `my-app` | 持久化 `clone_alias=my-app`；响应含 `git_repo_entries` |
| T3 | 编辑项目改别名 | PATCH 后详情返回新别名 |
| T4 | 别名含非法字符 | sanitize 后落盘或克隆时 sanitize |
| T5 | 空别名 | 等价未填 |
| T6 | trae-agent collect | 从 `git_repo_entries` / 对象项读取别名并用于目录名 |
| T7 | reclone | 有别名时目标目录用别名 |
| T8 | 同项目重复别名 | 前端阻止提交或提示 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-14 | 初版 |
