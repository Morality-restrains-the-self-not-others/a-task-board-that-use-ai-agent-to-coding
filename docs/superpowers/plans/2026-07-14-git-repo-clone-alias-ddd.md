# DDD：Git 仓库克隆别名

## 限界上下文

- **项目管理（taskProjectService）**：`ProjectRepo` 聚合局部实体，新增值 `CloneAlias`
- **任务运行（taskTaskService / taskCredentialService）**：只读透传，不拥有别名写权
- **容器运行时（trae-agent）**：消费 `CloneAlias` 决定工作区目录名

## 模型

```
ProjectRepo
  - id
  - projectId
  - repoUrl (identity for matching)
  - cloneAlias (optional; empty → derive from URL at runtime)
```

## 不变式

1. `repoUrl` 非空
2. `cloneAlias` 经 sanitize；空允许
3. 同项目非空 `cloneAlias` 建议唯一（应用层校验）
4. OAuth/匹配键 = URL，≠ alias
