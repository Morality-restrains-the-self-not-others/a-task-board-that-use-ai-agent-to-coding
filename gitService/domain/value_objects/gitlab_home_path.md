# Value Object: GitLabHomePath

- **Context**: gitService (infra)
- **Fields**: absolute path, fstype, durable(bool)
- **Invariants**: path absolute; durable=false iff fstype ∈ {tmpfs, ramfs}
- **Persistence**: host filesystem under `GITLAB_HOME` (`config`/`logs`/`data`/`bootstrap_marks`)
- **Events**: migration logged as `GitLabHomeMigrated` (ops log only, no MQ)
