package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImageSkillIDBackfillMigration 验证 D1=B 回填迁移（041）与 Go 侧
// deriveImageSkillID 派生规则完全一致：seed = external_image_id（空回退 id），
// id = sk_<sha1(name+seed)>[:12]。任何一侧改动规则都会使本测试失败。
func TestImageSkillIDBackfillMigration(t *testing.T) {
	setupCloudTestDB(t)

	// 模拟存量行：image_skills_json 技能无 id（迁移前的形态）
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,image_skills_json,image_skills_extract_status,image_skills_digest)
		VALUES('img-bf','t1','ext-bf','img','registry.example/img:bf',?, 'ok', '')`,
		`{"version":1,"default_skill":"a","skills":[{"name":"a","is_default":true},{"name":"b"}]}`)
	if err != nil {
		t.Fatalf("seed row: %v", err)
	}

	// 执行 041 迁移文件正文（strip 注释头；单条 UPDATE）
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "dataMigrate", "taskCloudService", "041_image_skill_ids.sql"))
	if err != nil {
		t.Fatalf("read 041: %v", err)
	}
	var lines []string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		lines = append(lines, line)
	}
	if _, err := db.Exec(strings.Join(lines, "\n")); err != nil {
		t.Fatalf("run 041 backfill: %v", err)
	}

	var got string
	if err := db.QueryRow(`SELECT image_skills_json FROM cloud_tenant_installed_images WHERE id='img-bf'`).Scan(&got); err != nil {
		t.Fatalf("read back: %v", err)
	}
	list := decodeImageSkillsJSON(got)
	if len(list.Skills) != 2 || list.Skills[0].ID != deriveImageSkillID("a", "ext-bf") ||
		list.Skills[1].ID != deriveImageSkillID("b", "ext-bf") {
		t.Fatalf("backfilled ids mismatch Go rule: %+v", list.Skills)
	}

	// 幂等：重复执行结果不变
	if _, err := db.Exec(strings.Join(lines, "\n")); err != nil {
		t.Fatalf("re-run 041: %v", err)
	}
	var again string
	if err := db.QueryRow(`SELECT image_skills_json FROM cloud_tenant_installed_images WHERE id='img-bf'`).Scan(&again); err != nil {
		t.Fatalf("read back again: %v", err)
	}
	if again != got {
		t.Fatalf("041 not idempotent:\nfirst=%s\nagain=%s", got, again)
	}

	// external_image_id 为空的行回退 installed id 作为 seed
	_, err = db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,image_skills_json,image_skills_extract_status,image_skills_digest)
		VALUES('img-bf2','t1','','img2','registry.example/img:bf2',?, 'ok', '')`,
		`{"version":1,"default_skill":"c","skills":[{"name":"c","is_default":true}]}`)
	if err != nil {
		t.Fatalf("seed row 2: %v", err)
	}
	if _, err := db.Exec(strings.Join(lines, "\n")); err != nil {
		t.Fatalf("run 041 on row2: %v", err)
	}
	var got2 string
	if err := db.QueryRow(`SELECT image_skills_json FROM cloud_tenant_installed_images WHERE id='img-bf2'`).Scan(&got2); err != nil {
		t.Fatalf("read back row2: %v", err)
	}
	list2 := decodeImageSkillsJSON(got2)
	if len(list2.Skills) != 1 || list2.Skills[0].ID != deriveImageSkillID("c", "img-bf2") {
		t.Fatalf("fallback seed mismatch: %+v", list2.Skills)
	}
}
