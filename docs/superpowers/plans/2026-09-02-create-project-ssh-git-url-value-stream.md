# 价值流 — 创建项目 ssh:// Git URL

- **Date:** 2026-09-02

## 增量（唯一）

租户成员在创建项目粘贴 `ssh://git@host[:port]/path.git` → 前端格式通过 → `validate-git-repos` 将 URL 规范为 HTTP(S) 做 OAuth/可达性 → 用户可授权 → 项目保存原始 `ssh://` URL。

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| VS-1 | 输入 `ssh://git@github.com/owner/repo.git` | 无格式错误；创建按钮不因格式禁用 |
| VS-2 | 输入 `ssh://git@gitlab.daydaymoney.com:2222/g/p.git` | 格式合法；normalize 为 website origin 或 `https://host/g/p` |
| VS-3 | 既有 `git@` / `https://` | 回归不回归失败 |
| VS-4 | `ftp://` / 无 path 的 `ssh://host` | 仍拒绝 |

## 事件

无新业务事件。项目创建仍走存量 `PROJECT_CREATED`（若已有）。
