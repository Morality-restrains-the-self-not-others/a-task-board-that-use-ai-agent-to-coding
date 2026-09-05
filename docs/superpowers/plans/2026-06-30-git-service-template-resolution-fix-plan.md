# 实施计划: 修复 git-service GitLab 容器 crash loop

> 文件: `gitService/run.sh`（Python 内联脚本 3 行）

## 任务清单

### Task 1: 确认当前代码位置
- [ ] 定位 `load_gitservice_config()` Python heredoc 中 `hostname` 推导逻辑
- [ ] 确认 `allowed_host = ""` 和 `hostname = allowed_host or host` 的位置

### Task 2: 增加模板变量检测
- [ ] 在 `allowed_host` 赋值后、`hostname` 赋值前增加:
  ```python
  if allowed_host and '${' in allowed_host:
      allowed_host = ""
  ```
- **验证:** `GITLAB_HOSTNAME` 和 `GITLAB_EXTERNAL_HOST` 输出为 `127.0.0.1`

### Task 3: 手动验证
- [ ] `bash gitService/run.sh stop --clean` 清理
- [ ] `bash gitService/run.sh start` 启动
- [ ] `docker ps --filter name=gitlab` 确认容器 Running（非 Restarting）
- [ ] `docker logs gitlab` 确认无 `URI::InvalidURIError`
- [ ] 确认 `生效配置: host=127.0.0.1` 输出

## 依赖

```
Task 1 → Task 2 → Task 3
```
