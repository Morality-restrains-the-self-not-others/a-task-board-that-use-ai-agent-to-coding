package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
	"tracelog"
)

var kafkaWriter *kafka.Writer

func initKafka() {
	if strings.TrimSpace(cfg.KafkaBootstrapServers) == "" {
		return
	}
	kafkaWriter = &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBootstrapServers),
		Topic:        "domain-events",
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true,
	}
}

func closeKafka() {
	if kafkaWriter != nil {
		_ = kafkaWriter.Close()
	}
}

func main() {
	const svc = "task-ai-comment"
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		tracelog.Fatalf(svc, "[taskAIComment] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	initKafka()
	defer closeKafka()
	tracelog.Init(svc)

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := openDB(cfg.MySQLDSN); err != nil {
			tracelog.Fatalf(svc, "[taskAIComment] migration: %v", err)
		}
		if err := runDataMigrate(repoRoot); err != nil {
			tracelog.Fatalf(svc, "[taskAIComment] dataMigrate: %v", err)
		}
		log.Println("[taskAIComment] migration complete")
		return
	}

	if err := openDB(cfg.MySQLDSN); err != nil {
		tracelog.Fatalf(svc, "[taskAIComment] db: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[taskAIComment] listening on %s", addr)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(corsMiddleware(authMiddleware(mux)))); err != nil {
		tracelog.Fatalf(svc, "[taskAIComment] listen: %v", err)
	}
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("/api/schema/", handleOpenAPISchema)
	mux.HandleFunc("/api/schema", handleOpenAPISchema)
	mux.HandleFunc("/api/swagger/", handleSwaggerUI)
	mux.HandleFunc("/api/swagger", handleSwaggerUI)
	mux.HandleFunc("/api/internal/task-ai-comment/", handleInternalRoutes)

	// 网关前缀 /api/ai-comment/* → taskAIComment（routes.yaml task-ai-comment）：
	//   /api/ai-comment/task-detail/tenant_id/{tid}/workspace_id/{wid}/task_id/{taskId}/ai-comments[/{commentId}]
	//   /api/ai-comment/container-agent-comments/tenant_id/{tid}/workspace_id/{wid}/task_id/{taskId}[/{id}[/{action}]]
	// 须用 gatewayauth.ParseConventionPath：kv-last 后仅剩 1 个位置段（如 ai-comments）时
	// 旧手写循环会丢 rest → HTTP 404（页面「ai_comments：HTTP 404」）。
	mux.HandleFunc("/api/ai-comment/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/ai-comment/")
		route, err := parseAICommentGatewayRoute(r, path)
		if err != nil {
			if errors.Is(err, errAICommentGatewayNotFound) {
				writeError(w, r, http.StatusNotFound, "not found")
				return
			}
			writeError(w, r, http.StatusBadRequest, "tenant_id/workspace_id/task_id required")
			return
		}
		funcName := route.FuncName
		tenantID := route.TenantID
		workspaceID := route.WorkspaceID
		taskID := route.TaskID
		rest := route.Rest
		switch funcName {
		case "task-detail":
			commentID := ""
			if rest == "ai-comments" {
				commentID = ""
			} else if strings.HasPrefix(rest, "ai-comments/") {
				commentID = strings.Trim(strings.TrimPrefix(rest, "ai-comments/"), "/")
			} else {
				writeError(w, r, http.StatusNotFound, "not found")
				return
			}
			if commentID != "" {
				switch r.Method {
				case http.MethodPatch:
					handlePatchAIComment(w, r, tenantID, workspaceID, taskID, commentID, getAuthUser(r))
				default:
					writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
				}
				return
			}
			switch r.Method {
			case http.MethodGet:
				handleListAIComments(w, r, tenantID, workspaceID, taskID, getAuthUser(r))
			case http.MethodPost:
				handleCreateAIComment(w, r, tenantID, workspaceID, taskID, getAuthUser(r))
			default:
				writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
			}
		case "container-agent-comments":
			segs := []string{}
			if rest != "" {
				segs = strings.Split(rest, "/")
			}
			if len(segs) == 0 {
				switch r.Method {
				case http.MethodGet:
					handleListContainerAgentComments(w, r, tenantID, workspaceID, taskID)
				case http.MethodPost:
					handlePublicCreateContainerAgentComment(w, r, tenantID, workspaceID, taskID)
				default:
					writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
				}
				return
			}
			id := segs[0]
			if len(segs) == 1 {
				if r.Method == http.MethodGet {
					handleGetContainerAgentComment(w, r, tenantID, workspaceID, taskID, id)
					return
				}
				writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			switch segs[1] {
			case "stream":
				handleContainerAgentStream(w, r, tenantID, workspaceID, taskID, id)
			case "complete":
				handleContainerAgentComplete(w, r, tenantID, workspaceID, taskID, id)
			case "fail":
				handleContainerAgentFail(w, r, tenantID, workspaceID, taskID, id)
			default:
				writeError(w, r, http.StatusNotFound, "not found")
			}
		default:
			writeError(w, r, http.StatusNotFound, "not found")
		}
	})

	mux.HandleFunc("/api/tenant/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/tenant/")
		if parts == nil {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		if len(parts) >= 6 && parts[1] == "workspace" && parts[3] == "task-detail" && parts[5] == "ai-comments" {
			handleAICommentRoutes(w, r)
			return
		}
		if len(parts) >= 6 && parts[1] == "workspace" && parts[3] == "task" && parts[5] == "container-agent-comments" {
			handleContainerAgentCommentRoutes(w, r)
			return
		}
		writeError(w, r, http.StatusNotFound, "not found")
	})
}
