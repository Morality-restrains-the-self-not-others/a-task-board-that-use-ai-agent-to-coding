package infrastructure

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	dbload "dbload"

	_ "github.com/go-sql-driver/mysql"

	"taskCredentialService/domain"
)

// SQLiteBusinessRepository implements ports.BusinessDataRepository.
// Reads task/repo data from Go SSOT DBs (task_task / task_project).
// git_identities migrated from legacy saas.accounts_user_company_git_identity (OPT-052).
type SQLiteBusinessRepository struct {
	taskDB    *sql.DB
	projectDB *sql.DB
}

func NewSQLiteBusinessRepository(taskDB, projectDB *sql.DB) *SQLiteBusinessRepository {
	return &SQLiteBusinessRepository{taskDB: taskDB, projectDB: projectDB}
}

func openMySQLDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

// OpenBusinessDBs opens Go task/project DBs for business data reads.
// git_identities read from task_db (was legacy saas, OPT-052).
func OpenBusinessDBs(monorepoRoot string) (taskDB, projectDB *sql.DB, err error) {
	taskDSN, err := dbload.ResolveMySQLDSN("task-task", monorepoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("task-task DSN: %w", err)
	}
	projectDSN, err := dbload.ResolveMySQLDSN("task-project", monorepoRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("task-project DSN: %w", err)
	}

	taskDB, err = openMySQLDB(taskDSN)
	if err != nil {
		return nil, nil, err
	}
	projectDB, err = openMySQLDB(projectDSN)
	if err != nil {
		_ = taskDB.Close()
		return nil, nil, err
	}
	log.Printf("[task-credential-service] business dbs opened (mysql)")
	return taskDB, projectDB, nil
}

// FetchTaskSnapshot queries taskTaskService.tasks by ID.
// commentID is accepted for interface parity; task-level rows are returned
// (comment-scoped override only applies to the HTTP container-snapshot path).
func (r *SQLiteBusinessRepository) FetchTaskSnapshot(taskID, commentID string) (*domain.TaskSnapshot, error) {
	start := time.Now()
	row := r.taskDB.QueryRow(`
		SELECT id, title, COALESCE(description,''), workspace_id, COALESCE(tenant_id, ''), COALESCE(auto_run, 0), COALESCE(installed_image_id, '')
		FROM task_tasks
		WHERE id = ? LIMIT 1
	`, taskID)

	var s domain.TaskSnapshot
	var autoRunInt int
	err := row.Scan(&s.ID, &s.Title, &s.Description, &s.WorkspaceID, &s.CompanyID, &autoRunInt, &s.InstalledImageID)
	if err == nil {
		s.AutoRun = autoRunInt != 0
	}
	if err == sql.ErrNoRows {
		log.Printf("[task-credential-service] task snapshot not found: task=%s duration=%dms", taskID, time.Since(start).Milliseconds())
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch task snapshot: %w", err)
	}
	r.attachBranchStrategy(&s, taskID)
	log.Printf("[task-credential-service] task snapshot fetched: task=%s target_branch=%s duration=%dms",
		taskID, s.TargetBranch, time.Since(start).Milliseconds())
	return &s, nil
}

func (r *SQLiteBusinessRepository) attachBranchStrategy(s *domain.TaskSnapshot, taskID string) {
	if s == nil || r.taskDB == nil {
		return
	}
	var work, merge, target string
	err := r.taskDB.QueryRow(`
		SELECT COALESCE(work_branch_name,''), COALESCE(merge_target_branch_name,''), COALESCE(target_branch_name,'')
		FROM task_branch_strategies
		WHERE task_id = ?
		LIMIT 1
	`, taskID).Scan(&work, &merge, &target)
	if err != nil {
		return
	}
	work = strings.TrimSpace(work)
	merge = strings.TrimSpace(merge)
	target = strings.TrimSpace(target)
	s.BranchStrategy = &domain.BranchStrategySnapshot{
		WorkBranchName:        work,
		MergeTargetBranchName: merge,
		TargetBranchName:      target,
	}
	// 容器契约 task.target_branch = 共享工作分支
	if work != "" {
		s.TargetBranch = work
	} else if target != "" {
		s.TargetBranch = target
	}
}

// FetchTaskRepos queries task_projects (+ optional project name from task_project.db).
// commentID is accepted for interface parity; task-level rows are returned.
func (r *SQLiteBusinessRepository) FetchTaskRepos(taskID, commentID string) ([]domain.TaskRepoSnapshot, error) {
	start := time.Now()
	rows, err := r.taskDB.Query(`
		SELECT project_id,
			COALESCE(repo_address, ''),
			COALESCE(base_branch, ''),
			COALESCE(target_branch, '')
		FROM task_projects
		WHERE task_id = ?
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("fetch task repos: %w", err)
	}
	defer rows.Close()

	grouped := make(map[string]*domain.TaskRepoSnapshot)
	for rows.Next() {
		var projectID, repoURL, baseBranch, targetBranch string
		if err := rows.Scan(&projectID, &repoURL, &baseBranch, &targetBranch); err != nil {
			return nil, fmt.Errorf("scan task repo: %w", err)
		}
		repoURL = strings.TrimSpace(repoURL)
		if repoURL == "" {
			continue
		}
		baseBranch = strings.TrimSpace(baseBranch)
		targetBranch = strings.TrimSpace(targetBranch)
		snap, ok := grouped[projectID]
		if !ok {
			name, autoClone := r.lookupProjectMeta(projectID)
			snap = &domain.TaskRepoSnapshot{
				ProjectID:            projectID,
				ProjectName:          name,
				AutoCloneNestedRepos: autoClone,
			}
			grouped[projectID] = snap
		}
		snap.RepoURLs = append(snap.RepoURLs, repoURL)
		snap.RepoBranches = append(snap.RepoBranches, domain.RepoBranchEntry{
			GitRepo:      repoURL,
			BaseBranch:   baseBranch,
			TargetBranch: targetBranch,
		})
	}
	var result []domain.TaskRepoSnapshot
	autoCloneOff := 0
	for _, s := range grouped {
		result = append(result, *s)
		if !s.AutoCloneNestedRepos {
			autoCloneOff++
		}
	}
	log.Printf("[task-credential-service] task repos fetched: task=%s projects=%d repos=%d auto_clone_nested_off=%d duration=%dms",
		taskID, len(result), totalRepoCount(result), autoCloneOff, time.Since(start).Milliseconds())
	return result, nil
}

func (r *SQLiteBusinessRepository) lookupProjectMeta(projectID string) (name string, autoClone bool) {
	if r.projectDB == nil || strings.TrimSpace(projectID) == "" {
		return "", true
	}
	var autoCloneInt int64
	err := r.projectDB.QueryRow(
		`SELECT COALESCE(name,''), COALESCE(auto_clone_nested_repos,1) FROM project_entries WHERE id = ? LIMIT 1`,
		projectID,
	).Scan(&name, &autoCloneInt)
	if err != nil {
		log.Printf("[task-credential-service] project meta lookup miss project=%s err=%v", projectID, err)
		return "", true
	}
	return name, autoCloneInt != 0
}

// FetchCommentCreatedByUserID reads task_comments.created_by_id.
func (r *SQLiteBusinessRepository) FetchCommentCreatedByUserID(commentID string) (int64, error) {
	commentID = strings.TrimSpace(commentID)
	if r == nil || r.taskDB == nil || commentID == "" {
		return 0, nil
	}
	var raw string
	err := r.taskDB.QueryRow(
		`SELECT COALESCE(created_by_id, '') FROM task_comments WHERE id = ? LIMIT 1`,
		commentID,
	).Scan(&raw)
	if err == sql.ErrNoRows {
		log.Printf("[task-credential-service] comment author not found comment=%s", commentID)
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("fetch comment author: %w", err)
	}
	uid := parseUserID(raw)
	log.Printf("[task-credential-service] comment author fetched comment=%s user=%d", commentID, uid)
	return uid, nil
}

// FetchUserGitIdentities reads task_git_identities for a platform user.
func (r *SQLiteBusinessRepository) FetchUserGitIdentities(userID int64) ([]domain.GitIdentitySnapshot, error) {
	if r == nil || r.taskDB == nil || userID <= 0 {
		return nil, nil
	}
	rows, err := r.taskDB.Query(`
		SELECT id, COALESCE(user_id, ''), COALESCE(git_user_name, ''), COALESCE(git_user_email, '')
		FROM task_git_identities
		WHERE user_id = ?
	`, fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("fetch user git identities: %w", err)
	}
	defer rows.Close()
	var result []domain.GitIdentitySnapshot
	for rows.Next() {
		var s domain.GitIdentitySnapshot
		var uidRaw string
		if err := rows.Scan(&s.GitIdentityID, &uidRaw, &s.UserName, &s.UserEmail); err != nil {
			return nil, fmt.Errorf("scan user git identity: %w", err)
		}
		s.UserID = parseUserID(uidRaw)
		s.UserName = strings.TrimSpace(s.UserName)
		s.UserEmail = strings.TrimSpace(s.UserEmail)
		result = append(result, s)
	}
	log.Printf("[task-credential-service] user git identities fetched user=%d count=%d", userID, len(result))
	return result, nil
}

func (r *SQLiteBusinessRepository) lookupGitIdentityDetails(identityID string) (userID int64, userName, userEmail string) {
	if r.taskDB == nil || strings.TrimSpace(identityID) == "" {
		return 0, "", ""
	}
	var uidRaw sql.NullString
	var name, email sql.NullString
	err := r.taskDB.QueryRow(`
		SELECT COALESCE(user_id, ''), COALESCE(git_user_name, ''), COALESCE(git_user_email, '')
		FROM task_git_identities
		WHERE id = ?
		LIMIT 1
	`, identityID).Scan(&uidRaw, &name, &email)
	if err != nil {
		return 0, "", ""
	}
	if uidRaw.Valid {
		_, _ = fmt.Sscan(strings.TrimSpace(uidRaw.String), &userID)
	}
	if name.Valid {
		userName = strings.TrimSpace(name.String)
	}
	if email.Valid {
		userEmail = strings.TrimSpace(email.String)
	}
	return userID, userName, userEmail
}

func totalRepoCount(snapshots []domain.TaskRepoSnapshot) int {
	n := 0
	for _, s := range snapshots {
		n += len(s.RepoURLs)
	}
	return n
}
