#!/usr/bin/env python3
"""重建 E2E 测试账号（OPT-20260812-022）并补齐公司默认体系（OPT-20260813-008）。

清库/重建后，`auth_user`/`auth_login_method` 中的约定测试账号
`contact@daydaymoney.com` 会丢失，所有经 `loginViaGatewayApi` 的 Playwright
E2E（如 CreateProject.git-repo-top）都会在网关登录返回 400「用户名或密码错误」。

本脚本幂等地重建：
  - task_auth.auth_user + auth_login_method（email，bcrypt(明文)）
  - task_tenant.tenant_company（E2E 租户 850256677331562496）+ tenant_company_member
  - task_project 公司默认体系（OPT-20260813-008）：模拟 COMPANY_CREATED 事件链产物——
    默认进度体系（ps_default_system 含列）+ 默认交付物体系（ds_default_global）+ 默认工作空间，
    并将已有工作空间绑定进度/交付物，避免任务创建报「进度列不属于当前工作空间」、
    面板「未加载交付物类别」。

用法：
  python3 tests/helpers/seed_e2e_account.py            # 默认 docker 容器 + root123456
  python3 tests/helpers/seed_e2e_account.py <container> [mysql_password] [company_id]
  python3 tests/helpers/seed_e2e_account.py --superadmin <container> [mysql_password]
      # 仅重置超管 author@example.com 的密码为已知 E2E 测试密码（OPT-20260824-001：
      # 种子管理员密码已改为随机值，E2E 测试环境须显式重置为已知明文才能登录）

说明：
  taskAuth handleLogin 对**明文**密码做 bcrypt 比对（bcrypt ≤72 字节，Django
  pbkdf2_sha256 产物恒超限），故 password_hash 存 bcrypt(明文)，与
  gatewayLoginE2e.js 直发明文一致。
"""
import os
import subprocess
import sys
import time

SUPERADMIN_EMAIL = "author@example.com"
# 超管 E2E 测试密码（只读环境变量；禁止硬编码回退 — OPT-20260827-015）。
# reset_superadmin_password 直接写 bcrypt 哈希，不需要明文；该常量仅供
# 文档/回归测试说明哈希对应的明文来自环境。
SUPERADMIN_PLAIN_PASSWORD = os.environ.get("SUPERADMIN_PLAIN_PASSWORD")
SUPERADMIN_BCRYPT_HASH = "$2a$12$LR3V2Zi5lb/s/nFMuOjA2eF5CyNiTRxDRs8mjhua7XKVAkDkvS0C."


def snowflake_id() -> int:
    """镜像 taskAuth src/snowflake.go（epoch 1577836800000, machineID 7）。"""
    epoch = 1577836800000
    ts = int(time.time() * 1000)
    return ((ts - epoch) << 22) | (7 << 12) | 0


def mysql(container: str, password: str, sql: str) -> None:
    cmd = ["docker", "exec", "-i", container, "mysql", "-uroot", f"-p{password}", "-e", sql]
    subprocess.run(cmd, check=True, text=True)


def build_ensure_defaults_sql(company_id: str) -> str:
    """补齐 COMPANY_CREATED 事件链产物（幂等，可重复执行）。

    对应 taskEvents company_created intents 1/2/3 与
    dataMigrate/taskProjectService/007_seed_defaults.sql：
      - 系统级默认体系：交付物 ds_default_global（含列）、进度 ps_default_system（含列）
      - 租户默认映射：project_deliverable_systems_default_tenant / project_progress_systems_default_tenant
      - 默认工作空间：project_workspace_entries + project_progress_systems_workspace 绑定
    行 ID 使用稳定派生 ID（ddst_/psdt_/psw_ 前缀），保证重复执行不产生重复行。
    """
    cid = company_id
    return f"""
-- 1. 系统级默认体系（与 007_seed_defaults.sql 同源，幂等）
INSERT IGNORE INTO task_project.project_deliverable_systems(id, name, description, company_id, is_system, is_default)
VALUES('ds_default_global', '全局默认交付物体系', '系统内置默认交付物体系', '', 1, 1);
INSERT IGNORE INTO task_project.project_deliverable_columns(id, system_id, name, order_num) VALUES
  ('dc_def_global_lv1', 'ds_default_global', '价值流', 0),
  ('dc_def_global_lv2', 'ds_default_global', '业务流程', 1),
  ('dc_def_global_lv3', 'ds_default_global', '活动', 2),
  ('dc_def_global_lv4', 'ds_default_global', '工作项 / 交付物', 3),
  ('dc_def_global_lv5', 'ds_default_global', '子工作项 / 子交付物', 4);
INSERT IGNORE INTO task_project.project_progress_systems(id, name, is_default)
VALUES('ps_default_system', '系统默认进度体系', 1);
INSERT IGNORE INTO task_project.project_progress_columns(id, system_id, name, order_num) VALUES
  ('pc_def_sys_1', 'ps_default_system', '待处理', 0),
  ('pc_def_sys_2', 'ps_default_system', '进行中', 1),
  ('pc_def_sys_3', 'ps_default_system', '已完成', 2),
  ('pc_def_sys_4', 'ps_default_system', '已取消', 3);

-- 2. 租户默认交付物体系（company-created intent 1，独立表，upsert）
INSERT INTO task_project.project_deliverable_systems_default_tenant(id, tenant_id, deliverable_id)
VALUES(CONCAT('ddst_{cid}'), '{cid}', 'ds_default_global')
ON DUPLICATE KEY UPDATE deliverable_id='ds_default_global', updated_at=CURRENT_TIMESTAMP;

-- 3. 租户默认进度体系（company-created intent 2，target_type='system' 指向 ps_default_system，upsert）
INSERT INTO task_project.project_progress_systems_default_tenant(id, tenant_id, target_type, target_id)
VALUES(CONCAT('psdt_{cid}'), '{cid}', 'system', 'ps_default_system')
ON DUPLICATE KEY UPDATE target_type='system', target_id='ps_default_system', updated_at=CURRENT_TIMESTAMP;

-- 4. 已有工作空间补齐交付物绑定（避免面板「未加载交付物类别」）
UPDATE task_project.project_workspace_entries
SET deliverable_system_id='ds_default_global', deliverable_system_from='company', updated_at=CURRENT_TIMESTAMP
WHERE company_id='{cid}' AND (deliverable_system_id IS NULL OR deliverable_system_id='');

-- 5. 无工作空间时创建默认工作空间（company-created intent 3）
INSERT IGNORE INTO task_project.project_workspace_entries(id, name, description, company_id, deliverable_system_id, deliverable_system_from, is_default, task_archive_tier, container_image_at_mode_enabled)
SELECT 'ws_e2e_default', 'E2E 默认工作空间', 'E2E 测试默认工作空间', '{cid}', 'ds_default_global', 'company', 1, '7d', 1
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM task_project.project_workspace_entries WHERE company_id='{cid}');

-- 6. 全部工作空间绑定默认进度体系（避免任务创建「进度列不属于当前工作空间」）
INSERT IGNORE INTO task_project.project_progress_systems_workspace(id, tenant_id, workspace_id, target_type, target_id)
SELECT CONCAT('psw_{cid}_', id), '{cid}', id, 'system', 'ps_default_system'
FROM task_project.project_workspace_entries WHERE company_id='{cid}';

-- 7. 成员 workspace_id 回填默认工作空间（便于 E2E 直达看板）
UPDATE task_tenant.tenant_company_member m
JOIN (SELECT id FROM task_project.project_workspace_entries WHERE company_id='{cid}' AND is_default=1 LIMIT 1) d
SET m.workspace_id = d.id
WHERE m.company_id='{cid}' AND (m.workspace_id IS NULL OR m.workspace_id='');
"""


def reset_superadmin_password(container: str, mysql_password: str) -> None:
    """将超管 author@example.com 的密码重置为已知 E2E 测试密码（幂等）。

    种子管理员密码已改为随机值（OPT-20260824-001，明文销毁），E2E 测试环境
    须显式重置为已知明文才能走登录流程。仅测试环境使用，勿用于生产。
    """
    sql = f"""
UPDATE task_auth.auth_login_method
SET password_hash = '{SUPERADMIN_BCRYPT_HASH}', updated_at = NOW()
WHERE LOWER(identifier) = '{SUPERADMIN_EMAIL}' AND method_type = 'email'
  AND binding_voided_at IS NULL;
"""
    mysql(container, mysql_password, sql)
    print(f"[seed] 超管 {SUPERADMIN_EMAIL} 密码已重置为 E2E 测试密码（仅测试环境）")


def main() -> None:
    # --superadmin 模式：仅重置超管密码，跳过公司默认体系
    if "--superadmin" in sys.argv:
        sys.argv.remove("--superadmin")
        container = sys.argv[1] if len(sys.argv) > 1 else "docker-mysql-mysql-1"
        mysql_password = sys.argv[2] if len(sys.argv) > 2 else "root123456"
        reset_superadmin_password(container, mysql_password)
        return

    container = sys.argv[1] if len(sys.argv) > 1 else "docker-mysql-mysql-1"
    password = sys.argv[2] if len(sys.argv) > 2 else "root123456"
    company_id = sys.argv[3] if len(sys.argv) > 3 else "850256677331562496"
    email = "contact@daydaymoney.com"
    bcrypt_hash = "$2a$12$LR3V2Zi5lb/s/nFMuOjA2eF5CyNiTRxDRs8mjhua7XKVAkDkvS0C."

    # 1. 幂等：账号已存在则跳过重建，但公司默认体系仍须补齐
    result = subprocess.run(
        [
            "docker", "exec", container, "mysql", "-uroot", f"-p{password}", "-N", "-B", "-e",
            f"SELECT COUNT(*) FROM task_auth.auth_login_method WHERE LOWER(identifier)='{email.lower()}'",
        ],
        capture_output=True, text=True,
    )
    account_exists = result.returncode == 0 and result.stdout.strip().startswith("1")
    if account_exists:
        print(f"[seed] E2E 账号 {email} 已存在，跳过账号重建")
    else:
        user_id = str(snowflake_id())
        lm_id = str(snowflake_id())
        member_id = str(snowflake_id())
        now = time.strftime("%Y-%m-%d %H:%M:%S")

        sql = f"""
SET @USER_ID := '{user_id}';
SET @LM_ID := '{lm_id}';
SET @MEMBER_ID := '{member_id}';
SET @COMPANY_ID := '{company_id}';
SET @PASS_HASH := '{bcrypt_hash}';
SET @NOW := '{now}';

INSERT INTO task_auth.auth_user (id, password, last_login, is_superuser, is_staff, is_active, is_tenant, date_joined)
VALUES (@USER_ID, '', NULL, 0, 0, 1, 0, @NOW);

INSERT INTO task_auth.auth_login_method (
    id, content_type_id, object_id, method_type, identifier,
    password_hash, is_verified, activation_token, activation_token_expires_at,
    created_at, updated_at
) VALUES (
    @LM_ID, 4, @USER_ID, 'email', '{email}',
    @PASS_HASH, 1, NULL, NULL, @NOW, @NOW
);

INSERT INTO task_tenant.tenant_company (id, name, creator_id, created_at, updated_at)
VALUES (@COMPANY_ID, 'E2E 测试公司', @USER_ID, @NOW, @NOW);

INSERT INTO task_tenant.tenant_company_member (
    id, user_id, company_id, is_admin, is_active, created_at, updated_at,
    invitation_token, invitation_token_expires_at, workspace_id, member_name, member_avatar
) VALUES (
    @MEMBER_ID, @USER_ID, @COMPANY_ID, 1, 1, @NOW, @NOW,
    NULL, NULL, '', 'E2E Tester', NULL
);
"""
        mysql(container, password, sql)
        print(f"[seed] 重建 E2E 账号 {email} (user={user_id}) 于公司 {company_id}")

    # 2. 补齐 COMPANY_CREATED 事件链产物（默认进度/交付物体系 + 默认工作空间 + 绑定）
    mysql(container, password, build_ensure_defaults_sql(company_id))
    print(f"[seed] 确保公司 {company_id} 默认进度/交付物体系与工作空间就绪")


if __name__ == "__main__":
    main()
