# ramsync 与大规模 Git 操作的顺序规范

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-06
- 维护者：Trae AI 团队
- 来源：OPT-20260806-001 执行中发现（git-hooks v13 迁移期间文件被磁盘镜像"复活"）

## 背景（为何是元规则）

`/tmp/ram-work` 是 tmpfs 内存挂载，由 `/home/ljy/bin/ramsync-daemon.sh` 守护进程与磁盘镜像 `/home/ljy/gitClone/ramDisk/ram-mount` 双向维护：

- **运行时**：周期 RAM → DISK 同步（含 `--delete` 镜像收敛）
- **启动时**：DISK → RAM 初始化（含 `--delete`）— **会把磁盘侧的旧文件整体倒回内存工作树**

守护进程重启（崩溃/系统重启/手动 start）恰好落在批量 git 操作期间时，磁盘镜像的陈旧文件会覆盖/复活刚删除的文件，且不触碰 git 索引——表现为「文件反复出现」「git status 出现莫名 D/A/M」。v13 迁移期间实测：`db/scripts/hooks` 等已删除文件在 00:25、00:28 两次被复活。

## 规则分类

### 核心规则

#### 大规模 Git 操作前先停守护进程

- **描述**：凡涉及**跨仓库批量** git 操作（`deploy_repo_random_precommit.sh`、`commit_with_submodules.py --apply`、子模块批量提交/推送、目录级 `git rm`/`git mv`、迁移类脚本），**必须先**：
  1. `bash /home/ljy/bin/ramsync-daemon.sh stop`
  2. 执行 git 操作
  3. `bash /home/ljy/bin/ramsync-daemon.sh sync`（RAM → DISK，封存正确状态）
  4. `bash /home/ljy/bin/ramsync-daemon.sh start`
- **优先级**：高
- **规则类型**：禁止忽略（操作失败可重试；状态被磁盘镜像覆盖则需重做）

#### 文件级删除后立即验证

- **描述**：`git rm`/`rm -rf` 删除文件后，**立即**执行 `git status` 验证删除生效；若文件"复活"，检查守护进程是否在重启窗口内，先 `stop` 再重删。
- **优先级**：中

### 例外

- 单仓库、少量文件的日常提交（无批量删除/迁移）不受影响，无需停守护进程
- 守护进程已确认长期未重启（`pgrep -af ramsync` 的 PID 起于数日前）时，风险低，可视情况跳过

## 命令速查

```bash
bash /home/ljy/bin/ramsync-daemon.sh stop     # 停止守护进程
bash /home/ljy/bin/ramsync-daemon.sh sync     # 手动 RAM → DISK 同步
bash /home/ljy/bin/ramsync-daemon.sh start    # 重启守护进程
bash /home/ljy/bin/ramsync-daemon.sh verify   # dry-run 对比差异
```

## 与其它规则的关系

- 与「子仓库优先提交」（第 32 条）：批量提交子仓属大规模操作，先停守护进程
- 与「Git Hooks 统一管理」（git-hooks v13 / OPT-20260806-001）：分发器批量部署 hooks 前先停守护进程
