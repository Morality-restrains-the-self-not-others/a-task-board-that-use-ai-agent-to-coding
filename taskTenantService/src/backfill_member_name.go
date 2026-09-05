package main

import (
	"log"
	"strconv"
	"strings"
)

// runMemberNameBackfill 一次性运维命令：回填误种子的 member_name（OPT-20260812-037）。
//
// 历史路径（USER_CREATED / onboarding create）曾把公司名写入 tenant_company_member.member_name，
// 读路径虽有自愈（resolveMemberDisplayName / healMisSeededMemberName），但未访问人员页的租户
// 行仍残留误种子。本命令扫描 member_name 为「我的公司」/ 空 / 等于公司名的行，批量取
// taskAuth auth_user_profile.username 回填。
//
// 用法: taskTenantService backfill-member-name [--dry-run] [--limit=N]
//   --dry-run   只打印将更新的行，不写库
//   --limit=N   单次扫描上限（默认 5000）
func runMemberNameBackfill(args []string) {
	dryRun := false
	limit := 5000
	for _, a := range args {
		switch {
		case a == "--dry-run":
			dryRun = true
		case strings.HasPrefix(a, "--limit="):
			if n, err := strconv.Atoi(strings.TrimPrefix(a, "--limit=")); err == nil && n > 0 {
				limit = n
			}
		}
	}
	if err := openDB(cfg.DBPath); err != nil {
		log.Fatalf("[taskTenantService] backfill-member-name: open db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT m.id, m.user_id, COALESCE(m.member_name,''), COALESCE(c.name,'')
		FROM tenant_company_member m
		JOIN tenant_company c ON c.id = m.company_id
		WHERE COALESCE(m.member_name,'') = ? OR (c.name <> '' AND COALESCE(m.member_name,'') = c.name)
		LIMIT ?`, defaultPersonalCompanyName, limit)
	if err != nil {
		log.Fatalf("[taskTenantService] backfill-member-name: query: %v", err)
	}
	defer rows.Close()

	type targetRow struct {
		id          string
		userID      string
		memberName  string
		companyName string
	}
	var targets []targetRow
	userIDs := make([]string, 0)
	for rows.Next() {
		var r targetRow
		if err := rows.Scan(&r.id, &r.userID, &r.memberName, &r.companyName); err != nil {
			log.Fatalf("[taskTenantService] backfill-member-name: scan: %v", err)
		}
		if !isMisSeededMemberName(r.memberName, r.companyName) {
			continue
		}
		targets = append(targets, r)
		userIDs = append(userIDs, r.userID)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("[taskTenantService] backfill-member-name: rows: %v", err)
	}

	if len(targets) == 0 {
		log.Printf("[taskTenantService] backfill-member-name: no mis-seeded rows found")
		return
	}

	nicknames := fetchPersonalNicknamesFromAuth(userIDs)

	updated := 0
	skipped := 0
	for _, r := range targets {
		nick := strings.TrimSpace(nicknames[r.userID])
		if nick == "" {
			skipped++
			log.Printf("[taskTenantService] backfill-member-name: member=%s skip (no nickname) user=%s current=%q", r.id, r.userID, r.memberName)
			continue
		}
		if nick == r.memberName {
			skipped++
			continue
		}
		if dryRun {
			log.Printf("[taskTenantService] backfill-member-name: member=%s WOULD update %q -> %q (user=%s)", r.id, r.memberName, nick, r.userID)
			updated++
			continue
		}
		healMisSeededMemberName(r.id, nick)
		log.Printf("[taskTenantService] backfill-member-name: member=%s updated %q -> %q (user=%s)", r.id, r.memberName, nick, r.userID)
		updated++
	}
	log.Printf("[taskTenantService] backfill-member-name: done dryRun=%v targets=%d updated=%d skipped=%d", dryRun, len(targets), updated, skipped)
}
