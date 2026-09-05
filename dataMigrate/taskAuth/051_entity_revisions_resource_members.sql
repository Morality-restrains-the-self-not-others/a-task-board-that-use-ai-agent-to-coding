-- Register task/project revision GET APIs on existing work-panel / projects regions (ADR-0051).
-- No new roles; read history == read live entity.

INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('rm-wp-task-rev-list', 'rg-reg-nav-work-panel-main', 'api', 'GET /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}/revisions/'),
('rm-wp-task-rev-get', 'rg-reg-nav-work-panel-main', 'api', 'GET /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}/revisions/{revisionId}/'),
('rm-proj-rev-list', 'rg-reg-nav-projects-main', 'api', 'GET /api/projects/tenant_id/{tid}/{projectId}/revisions/'),
('rm-proj-rev-get', 'rg-reg-nav-projects-main', 'api', 'GET /api/projects/tenant_id/{tid}/{projectId}/revisions/{revisionId}/');
