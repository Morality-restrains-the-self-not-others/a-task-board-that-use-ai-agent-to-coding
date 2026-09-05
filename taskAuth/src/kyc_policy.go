package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
)

func getKycLimitPolicy(tier string) (kycLimitPolicy, error) {
	tier = strings.TrimSpace(tier)
	var p kycLimitPolicy
	var allowed int
	err := db.QueryRow(`
		SELECT tier, max_single_yuan, max_daily_yuan, recharge_allowed, updated_at
		FROM auth_kyc_limit_policy WHERE tier = ?`, tier,
	).Scan(&p.Tier, &p.MaxSingleYuan, &p.MaxDailyYuan, &allowed, &p.UpdatedAt)
	if err != nil {
		return kycLimitPolicy{}, err
	}
	p.RechargeAllowed = allowed != 0
	return p, nil
}

func listKycLimitPolicies() ([]kycLimitPolicy, error) {
	rows, err := db.Query(`
		SELECT tier, max_single_yuan, max_daily_yuan, recharge_allowed, updated_at
		FROM auth_kyc_limit_policy
		ORDER BY CASE tier
			WHEN 'T0_unverified' THEN 0
			WHEN 'T1_basic' THEN 1
			WHEN 'T2_enhanced' THEN 2
			ELSE 9 END`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []kycLimitPolicy
	for rows.Next() {
		var p kycLimitPolicy
		var allowed int
		if err := rows.Scan(&p.Tier, &p.MaxSingleYuan, &p.MaxDailyYuan, &allowed, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.RechargeAllowed = allowed != 0
		out = append(out, p)
	}
	return out, rows.Err()
}

func upsertKycLimitPolicy(tier string, maxSingle, maxDaily int, rechargeAllowed bool) (kycLimitPolicy, error) {
	if !isValidKycTier(tier) {
		return kycLimitPolicy{}, errKycInvalidTier
	}
	if maxSingle < 0 {
		maxSingle = 0
	}
	if maxDaily < 0 {
		maxDaily = 0
	}
	allowed := 0
	if rechargeAllowed {
		allowed = 1
	}
	now := timeNowUTC()
	_, err := db.Exec(`
		INSERT INTO auth_kyc_limit_policy (
			tier, max_single_yuan, max_daily_yuan, recharge_allowed, updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			max_single_yuan = VALUES(max_single_yuan),
			max_daily_yuan = VALUES(max_daily_yuan),
			recharge_allowed = VALUES(recharge_allowed),
			updated_at = VALUES(updated_at)`,
		tier, maxSingle, maxDaily, allowed, now,
	)
	if err != nil {
		return kycLimitPolicy{}, err
	}
	return getKycLimitPolicy(tier)
}

func checkKycRechargeGate(userID string, amountYuan int64) (kycRechargeGateResult, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return kycRechargeGateResult{}, errKycUserRequired
	}
	if amountYuan < 0 {
		return kycRechargeGateResult{}, errKycInvalidAmount
	}
	profile, err := getOrCreateKycProfile(userID)
	if err != nil {
		return kycRechargeGateResult{}, err
	}
	policy, err := getKycLimitPolicy(profile.Tier)
	if err == sql.ErrNoRows {
		policy = kycLimitPolicy{Tier: profile.Tier, MaxSingleYuan: 0, MaxDailyYuan: 0, RechargeAllowed: false}
	} else if err != nil {
		return kycRechargeGateResult{}, err
	}

	result := kycRechargeGateResult{
		Allowed:           false,
		Tier:              profile.Tier,
		Status:            profile.Status,
		MaxSingleYuan:     policy.MaxSingleYuan,
		MaxDailyYuan:      policy.MaxDailyYuan,
		DailyUsedYuanHint: 0,
	}

	aml, err := latestAmlScreeningResult(userID)
	if err != nil {
		return kycRechargeGateResult{}, err
	}
	if aml == amlResultHit || aml == amlResultReview {
		result.ReasonCode = reasonKycAMLReview
		return result, nil
	}

	if !policy.RechargeAllowed {
		result.ReasonCode = reasonKycTierBlocked
		return result, nil
	}
	if amountYuan > int64(policy.MaxSingleYuan) {
		result.ReasonCode = reasonKycAmountExceeds
		return result, nil
	}
	result.Allowed = true
	result.ReasonCode = reasonKycOK
	return result, nil
}

func emitKycChangeEvents(
	ctx context.Context,
	old, updated kycProfile,
	triggerSource, actorID, reasonCode, requestID string,
) {
	base := map[string]interface{}{
		"user_id":        updated.UserID,
		"old_tier":       old.Tier,
		"new_tier":       updated.Tier,
		"old_status":     old.Status,
		"new_status":     updated.Status,
		"trigger_source": triggerSource,
		"actor_id":       actorID,
		"reason_code":    reasonCode,
		"request_id":     requestID,
		"updated_at":     updated.UpdatedAt,
	}
	if old.Tier != updated.Tier {
		publishKycEvent(ctx, "KYC_TIER_CHANGED", base, updated.UserID)
	}
	if old.Status != updated.Status {
		publishKycEvent(ctx, "KYC_STATUS_CHANGED", base, updated.UserID)
	}
	log.Printf(
		"[taskAuth] kyc change user_id=%s tier=%s->%s status=%s->%s source=%s request_id=%s",
		updated.UserID, old.Tier, updated.Tier, old.Status, updated.Status, triggerSource, requestID,
	)
}

func publishKycEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) {
	if err := publishDomainEventKafka(ctx, eventType, data, key); err != nil {
		if err == errKafkaNotConfigured {
			log.Printf("[taskAuth] kyc event %s skipped: kafka not configured", eventType)
			return
		}
		log.Printf("[taskAuth] kyc event %s failed: %v", eventType, err)
	}
}
