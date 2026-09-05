package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	dbload "dbload"
	"taskEvents/internal/cloud/aliyun"
)

func main() {
	platformIP := os.Getenv("PLATFORM_IP")
	if platformIP == "" {
		fmt.Fprintln(os.Stderr, "PLATFORM_IP env var is required")
		os.Exit(1)
	}
	taskID := "task_13165596112687184871"
	if v := os.Getenv("TASK_ID"); v != "" {
		taskID = v
	}

	monorepoRoot, err := dbload.FindMonorepoRoot("")
	if err != nil {
		panic(fmt.Errorf("monorepo root: %w", err))
	}
	dsn, err := dbload.ResolveMySQLDSN("task-cloud", monorepoRoot)
	if err != nil {
		panic(fmt.Errorf("resolve MySQL DSN: %w", err))
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var region, sgID, authID string
	if err := db.QueryRow(
		`SELECT region, security_group_id, authorization_id FROM cloud_server_configs WHERE task_id=?`,
		taskID,
	).Scan(&region, &sgID, &authID); err != nil {
		panic(err)
	}
	var ak, sk string
	if err := db.QueryRow(
		`SELECT secret_id, secret_key FROM cloud_platform_authorizations WHERE id=?`,
		authID,
	).Scan(&ak, &sk); err != nil {
		panic(err)
	}
	fmt.Printf("authorizing sg=%s region=%s platform_ip=%s task=%s\n", sgID, region, platformIP, taskID)
	if err := aliyun.AuthorizeServerPublicIPIngress(ak, sk, region, sgID, platformIP); err != nil {
		panic(err)
	}
	fmt.Println("ok")
}
