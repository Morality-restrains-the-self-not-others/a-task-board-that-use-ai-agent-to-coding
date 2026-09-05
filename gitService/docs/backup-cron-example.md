# GitLab Backup Cron 示例

定期备份 GitLab 数据到非易失路径，防止单盘故障。

## 前提

- GitLab 容器名：`gitlab`
- GITLAB_HOME 已配置为非易失路径（见 `scripts/gitlab_home.sh`）
- 备份目标路径有足够空间

## Cron 示例

每天凌晨 3 点执行完整备份，保留最近 7 天：

```cron
0 3 * * * docker exec gitlab gitlab-backup create CRATE=backup STRATEGY=copy && find /path/to/GITLAB_HOME/data/backups -name '*.tar' -mtime +7 -delete
```

### 参数说明

- `STRATEGY=copy`: 使用 `cp` 而非 `mv`，备份期间数据库仍可写入
- `CRATE=backup`: 创建 tar 包
- `SKIP=artifacts,registry`: 可选，跳过 artifacts / container registry 以节省空间

## 备份验证

```bash
# 列出备份文件
ls -lh $GITLAB_HOME/data/backups/*.tar

# 验证备份完整性（需停服）
docker exec gitlab gitlab-backup restore BACKUP=<timestamp>
```

## 异地备份（可选）

```bash
# rsync 到远程
rsync -avz $GITLAB_HOME/data/backups/ user@backup-host:/backups/gitlab/
```

## 参考

- [GitLab Backup Documentation](https://docs.gitlab.com/ee/raketasks/backup_gitlab.html)
- 相关 OPT：OPT-20260722-067
