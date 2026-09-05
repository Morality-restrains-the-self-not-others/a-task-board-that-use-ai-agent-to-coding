# Test Intent: 9999 初始化按钮标注未 migrate

## 测试目标

证明 missing SQL 会计入 pending 并驱动按钮标注；不可达不计 pending；全量已应用无高亮。

## 测试分层

- 单元：`runAll/src/domain/migrate_pending_test.go`（diff / report 聚合）
- HTTP：`runAll/src/ui_migrate_pending_test.go`
- 页面锚点：`runAll/src/ui_test.go` / `status_page_test.go`（id、API 路径、刷新函数名）

## 用例矩阵

1. **missing 计 pending**  
   Given local 有 `002.sql`、applied 仅 `001.sql`  
   When Diff  
   Then missing=`002.sql`，report.pending_count=1

2. **stale 不计 pending**  
   Given applied 多一个 Go step_key  
   When Diff  
   Then stale 非空，pending_count=0，status=ok

3. **表不存在 = 全部 pending**  
   Given ListApplied 返回空（表缺失）且 local 有文件  
   Then missing=全部 local

4. **不可达**  
   Given ListApplied error  
   Then status=unreachable，pending_count 不增加

5. **GET 契约**  
   Given 注入 fake reader  
   When GET `/api/dev/migrate-status`  
   Then 200 JSON 含 pending_count 与 databases

6. **UI 锚点**  
   Then 装配页含 `dev-init-db-pending-label`、`refreshMigratePendingStatus`、`/api/dev/migrate-status`，且不在 `refresh()` 内调用该刷新

## 数据与环境

- 临时目录构造 registry + migrate.sh + sql 文件；不连现网 MySQL
- 不依赖 :9999 进程

## 通过标准

```bash
cd runAll && go test ./src/ ./src/domain/ ./src/infrastructure/ -count=1 -run 'MigratePending|DiffStepKeys|StatusPage'
```

全部 PASS。

## 业务意图 → 事件对照

本意图无领域事件；测试不断言 MQ publish。
