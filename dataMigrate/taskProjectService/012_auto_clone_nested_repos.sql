-- taskProjectService: project-level flag to auto-clone nested git repos on container bootstrap
-- DEFAULT 1 preserves existing enrich/clone behaviour for all rows.

ALTER TABLE project_entries
  ADD COLUMN auto_clone_nested_repos TINYINT NOT NULL DEFAULT 1
  COMMENT '1=container bootstrap merges .gitmodules nested repos into clone list';
