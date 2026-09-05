# 实施计划: 修复 git-service 启动 mkdir 权限拒绝

> 输入:
> - 设计文档: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-git-service-mkdir-permission-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-30-git-service-mkdir-permission-fix-nfr-clarification.md`
>
> 文件: `gitService/run.sh`（3 行变更）

## 任务清单

### Task 1: 确认当前代码状态
- [ ] 读取 `gitService/run.sh` L159-L167 确认 `run_bootstrap_if_needed()` 当前实现
- [ ] 读取 `gitService/run.sh` L345-L350 确认 `stop --clean` 当前实现
- **验证:** 确认 L162 `mkdir -p` 在 L164 守卫条件之前，L161 路径为 `./gitlab_home/bootstrap_marks`

### Task 2: 修复 run_bootstrap_if_needed() — 移动守卫条件 + 更换路径
- [ ] 将 `NEED_BOOTSTRAP` 守卫条件（L164-L167）移至函数体最前面（`local BOOTSTRAP_MARKS_DIR` 之前）
- [ ] 将 `local BOOTSTRAP_MARKS_DIR` 和 `mkdir -p` 移至守卫条件之后
- [ ] 将 `BOOTSTRAP_MARKS_DIR` 从 `"./gitlab_home/bootstrap_marks"` 改为 `"./.bootstrap_marks"`
- **修改后结构:**
  ```bash
  run_bootstrap_if_needed() {
    if [[ "${NEED_BOOTSTRAP:-false}" != "true" ]]; then
      echo "复用已有容器，跳过 bootstrap 脚本。"
      return 0
    fi

    local BOOTSTRAP_MARKS_DIR="./.bootstrap_marks"
    mkdir -p "$BOOTSTRAP_MARKS_DIR"
    # ... bootstrap scripts unchanged
  }
  ```

### Task 3: 修复 stop --clean 路径
- [ ] L350: 将 `rm -rf ./gitlab_home/bootstrap_marks/*` 改为 `rm -rf ./.bootstrap_marks/*`
- **验证:** 路径与 Task 2 中的 `BOOTSTRAP_MARKS_DIR` 一致

### Task 4: 手动验证 — 容器已运行场景
- [ ] 运行 `bash gitService/run.sh start`
- [ ] 确认不再出现 `mkdir: Permission denied`
- [ ] 确认输出 `复用已有容器，跳过 bootstrap 脚本。`

### Task 5: 手动验证 — 全新容器场景（如可行）
- [ ] 运行 `bash gitService/run.sh stop --clean`
- [ ] 运行 `bash gitService/run.sh start`
- [ ] 确认 `.bootstrap_marks/` 目录创建成功
- [ ] 确认 bootstrap 标记文件创建成功

## 依赖关系

```
Task 1 → Task 2 → Task 3 → Task 4 → Task 5
```

所有任务顺序执行，无并行化机会。

## 风险

| 风险 | 缓解措施 |
|------|---------|
| `.bootstrap_marks/` 可能与现有 `.gitignore` 冲突 | `.bootstrap_marks/` 是隐藏目录，不在 git 跟踪范围内确认 |
| 其他脚本引用旧路径 | `grep -rn 'gitlab_home/bootstrap_marks'` 确认仅 `run.sh` 引用 |

## 回滚

还原 `gitService/run.sh` 中 3 行变更即可。
