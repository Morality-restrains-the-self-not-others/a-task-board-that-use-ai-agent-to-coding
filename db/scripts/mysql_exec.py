#!/usr/bin/env python3
"""
Resolve MySQL DSN from registry.yaml and execute SQL via PyMySQL.
Usage: python3 db/scripts/mysql_exec.py <db_key> <sql_file_or_dir>
  - Resolves DSN using the same logic as dbload.ResolveMySQLDSN
  - Executes SQL against MySQL using PyMySQL (no mysql CLI required)
  - If sql_path is a directory, executes all .sql files in order
"""
import os, sys, yaml, glob, re

try:
    import pymysql
except ImportError:
    import sys
    print("PyMySQL is required: pip3 install PyMySQL", file=sys.stderr)
    sys.exit(1)


def resolve_dsn(registry, key):
    """Replicate dbload.ResolveMySQLDSN logic."""
    key = key.strip()

    # Per-key env override
    env_map = {
        "saas": "SAAS_MYSQL_DSN", "task-auth": "TASKAUTH_MYSQL_DSN",
        "task-bill": "TASKBILL_MYSQL_DSN", "task-budget": "TASK_BUDGET_MYSQL_DSN",
        "git-oauth": "GITOAUTH_MYSQL_DSN", "ai-provider": "AI_PROVIDER_MYSQL_DSN",
        "task-project": "TASK_PROJECT_MYSQL_DSN", "task-task": "TASK_TASK_MYSQL_DSN",
        "task-cloud": "TASK_CLOUD_MYSQL_DSN", "task-ai-comment": "TASK_AI_COMMENT_MYSQL_DSN",
        "container": "CONTAINER_MYSQL_DSN", "task-tenant": "TASK_TENANT_MYSQL_DSN",
        "task-referral": "TASK_REFERRAL_MYSQL_DSN",
    }
    env_name = env_map.get(key)
    if env_name:
        dsn = os.environ.get(env_name, "").strip()
        if dsn:
            return dsn

    # Global MYSQL_DSN
    dsn = os.environ.get("MYSQL_DSN", "").strip()
    if dsn:
        return dsn

    # From registry.yaml
    entries = registry.get("databases", {})
    entry = entries.get(key)
    if not entry:
        raise SystemExit(f"registry.yaml missing database key: {key}")

    mysql = registry.get("mysql")
    if not mysql:
        raise SystemExit("registry.yaml has no mysql block")

    db_name = entry.get("database", "").strip() or key
    dsn = f"{mysql['user']}:{mysql['password']}@tcp({mysql['host']}:{mysql['port']})/{db_name}?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true"
    return dsn


def dsn_to_conn_params(dsn):
    """Convert Go DSN to PyMySQL connection parameters."""
    # Format: user:password@tcp(host:port)/dbname?params
    m = re.match(r'([^:]+):([^@]+)@tcp\(([^:]+):(\d+)\)/([^?]+)', dsn)
    if not m:
        raise SystemExit(f"Cannot parse DSN: {dsn}")
    user, password, host, port, dbname = m.groups()
    return {
        "host": host,
        "port": int(port),
        "user": user,
        "password": password,
        "database": dbname,
        "charset": "utf8mb4",
        "autocommit": True,
    }


def is_non_fatal_error(msg):
    """Check if the error is non-fatal (duplicate key/column/index)."""
    msg_lower = msg.lower()
    non_fatal_patterns = [
        "duplicate key name",
        "already exists",
        "duplicate column name",
        "duplicate entry",
        "table .* already exists",
        "can't drop .* check that",
        "error 1060",   # Duplicate column name
        "error 1061",   # Duplicate key name
        "error 1050",   # Table already exists
        "error 1062",   # Duplicate entry
        "error 1091",   # Can't DROP; check that column/key exists
    ]
    for pattern in non_fatal_patterns:
        if re.search(pattern, msg_lower):
            return True
    return False


def execute_sql(conn_params, sql_content, db_key, filename):
    """Execute SQL content against MySQL using PyMySQL."""
    conn = pymysql.connect(**conn_params)
    try:
        with conn.cursor() as cursor:
            # Split by semicolons and strip comment-only / blank stanzas.
            # SQL comments containing ";" (e.g. "-- reference; the real...")
            # will split the sentence; discard the non-SQL fragment that follows.
            _SQL_KEYWORDS = {
                'CREATE', 'ALTER', 'DROP', 'INSERT', 'UPDATE', 'DELETE',
                'SELECT', 'SET', 'USE', 'GRANT', 'REVOKE', 'TRUNCATE',
                'RENAME', 'REPLACE', 'LOCK', 'UNLOCK', 'BEGIN', 'COMMIT',
                'ROLLBACK', 'SAVEPOINT', 'CALL', 'EXPLAIN', 'DESCRIBE',
                'SHOW', 'FLUSH', 'KILL', 'LOAD', 'HANDLER', 'PREPARE',
                'EXECUTE', 'DEALLOCATE', 'ANALYZE', 'CHECK', 'CHECKSUM',
                'OPTIMIZE', 'REPAIR', 'CACHE', 'RESET', 'PURGE', 'CHANGE',
                'START', 'STOP', 'XA',
            }
            raw = [s.strip() for s in sql_content.split(';') if s.strip()]
            statements = []
            for s in raw:
                lines = s.split('\n')
                # Strip leading SQL line comments (--) and blank lines
                while lines and (not lines[0].strip() or lines[0].strip().startswith('--')):
                    lines.pop(0)
                # Strip leading non-SQL text (e.g. fragment from a ";"-split comment)
                while lines and lines[0].strip():
                    first_word = lines[0].strip().split()[0].upper() if lines[0].strip().split() else ''
                    if first_word in _SQL_KEYWORDS or first_word.endswith(';'):
                        break
                    lines.pop(0)
                stripped = '\n'.join(lines).strip()
                if stripped:
                    statements.append(stripped)
            for stmt in statements:
                try:
                    cursor.execute(stmt)
                except pymysql.err.OperationalError as e:
                    error_code = e.args[0] if e.args else 0
                    error_msg = e.args[1] if len(e.args) > 1 else str(e)
                    if is_non_fatal_error(error_msg):
                        print(f"[mysql_exec] {db_key}: non-fatal error (skipped): {error_msg}", file=sys.stderr)
                        continue
                    raise
    finally:
        conn.close()


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <db_key> <sql_file_or_dir>", file=sys.stderr)
        sys.exit(1)

    db_key = sys.argv[1]
    sql_path = sys.argv[2]

    # Find monorepo root
    script_dir = os.path.dirname(os.path.abspath(__file__))
    repo_root = os.path.dirname(script_dir)  # db/scripts/../ = repo root
    while repo_root and not os.path.exists(os.path.join(repo_root, "db", "registry.yaml")):
        parent = os.path.dirname(repo_root)
        if parent == repo_root:
            raise SystemExit("Cannot find monorepo root (db/registry.yaml)")
        repo_root = parent

    with open(os.path.join(repo_root, "db", "registry.yaml")) as f:
        registry = yaml.safe_load(f)

    dsn = resolve_dsn(registry, db_key)
    conn_params = dsn_to_conn_params(dsn)

    # Collect SQL files
    sql_files = []
    if os.path.isdir(sql_path):
        for f in sorted(glob.glob(os.path.join(sql_path, "*.sql"))):
            sql_files.append(f)
    else:
        sql_files.append(sql_path)

    if not sql_files:
        print(f"No SQL files found in {sql_path}", file=sys.stderr)
        sys.exit(1)

    for sql_file in sql_files:
        print(f"[mysql_exec] {db_key}: executing {os.path.basename(sql_file)}...", file=sys.stderr)
        with open(sql_file, "r") as f:
            sql_content = f.read()
        try:
            execute_sql(conn_params, sql_content, db_key, os.path.basename(sql_file))
        except Exception as e:
            error_msg = str(e)
            if is_non_fatal_error(error_msg):
                print(f"[mysql_exec] {db_key}: non-fatal error (skipped): {error_msg}", file=sys.stderr)
                continue
            print(f"[mysql_exec] {db_key}: FAILED: {error_msg}", file=sys.stderr)
            sys.exit(1)
        print(f"[mysql_exec] {db_key}: OK", file=sys.stderr)


if __name__ == "__main__":
    main()
