# 嵌套仓 gitlink submodule 迁移评估（OPT-20260717-030）

## 现状

- 父仓 `ram-work` 用 **`.gitmodules` 注册表** + `.gitignore` 忽略嵌套目录；各嵌套仓独立 `.git`，父索引**无** mode=160000 gitlink。
- 发现与克隆：`taskProjectService` / onlineServiceJS bootstrap 解析 `.gitmodules` 与 Git 相对 URL；容器与工作区约定为 `{父仓}/{path}`。

## 完整 gitlink submodule 的利弊

| 方面 | gitlink submodule | 维持注册表 |
|------|-------------------|------------|
| `git clone --recurse-submodules` | 标准 Git 工作流 | 需脚本/文档逐个 clone |
| runAll / 容器 bootstrap | 须改克隆顺序与 path 布局 | 已对齐 |
| 嵌套仓版本 pin | 父仓 commit 锁定子 SHA | 各仓独立 main，靠 CI/约定 |
| worktree | 子模块 worktree 复杂 | 在子仓内开 worktree（见 nested-repo-worktree-guide.md） |
| 迁移成本 | 高：历史、CI、镜像、Go 解析 | 无 |

## 结论

**暂不迁移**为完整 gitlink submodule。维持 **`.gitmodules` 注册表 + 独立克隆** 即可满足产品需求；若未来必须 pin 子 SHA 或统一 `git submodule update`，再单独立项评估 runAll/容器/runner 兼容性。
