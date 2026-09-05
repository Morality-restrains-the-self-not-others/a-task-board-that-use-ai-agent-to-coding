-- taskTenantService: Core schema
-- Tables: tenant_company_member, tenant_invitation, tenant_company_group,
--          tenant_company_group_member, tenant_company

CREATE TABLE IF NOT EXISTS tenant_company_member (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    is_admin TINYINT NOT NULL DEFAULT 0,
    is_active TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    invitation_token VARCHAR(255),
    invitation_token_expires_at DATETIME,
    workspace_id VARCHAR(64),
    member_name VARCHAR(255),
    member_avatar VARCHAR(512),
    UNIQUE(user_id, company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_acm_company ON tenant_company_member(company_id);
CREATE INDEX idx_acm_user ON tenant_company_member(user_id);
CREATE UNIQUE INDEX idx_acm_invite_token ON tenant_company_member(invitation_token);

CREATE TABLE IF NOT EXISTS tenant_invitation (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    is_admin TINYINT NOT NULL DEFAULT 0,
    workspace_id VARCHAR(64) NOT NULL DEFAULT '',
    invite_method VARCHAR(64) NOT NULL DEFAULT 'link',
    invite_target VARCHAR(255) NOT NULL DEFAULT '',
    invitation_token VARCHAR(255),
    invitation_token_expires_at DATETIME,
    is_accepted TINYINT NOT NULL DEFAULT 0,
    company_member_name VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_inv_company ON tenant_invitation(company_id);
CREATE UNIQUE INDEX idx_inv_token ON tenant_invitation(invitation_token);

CREATE TABLE IF NOT EXISTS tenant_company_group (
    id VARCHAR(64) PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    company_id VARCHAR(64) NOT NULL,
    created_by_id VARCHAR(64),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_acg_company ON tenant_company_group(company_id);

CREATE TABLE IF NOT EXISTS tenant_company_group_member (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_acgm_group ON tenant_company_group_member(group_id);
CREATE INDEX idx_acgm_user ON tenant_company_group_member(user_id);

CREATE TABLE IF NOT EXISTS tenant_company (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '',
    creator_id VARCHAR(64) NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_ac_creator ON tenant_company(creator_id);
