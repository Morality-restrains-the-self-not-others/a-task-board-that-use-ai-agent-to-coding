# Go DDD Compliance (valueStream / generic module)

执行命令：

```bash
python3 valueStream/scripts/ci/check_go_ddd_compliance.py
```

## 参数

- `--module-root <name>`：指定模块目录（默认 `valueStream`），脚本会扫描 `<module-root>/domain/**/*.go`。
- `--forbid-external`：开启后，领域层禁止任何第三方依赖导入（默认关闭，避免误报）。
- `--strict-module-root`：开启后，若 `<module-root>/domain` 不存在则直接失败（退出码 2）。

示例：

```bash
python3 valueStream/scripts/ci/check_go_ddd_compliance.py --module-root valueStream
python3 valueStream/scripts/ci/check_go_ddd_compliance.py --module-root valueStream --forbid-external
python3 valueStream/scripts/ci/check_go_ddd_compliance.py --module-root valueStream --strict-module-root
```

## 校验规则

1. 领域层禁止导入基础设施/应用层包（如 `<module-root>/src`、`net/http`、`database/sql`、`gopkg.in/yaml.v3`）。
2. `domain/aggregates` 禁止依赖 `domain/services` 和 `domain/repositories`。
3. `domain/repositories` 至少需要一个 `interface` 声明。

## 与统一入口关系

仓库根命令：

```bash
python3 runAll/scripts/ci/check_ddd_bdd_compliance.py
```

会同时执行：

- `task2app/scripts/ci/check_ddd_bdd_compliance.py`（Python DDD/BDD）
- `valueStream/scripts/ci/check_go_ddd_compliance.py`（Go DDD）

