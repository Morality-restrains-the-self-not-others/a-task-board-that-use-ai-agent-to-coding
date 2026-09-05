package main

import (
	"fmt"
	"strings"

	"taskAuth/domain"
)

// IdentifierConflict 描述解档用户时发现的一个标识冲突：该用户仍持有未作废标识，
// 而同一标识已被另一活跃用户绑定。
type IdentifierConflict struct {
	MethodType string
	Identifier string
}

// findActiveIdentifierConflicts 返回解档前需要处理的标识冲突列表（OPT-20260824-089）。
//
// 注册/管理端建号只在「占用者已归档或不存在」时才回收旧标识；若占用者曾被归档后
// 又被新用户占用同一手机号/邮箱（如系统管理端直接建号，不走 voidStale 回收），
// 旧用户的 login_method 可能仍是未作废状态。直接解档会恢复绑定，造成双活占用或
// 登录歧义，故解档前必须检测。
//
// 手机号按 phone_country_calling_code + identifier 精确配对；邮箱/用户名按
// LOWER(identifier) 不区分大小写配对。只与 is_archived=0 的活跃用户比较。
func findActiveIdentifierConflicts(userID string) ([]IdentifierConflict, error) {
	rows, err := db.Query(`
		SELECT lm2.method_type, lm2.identifier, COALESCE(lm.phone_country_calling_code, '')
		FROM auth_login_method lm
		JOIN auth_login_method lm2 ON (
			(lm.method_type = 'phone' AND lm2.method_type = 'phone'
				AND lm2.phone_country_calling_code = lm.phone_country_calling_code
				AND lm2.identifier = lm.identifier)
			OR (lm.method_type != 'phone' AND lm2.method_type = lm.method_type
				AND LOWER(lm2.identifier) = LOWER(lm.identifier))
		)
		JOIN auth_user u2 ON u2.id = lm2.object_id AND COALESCE(u2.is_archived, 0) = 0
		WHERE lm.object_id = ?
		  AND lm.binding_voided_at IS NULL
		  AND lm2.binding_voided_at IS NULL
		  AND lm2.object_id != ?
		GROUP BY lm2.method_type, lm2.identifier, lm.phone_country_calling_code
		ORDER BY lm2.method_type, lm2.identifier
		LIMIT 10`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []IdentifierConflict
	for rows.Next() {
		var c IdentifierConflict
		var cc string
		if err := rows.Scan(&c.MethodType, &c.Identifier, &cc); err != nil {
			return nil, err
		}
		if c.MethodType == "phone" {
			d, _, err := phoneBindDecision(userID, cc, c.Identifier)
			if err != nil {
				return nil, err
			}
			if d == domain.PhoneShareAllow {
				continue
			}
		}
		conflicts = append(conflicts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conflicts, nil
}

// describeIdentifierConflict 返回给管理员的冲突说明文案。
func describeIdentifierConflict(conflicts []IdentifierConflict) string {
	if len(conflicts) == 0 {
		return ""
	}
	labels := map[string]string{"phone": "手机号", "email": "邮箱", "username": "用户名"}
	parts := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		label := labels[c.MethodType]
		if label == "" {
			label = c.MethodType
		}
		parts = append(parts, fmt.Sprintf("%s %s", label, c.Identifier))
	}
	return "该用户仍持有的未作废标识已被活跃用户占用：" + strings.Join(parts, "、") + "。请先解绑或更换后再解档"
}
