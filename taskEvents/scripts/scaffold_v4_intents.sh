#!/usr/bin/env bash
# Scaffold v4 cmd/{event}/{intent}/ and bin/{event}/{intent}/ from config registry.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

python3 << 'PY'
import os
from pathlib import Path

ROOT = Path(".")

INTENTS = [
    ("billing_transaction_created", "1_process_billing_transaction", "billing"),
    ("sse_message", "1_send_sse_message", "sse"),
    ("email_sent", "1_send_email", "email"),
    ("invitation_created", "1_send_invitation_email", "invitation"),
    ("user_activated", "1_send_welcome_notification", "user_activated_notif"),
    ("user_created", "0_create_company", "user_created"),
    ("user_created", "1_send_welcome_email", "user_welcome_stub"),
    ("company_created", "1_set_default_deliverable_system", "company_deliverable"),
    ("company_created", "2_set_default_progress_system", "company_progress"),
    ("company_created", "3_create_default_workspace", "company_workspace"),
    ("workspace_created", "1_process_workspace_creation", "workspace"),
    ("project_updated", "1_process_project_update", "project"),
    ("task_completed", "1_process_task_completion", "task"),
    ("ai_assistant_reply_completed", "1_persist_assistant_reply", "ai"),
    ("cloud_server_started", "1_process_server_start", "cloud_started"),
    ("cloud_server_stopped", "1_process_server_stop", "cloud_stopped"),
    ("cloud_server_start_auto", "1_process_server_start_auto", "cloud_start_auto"),
    ("cloud_platform_authorization_created", "1_process_cloud_platform_authorization", "cloud_platform"),
]

TEMPLATES = {
    "billing": '''package main

import (
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/billing"
)

func main() {
	eventbin.RunIntent("billing_transaction_created", "1_process_billing_transaction", &billing.Handler{}, consumer.IdempotencyKeyFromEnvelope)
}
''',
    "company_deliverable": '''package main

import (
	"log"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/companycreated"
	"taskEvents/internal/repository/saas"
)

func main() {
	_, _, root, err := config.LoadIntent("company_created", "1_set_default_deliverable_system")
	if err != nil {
		log.Fatal(err)
	}
	repo, err := saas.Open(root)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()
	eventbin.RunIntent("company_created", "1_set_default_deliverable_system", &companycreated.DeliverableIntent{Repo: repo}, consumer.IdempotencyKeyFromEnvelope)
}
''',
    "company_progress": '''package main

import (
	"log"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/companycreated"
	"taskEvents/internal/repository/saas"
)

func main() {
	_, _, root, err := config.LoadIntent("company_created", "2_set_default_progress_system")
	if err != nil {
		log.Fatal(err)
	}
	repo, err := saas.Open(root)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()
	eventbin.RunIntent("company_created", "2_set_default_progress_system", &companycreated.ProgressIntent{Repo: repo}, consumer.IdempotencyKeyFromEnvelope)
}
''',
    "company_workspace": '''package main

import (
	"log"

	"taskEvents/config"
	"taskEvents/consumer"
	"taskEvents/eventbin"
	"taskEvents/internal/handlers/companycreated"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

func main() {
	appCfg, _, root, err := config.LoadIntent("company_created", "3_create_default_workspace")
	if err != nil {
		log.Fatal(err)
	}
	repo, err := saas.Open(root)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()
	pub := publish.NewFromConfig(appCfg)
	defer pub.Close()
	eventbin.RunIntent("company_created", "3_create_default_workspace", &companycreated.WorkspaceIntent{Repo: repo, Publisher: pub}, consumer.IdempotencyKeyFromEnvelope)
}
''',
}

# For intents not in TEMPLATES, copy from legacy cmd/event/{event}/main.go if exists
for event, intent, kind in INTENTS:
    cmd_dir = ROOT / "cmd" / event / intent
    bin_dir = ROOT / "bin" / event / intent
    cmd_dir.mkdir(parents=True, exist_ok=True)
    bin_dir.mkdir(parents=True, exist_ok=True)
    main_path = cmd_dir / "main.go"
    if kind in TEMPLATES:
        main_path.write_text(TEMPLATES[kind])
    else:
        legacy = ROOT / "cmd" / event / intent / "main.go"
        if not main_path.exists():
            main_path.write_text(f'package main\n\nfunc main() {{}}\n')
    print(f"scaffolded {event}/{intent}")
PY

echo "v4 layout scaffold done."
