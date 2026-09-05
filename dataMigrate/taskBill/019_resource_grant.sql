-- billing_resource_grant: 管理员/系统赠送资源记录（支持过期时间）
CREATE TABLE IF NOT EXISTS billing_resource_grant (
    id bigint NOT NULL PRIMARY KEY,
    tenant_id bigint NOT NULL,
    resource_type varchar(32) NOT NULL,
    quantity integer NOT NULL CHECK (quantity >= 1),
    reason text NOT NULL DEFAULT (''),
    expires_at datetime NULL,
    created_at datetime NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX bill_res_grant_tenant ON billing_resource_grant (tenant_id, resource_type);
