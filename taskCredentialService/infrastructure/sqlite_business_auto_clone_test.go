package infrastructure

import (
	"testing"
)

func TestSQLiteBusinessRepository_FetchTaskReposReadsAutoCloneNestedRepos(t *testing.T) {
	db := setupInfraTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE task_projects (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) NOT NULL,
			project_id VARCHAR(64) NOT NULL,
			base_branch VARCHAR(255) DEFAULT '',
			target_branch VARCHAR(255) DEFAULT '',
			repo_address VARCHAR(512) DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		CREATE TABLE project_entries (
			id VARCHAR(64) PRIMARY KEY,
			name TEXT NOT NULL,
			auto_clone_nested_repos TINYINT NOT NULL DEFAULT 1
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO project_entries(id, name, auto_clone_nested_repos) VALUES
			('proj_off', 'OffProj', 0),
			('proj_on', 'OnProj', 1);
		INSERT INTO task_projects(id, task_id, project_id, repo_address) VALUES
			('tp1', 'task_auto_clone', 'proj_off', 'https://git.example/parent.git'),
			('tp2', 'task_auto_clone', 'proj_on', 'https://git.example/other.git');
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewSQLiteBusinessRepository(db, db)
	repos, err := repo.FetchTaskRepos("task_auto_clone", "")
	if err != nil {
		t.Fatalf("FetchTaskRepos: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("projects=%d want 2: %+v", len(repos), repos)
	}
	byID := map[string]bool{}
	names := map[string]string{}
	for _, r := range repos {
		byID[r.ProjectID] = r.AutoCloneNestedRepos
		names[r.ProjectID] = r.ProjectName
	}
	if names["proj_off"] != "OffProj" {
		t.Fatalf("proj_off name=%q want OffProj (must read project_entries, not legacy projects)", names["proj_off"])
	}
	if names["proj_on"] != "OnProj" {
		t.Fatalf("proj_on name=%q want OnProj", names["proj_on"])
	}
	if byID["proj_off"] {
		t.Fatalf("proj_off auto_clone_nested_repos must be false, got %+v", repos)
	}
	if !byID["proj_on"] {
		t.Fatalf("proj_on auto_clone_nested_repos must be true, got %+v", repos)
	}
}
