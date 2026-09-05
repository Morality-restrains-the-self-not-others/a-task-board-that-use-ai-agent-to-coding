package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

type autoRunStepsExtractor interface {
	Extract(imageURL string) infrastructure.AutoRunStepsExtractResult
}

type defaultAutoRunExtractor struct{}

func (defaultAutoRunExtractor) Extract(imageURL string) infrastructure.AutoRunStepsExtractResult {
	return infrastructure.ExtractAutoRunStepsFromImage(imageURL)
}

type noopAutoRunExtractor struct{}

func (noopAutoRunExtractor) Extract(string) infrastructure.AutoRunStepsExtractResult {
	return infrastructure.AutoRunStepsExtractResult{Status: "not_found"}
}

// autoRunAndSkillsExtractor walks the image layers once and returns both
// autoRunStep.md and imageSkills.yaml (OPT-20260821-018). Production uses the
// default single-pass walk; tests inject a stub to assert one walk per save.
type autoRunAndSkillsExtractor interface {
	Extract(imageURL string) infrastructure.AutoRunAndSkillsExtractResult
}

type defaultAutoRunAndSkillsExtractor struct{}

func (defaultAutoRunAndSkillsExtractor) Extract(imageURL string) infrastructure.AutoRunAndSkillsExtractResult {
	return infrastructure.ExtractAutoRunAndSkillsFromImage(imageURL)
}

var extractInflight sync.Map
var combinedExtractInflight sync.Map

func (a *App) autoRunExtractor() autoRunStepsExtractor {
	if a != nil && a.AutoRunExtractor != nil {
		return a.AutoRunExtractor
	}
	return defaultAutoRunExtractor{}
}

// autoRunAndSkillsExtractor returns the combined single-pass extractor when no
// individual extractor is injected (production). Tests that stub
// AutoRunExtractor / SkillsExtractor get nil so the combined runner falls back
// to calling them separately — keeping existing per-file stub assertions valid.
func (a *App) autoRunAndSkillsExtractor() autoRunAndSkillsExtractor {
	if a != nil && a.AutoRunAndSkillsExtractor != nil {
		return a.AutoRunAndSkillsExtractor
	}
	if a != nil && a.AutoRunExtractor == nil && a.SkillsExtractor == nil {
		return defaultAutoRunAndSkillsExtractor{}
	}
	return nil
}

func (a *App) scheduleAutoRunExtract(ctx context.Context, id int64, imageURL, version string) {
	if id == 0 || strings.TrimSpace(imageURL) == "" || a == nil || a.DB == nil {
		return
	}
	if _, loaded := extractInflight.LoadOrStore(id, true); loaded {
		return
	}
	run := func() {
		defer extractInflight.Delete(id)
		a.runAutoRunExtract(ctx, id, imageURL, version)
	}
	if a.ExtractSync {
		run()
		return
	}
	go run()
}

func (a *App) runAutoRunExtract(ctx context.Context, id int64, imageURL, version string) {
	merged, err := infrastructure.MergeImageURLWithVersion(imageURL, version)
	if err != nil {
		logWarn(ctx, "event=ContainerImageAutoRunStepsExtractFailed id=%d err=%s", id, err.Error())
		if dbErr := a.DB.UpdateContainerImageFields(id, map[string]any{
			"auto_run_steps_md":             "",
			"auto_run_steps_extract_status": "failed",
			"auto_run_steps_digest":         "",
		}); dbErr != nil {
			logError(ctx, "event=ContainerImageAutoRunStepsExtractPersistFailed id=%d err=%s", id, dbErr.Error())
		}
		return
	}
	start := time.Now()
	if err := a.DB.UpdateContainerImageFields(id, map[string]any{
		"auto_run_steps_extract_status": "pending",
	}); err != nil {
		logError(ctx, "event=ContainerImageAutoRunStepsExtractPendingFailed id=%d err=%s", id, err.Error())
		return
	}
	logInfo("event=ContainerImageAutoRunStepsExtractStarted id=%d", id)
	result := a.autoRunExtractor().Extract(merged)
	a.persistAutoRunSteps(ctx, id, result, start)
}

// persistAutoRunSteps writes one auto_run_steps_* extraction outcome and emits
// the corresponding event. Shared by the single-file and combined runners.
func (a *App) persistAutoRunSteps(ctx context.Context, id int64, result infrastructure.AutoRunStepsExtractResult, start time.Time) {
	if err := a.DB.UpdateContainerImageFields(id, map[string]any{
		"auto_run_steps_md":             result.Markdown,
		"auto_run_steps_extract_status": result.Status,
		"auto_run_steps_digest":         result.Digest,
	}); err != nil {
		logError(ctx, "event=ContainerImageAutoRunStepsExtractPersistFailed id=%d err=%s", id, err.Error())
		return
	}
	durationMS := time.Since(start).Milliseconds()
	if result.Status == "ok" {
		logInfo("event=ContainerImageAutoRunStepsExtracted id=%d status=%s duration_ms=%d", id, result.Status, durationMS)
	} else {
		logWarn(ctx, "event=ContainerImageAutoRunStepsExtracted id=%d status=%s detail=%s duration_ms=%d", id, result.Status, result.Detail, durationMS)
	}
	_ = a.eventBus().Publish(context.Background(), domain.EventContainerImageAutoRunStepsExtracted, map[string]any{
		"image_id": infrastructure.IDStr(id),
		"status":   result.Status,
		"digest":   result.Digest,
	})
}

// scheduleAutoRunAndSkillsExtract extracts both autoRunStep.md and
// imageSkills.yaml in one registry walk (OPT-20260821-018). It replaces the
// pair of scheduleAutoRunExtract + scheduleImageSkillsExtract calls that used
// to double the manifest/blobs fetched per image save.
func (a *App) scheduleAutoRunAndSkillsExtract(ctx context.Context, id int64, imageURL, version string) {
	if id == 0 || strings.TrimSpace(imageURL) == "" || a == nil || a.DB == nil {
		return
	}
	if _, loaded := combinedExtractInflight.LoadOrStore(id, true); loaded {
		return
	}
	run := func() {
		defer combinedExtractInflight.Delete(id)
		a.runAutoRunAndSkillsExtract(ctx, id, imageURL, version)
	}
	if a.ExtractSync {
		run()
		return
	}
	go run()
}

func (a *App) runAutoRunAndSkillsExtract(ctx context.Context, id int64, imageURL, version string) {
	merged, err := infrastructure.MergeImageURLWithVersion(imageURL, version)
	if err != nil {
		logWarn(ctx, "event=ContainerImageExtractFailed id=%d err=%s", id, err.Error())
		if dbErr := a.DB.UpdateContainerImageFields(id, map[string]any{
			"auto_run_steps_md":             "",
			"auto_run_steps_extract_status": "failed",
			"auto_run_steps_digest":         "",
			"image_skills_json":             "",
			"image_skills_extract_status":   "failed",
			"image_skills_digest":           "",
		}); dbErr != nil {
			logError(ctx, "event=ContainerImageExtractPersistFailed id=%d err=%s", id, dbErr.Error())
		}
		return
	}
	start := time.Now()
	if err := a.DB.UpdateContainerImageFields(id, map[string]any{
		"auto_run_steps_extract_status": "pending",
		"image_skills_extract_status":   "pending",
	}); err != nil {
		logError(ctx, "event=ContainerImageExtractPendingFailed id=%d err=%s", id, err.Error())
		return
	}
	logInfo("event=ContainerImageExtractStarted id=%d", id)
	if ext := a.autoRunAndSkillsExtractor(); ext != nil {
		result := ext.Extract(merged)
		a.persistAutoRunSteps(ctx, id, result.AutoRun, start)
		a.persistImageSkills(ctx, id, result.Skills, start)
		return
	}
	// Tests stub AutoRunExtractor / SkillsExtractor separately; fall back to the
	// two single-file walks so per-file stub assertions keep working.
	ar := a.autoRunExtractor().Extract(merged)
	a.persistAutoRunSteps(ctx, id, ar, start)
	sk := a.imageSkillsExtractor().Extract(merged)
	a.persistImageSkills(ctx, id, sk, start)
}

func catalogNeedsAutoRunExtract(status string) bool {
	st := strings.TrimSpace(status)
	return st == "" || st == "pending"
}

func (a *App) scheduleCatalogItemExtract(ctx context.Context, item map[string]any) {
	if item == nil {
		return
	}
	autoStatus, _ := item["auto_run_steps_extract_status"].(string)
	skillsStatus, _ := item["image_skills_extract_status"].(string)
	needAuto := catalogNeedsAutoRunExtract(autoStatus)
	needSkills := catalogNeedsAutoRunExtract(skillsStatus)
	if !needAuto && !needSkills {
		return
	}
	imageURL, _ := item["image_url"].(string)
	version, _ := item["version"].(string)
	id, err := parseIDFlexible(item["id"])
	if err != nil || id == 0 || strings.TrimSpace(imageURL) == "" {
		return
	}
	if needAuto && strings.TrimSpace(autoStatus) == "" {
		item["auto_run_steps_extract_status"] = "pending"
	}
	if needSkills && strings.TrimSpace(skillsStatus) == "" {
		item["image_skills_extract_status"] = "pending"
	}
	// Both files missing → one combined walk; otherwise keep the single-file
	// scheduler for the missing half.
	if needAuto && needSkills {
		a.scheduleAutoRunAndSkillsExtract(ctx, id, imageURL, version)
		return
	}
	if needAuto {
		a.scheduleAutoRunExtract(ctx, id, imageURL, version)
	}
	if needSkills {
		a.scheduleImageSkillsExtract(ctx, id, imageURL, version)
	}
}

func (a *App) scheduleMissingCatalogExtracts(ctx context.Context, items []map[string]any) {
	for _, item := range items {
		a.scheduleCatalogItemExtract(ctx, item)
	}
}
