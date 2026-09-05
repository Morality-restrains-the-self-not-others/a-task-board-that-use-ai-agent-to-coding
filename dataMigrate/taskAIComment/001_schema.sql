-- taskAIComment: Core schema
-- Tables: ai_comment_task_comments, ai_comment_container_agent_comments
-- Prefix: ai_comment_ (conforms to service prefix convention)

CREATE TABLE IF NOT EXISTS ai_comment_task_comments (
    id VARCHAR(64) PRIMARY KEY,
    content TEXT NOT NULL,
    assistant_response TEXT,
    task_id VARCHAR(64) NOT NULL,
    tenant_id VARCHAR(64) NOT NULL DEFAULT '',
    workspace_id VARCHAR(64) NOT NULL DEFAULT '',
    created_by_id VARCHAR(64) NOT NULL,
    execution_mode VARCHAR(64) NOT NULL DEFAULT 'wait_previous',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_ai_comment_task_comments_task ON ai_comment_task_comments(task_id);
CREATE INDEX idx_ai_comment_task_comments_tenant ON ai_comment_task_comments(tenant_id);

CREATE TABLE IF NOT EXISTS ai_comment_container_agent_comments (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    task_id VARCHAR(64) NOT NULL,
    parent_comment_id VARCHAR(64) NOT NULL,
    installed_image_id VARCHAR(64) NOT NULL,
    run_status VARCHAR(64) NOT NULL DEFAULT 'pending',
    content TEXT,
    assistant_response TEXT,
    context_pack_json TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE INDEX idx_ai_comment_container_agent_comments_task ON ai_comment_container_agent_comments(task_id);
CREATE INDEX idx_ai_comment_container_agent_comments_parent ON ai_comment_container_agent_comments(parent_comment_id);
