package main

func listMembersByUser(userID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT m.id, m.user_id, m.company_id, m.is_admin, m.is_active,
			COALESCE(m.workspace_id,''), COALESCE(m.member_name,''), COALESCE(m.member_avatar,''), m.created_at,
			COALESCE(c.name, '') AS company_name,
			CASE WHEN c.creator_id = m.user_id THEN 1 ELSE 0 END AS is_creator
		FROM tenant_company_member m
		LEFT JOIN tenant_company c ON c.id = m.company_id
		WHERE m.user_id=? ORDER BY m.created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type rowScan struct {
		id, uid, cid, wsID, memberName, memberAvatar, createdAt, companyName string
		isAdmin, isActive, isCreator                                         int
	}
	scanned := make([]rowScan, 0)
	for rows.Next() {
		var s rowScan
		if err := rows.Scan(&s.id, &s.uid, &s.cid, &s.isAdmin, &s.isActive, &s.wsID, &s.memberName, &s.memberAvatar, &s.createdAt, &s.companyName, &s.isCreator); err != nil {
			continue
		}
		scanned = append(scanned, s)
	}
	nicknames := fetchPersonalNicknamesFn([]string{userID})
	personalNick := nicknames[userID]
	list := make([]map[string]interface{}, 0, len(scanned))
	for _, s := range scanned {
		display := resolveMemberDisplayName(s.memberName, s.companyName, personalNick, s.uid)
		if isMisSeededMemberName(s.memberName, s.companyName) && personalNick != "" {
			healMisSeededMemberName(s.id, personalNick)
		}
		item := map[string]interface{}{
			"id":                s.id,
			"user_id":           s.uid,
			"company_id":        s.cid,
			"is_admin":          s.isAdmin == 1,
			"is_creator":        s.isCreator == 1,
			"is_active":         s.isActive == 1,
			"workspace_id":      s.wsID,
			"member_name":       display,
			"member_avatar":     s.memberAvatar,
			"member_avatar_url": memberAvatarPublicURL(s.cid, s.id, s.memberAvatar),
			"created_at":        s.createdAt,
			"company_name":      s.companyName,
		}
		list = append(list, item)
	}
	return list, nil
}
