package main

import (
	"database/sql"

	"taskAuth/domain"
)

// userAllowsSharedPhone 超级用户/员工/测试（含平台角色 super_admin、employee）可共享同一手机号。
func userAllowsSharedPhone(userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	admin, err := isPlatformAdminStaff(userID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}
	var tester int
	err = db.QueryRow(`SELECT COALESCE(is_tester, 0) FROM auth_user WHERE id = ?`, userID).Scan(&tester)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return tester == 1, nil
}

// phoneBindDecision 对 userID 绑定 (cc,national) 给出 Allow/Taken/Limit；otherCount 为其他活跃占用数。
func phoneBindDecision(userID, countryCode, national string) (domain.PhoneShareDecision, int, error) {
	methods, err := listLiveLoginMethodsByPhone(countryCode, national)
	if err != nil {
		return "", 0, err
	}
	actorPriv, err := userAllowsSharedPhone(userID)
	if err != nil {
		return "", 0, err
	}
	otherPriv := make([]bool, 0, len(methods))
	for _, m := range methods {
		if m.ObjectID == userID {
			continue
		}
		p, err := userAllowsSharedPhone(m.ObjectID)
		if err != nil {
			return "", 0, err
		}
		otherPriv = append(otherPriv, p)
	}
	d := domain.EvaluatePhoneShareBind(actorPriv, otherPriv)
	return d, len(otherPriv), nil
}

// matchPhoneLoginMethodByPassword 在同一号码的活跃绑定中按密码消歧。
// 无绑定时返回 (nil, nil)；无一匹配时返回第一条供后续走统一 password_mismatch；
// 两条及以上匹配返回 errPhoneAmbiguous。
func matchPhoneLoginMethodByPassword(canonical, password string) (*LoginMethodRow, error) {
	cc, nat := splitCountryCallingCodeAndNational(canonical)
	if cc == "" || nat == "" {
		return nil, nil
	}
	methods, err := listLiveLoginMethodsByPhone(cc, nat)
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, nil
	}
	var matches []LoginMethodRow
	for _, m := range methods {
		if m.PasswordHash != "" && checkPasswordHash(password, m.PasswordHash) {
			matches = append(matches, m)
		}
	}
	if domain.IsPhoneLoginAmbiguous(len(matches)) {
		return nil, errPhoneAmbiguous
	}
	if len(matches) == 1 {
		lm := matches[0]
		return &lm, nil
	}
	lm := methods[0]
	return &lm, nil
}

func livePhoneBindingCount(phone string) (int, error) {
	canonical := canonicalPhoneForSMSAndLogin(phone)
	cc, nat := splitCountryCallingCodeAndNational(canonical)
	if cc == "" || nat == "" {
		return 0, nil
	}
	methods, err := listLiveLoginMethodsByPhone(cc, nat)
	if err != nil {
		return 0, err
	}
	return len(methods), nil
}
