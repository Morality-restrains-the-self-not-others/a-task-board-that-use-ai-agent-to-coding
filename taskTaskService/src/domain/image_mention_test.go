package domain

import (
	"errors"
	"testing"
)

func TestValidateMentions(t *testing.T) {
	t.Run("empty ok", func(t *testing.T) {
		if err := ValidateMentions(false, nil); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("multiple rejected", func(t *testing.T) {
		mentions := []ImageMention{
			{Type: MentionTypeInstalledImage, ID: "img1", Name: "A"},
			{Type: MentionTypeInstalledImage, ID: "img2", Name: "B"},
		}
		if err := ValidateMentions(true, mentions); !errors.Is(err, ErrTooManyMentions) {
			t.Fatalf("expected ErrTooManyMentions, got %v", err)
		}
	})
	t.Run("disabled rejected", func(t *testing.T) {
		mentions := []ImageMention{{Type: MentionTypeInstalledImage, ID: "img1", Name: "A"}}
		if err := ValidateMentions(false, mentions); !errors.Is(err, ErrMentionAtModeDisabled) {
			t.Fatalf("expected ErrMentionAtModeDisabled, got %v", err)
		}
	})
	t.Run("enabled single ok", func(t *testing.T) {
		mentions := []ImageMention{{Type: MentionTypeInstalledImage, ID: "img1", Name: "A"}}
		if err := ValidateMentions(true, mentions); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
	t.Run("invalid skill name", func(t *testing.T) {
		mentions := []ImageMention{{Type: MentionTypeInstalledImage, ID: "img1", Name: "A", Skill: "Bad_Name"}}
		if err := ValidateMentions(true, mentions); !errors.Is(err, ErrInvalidMention) {
			t.Fatalf("expected ErrInvalidMention, got %v", err)
		}
	})
}

func TestApplyMentionSkill(t *testing.T) {
	list := ImageSkillList{
		Version: 1, DefaultSkill: "general-coding",
		Skills: []ImageSkill{{Name: "general-coding", IsDefault: true}, {Name: "k8s-debug"}},
	}
	t.Run("empty fills default", func(t *testing.T) {
		m := ImageMention{Type: MentionTypeInstalledImage, ID: "1"}
		if err := ApplyMentionSkill(&m, list); err != nil {
			t.Fatal(err)
		}
		if m.Skill != "general-coding" {
			t.Fatalf("skill=%q", m.Skill)
		}
	})
	t.Run("unknown rejected when list present", func(t *testing.T) {
		m := ImageMention{Type: MentionTypeInstalledImage, ID: "1", Skill: "nope"}
		if err := ApplyMentionSkill(&m, list); !errors.Is(err, ErrUnknownImageSkill) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("known accepted", func(t *testing.T) {
		m := ImageMention{Type: MentionTypeInstalledImage, ID: "1", Skill: "k8s-debug"}
		if err := ApplyMentionSkill(&m, list); err != nil || m.Skill != "k8s-debug" {
			t.Fatalf("err=%v skill=%q", err, m.Skill)
		}
	})
	t.Run("empty catalog accepts format-valid skill", func(t *testing.T) {
		m := ImageMention{Type: MentionTypeInstalledImage, ID: "1", Skill: "k8s-debug"}
		if err := ApplyMentionSkill(&m, ImageSkillList{}); err != nil {
			t.Fatal(err)
		}
	})
}
