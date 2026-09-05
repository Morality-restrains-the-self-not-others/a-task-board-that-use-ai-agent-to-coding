package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"taskAiProvider/infrastructure"
)

// platformStaffTable 是 ai_provider platform staff 的当前表名（表前缀迁移前旧名
// marketplace_platformstaff 已被 ai_provider_platformstaff 取代）。保持单点定义，
// 供 staff 分支 SELECT/INSERT 使用，并在 main_test.go 中断言防回退。
const platformStaffTable = "ai_provider_platformstaff"
const vendorTable = "ai_provider_vendor"

func mysqlConnectDSN(path string) string {
	return strings.TrimSpace(path)
}

// staffLookupSQL 返回查找可用 staff 的 SQL（minttoken staff 复用现有行）。
func staffLookupSQL() string {
	return "SELECT id FROM " + platformStaffTable + " WHERE is_active=1 ORDER BY id LIMIT 1"
}

// staffInsertSQL 返回新建 playwright_staff 的 SQL（与 infrastructure/store_auth.go 同构）。
func staffInsertSQL() string {
	return "INSERT INTO " + platformStaffTable +
		" (id, username, password_hash, display_name, is_active, created_at, updated_at, saas_superadmin_id)" +
		" VALUES (?,?,?,?,1,?,?,NULL)"
}

func vendorLookupSQL() string {
	return "SELECT id FROM " + vendorTable + " WHERE lower(email)=lower(?)"
}

func vendorInsertSQL() string {
	return "INSERT INTO " + vendorTable +
		" (id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at, saas_user_id)" +
		" VALUES (?,?,?,?,?,1,?,?,NULL)"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: minttoken staff | minttoken vendor [email] [company]")
		os.Exit(1)
	}
	root, err := infrastructure.FindMonorepoRoot()
	if err != nil {
		panic(err)
	}
	cfg, err := infrastructure.LoadConfig(root)
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("mysql", mysqlConnectDSN(cfg.DatabasePath))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	switch os.Args[1] {
	case "staff":
		var id int64
		err := db.QueryRow(staffLookupSQL()).Scan(&id)
		if err == sql.ErrNoRows {
			id = infrastructure.NextID()
			now := "2026-01-01 00:00:00.000000"
			_, err = db.Exec(staffInsertSQL(), id, "playwright_staff", infrastructure.RandomPasswordHash(), "Playwright Staff", now, now)
			if err != nil {
				panic(err)
			}
		} else if err != nil {
			panic(err)
		}
		tok, err := infrastructure.IssueToken(cfg.SecretKey, fmt.Sprintf("%d", id), "staff", cfg.JWTTTLSeconds)
		if err != nil {
			panic(err)
		}
		fmt.Println(tok)
	case "vendor":
		email := "playwright-vendor@example.com"
		company := "Playwright Vendor"
		if len(os.Args) >= 3 && strings.TrimSpace(os.Args[2]) != "" {
			email = strings.TrimSpace(os.Args[2])
		}
		if len(os.Args) >= 4 && strings.TrimSpace(os.Args[3]) != "" {
			company = strings.TrimSpace(os.Args[3])
		}
		var id int64
		err := db.QueryRow(vendorLookupSQL(), email).Scan(&id)
		if err == sql.ErrNoRows {
			id = infrastructure.NextID()
			now := "2026-01-01 00:00:00.000000"
			_, err = db.Exec(vendorInsertSQL(), id, email, infrastructure.RandomPasswordHash(), company, "pw", now, now)
			if err != nil {
				panic(err)
			}
		} else if err != nil {
			panic(err)
		}
		tok, err := infrastructure.IssueToken(cfg.SecretKey, fmt.Sprintf("%d", id), "vendor", cfg.JWTTTLSeconds)
		if err != nil {
			panic(err)
		}
		fmt.Println(tok)
	default:
		fmt.Fprintln(os.Stderr, "unknown type", os.Args[1])
		os.Exit(1)
	}
}
