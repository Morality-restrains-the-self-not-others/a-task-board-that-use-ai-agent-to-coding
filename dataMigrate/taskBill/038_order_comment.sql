-- billing_order_comment: 租户/系统管理员对资源订单的评论串
CREATE TABLE IF NOT EXISTS billing_order_comment (
    id bigint NOT NULL PRIMARY KEY,
    tenant_id bigint NOT NULL,
    order_id bigint NOT NULL,
    author_user_id varchar(64) NOT NULL,
    author_side varchar(32) NOT NULL,
    content text NOT NULL,
    created_at datetime NOT NULL,
    INDEX idx_billing_order_comment_order (order_id, tenant_id, created_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
