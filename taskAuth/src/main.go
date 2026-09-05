package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"tracelog"
)

// parseBootstrapEmailInviteArgs 解析 bootstrap-email-invite CLI 参数。
// 支持 "--email <addr>" 与位置参数两种形式；返回邮箱或空串。
// 回归修复: 旧实现直接把 os.Args[2] 当邮箱，--email flag 被当作邮箱值。
func parseBootstrapEmailInviteArgs(args []string) string {
	email := ""
	for i := 2; i < len(args); i++ {
		if args[i] == "--email" {
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
			continue
		}
		if email == "" {
			email = args[i]
		}
	}
	return email
}

func main() {
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		log.Fatalf("[taskAuth] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		// Explicit migrate CLI only — business server must not auto-migrate (9999 / migrate.sh).
		if err := runDataMigrateFromDir(cfg.MySQLDSN, repoRoot); err != nil {
			log.Fatalf("[taskAuth] migrations: %v", err)
		}
		if err := openDB(cfg.MySQLDSN); err != nil {
			log.Fatalf("[taskAuth] mysql: %v", err)
		}
		defer db.Close()
		if err := RunGoDataMigrate(); err != nil {
			log.Fatalf("[taskAuth] go dataMigrate: %v", err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "bootstrap-admin" {
		if err := ensureBootstrapAdminSeeded(cfg.MySQLDSN, repoRoot); err != nil {
			log.Fatalf("[taskAuth] bootstrap-admin: %v", err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "bootstrap-email-invite" {
		if err := runBootstrapEmailInvite(cfg.MySQLDSN, repoRoot, parseBootstrapEmailInviteArgs(os.Args)); err != nil {
			log.Fatalf("[taskAuth] bootstrap-email-invite: %v", err)
		}
		return
	}

	tracelog.Init("task-auth")
	shutdownOtel, err := tracelog.InitOtel(context.Background(), "task-auth")
	if err != nil {
		log.Printf("[taskAuth] otel disabled: %v", err)
	} else {
		defer func() { _ = shutdownOtel(context.Background()) }()
	}

	if err := openDB(cfg.MySQLDSN); err != nil {
		log.Fatalf("[taskAuth] mysql: %v", err)
	}
	defer db.Close()

	if err := loadUserContentTypeID(); err != nil {
		log.Fatalf("[taskAuth] content_type: %v", err)
	}

	// Init OIDC signing key
	if err := initOidcSigningKey(cfg.OidcSigningKeyPath); err != nil {
		log.Fatalf("[taskAuth] oidc signing key: %v", err)
	}

	// 微信扫码 state 过期条目定时清扫（回调消费 + GC 双路径，防内存增长）
	startWechatStateGC()

	// 初始化共享 Kafka Writer（长连接 + 自动重连，替代每次新建 writer）
	initKafkaWriter()
	defer closeKafkaWriter()

	// 初始化 Membership Redis 客户端（可选，不可达时优雅降级为 rev=0）
	initMembershipRedis()

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[taskAuth] listening on %s", addr)
	handler := tracelog.Middleware(impersonationLogMiddleware(corsMiddleware(tracelog.MetricsMiddleware(mux))))
	if err := tracelog.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
