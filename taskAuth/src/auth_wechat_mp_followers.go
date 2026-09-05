package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

const (
	wechatMPQRSceneReconcileMaxOpenIDs = 20000
	wechatMPUserInfoBatchSize          = 100
)

type wechatMPFollowerInfo struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	Subscribe  int    `json:"subscribe"`
	QRSceneStr string `json:"qr_scene_str"`
}

func matchFollowerQRScene(followers []wechatMPFollowerInfo, tempID string) (openID, unionID string, ok bool) {
	tempID = strings.TrimSpace(tempID)
	if tempID == "" {
		return "", "", false
	}
	for _, f := range followers {
		if f.Subscribe == 0 {
			continue
		}
		if strings.TrimSpace(f.QRSceneStr) != tempID {
			continue
		}
		oid := strings.TrimSpace(f.OpenID)
		if oid == "" {
			continue
		}
		return oid, strings.TrimSpace(f.UnionID), true
	}
	return "", "", false
}

// claimMPFollowFromLiveQRScene binds a pending ticket when WeChat dropped the
// subscribe/SCAN POST but still records qr_scene_str on the follower.
func claimMPFollowFromLiveQRScene(ctx context.Context, userID string) {
	if userID == "" {
		return
	}
	ticket, ok := loadReusableMPFollowTicket(userID)
	if !ok {
		return
	}
	app := wechatMPApp()
	if app == nil || strings.TrimSpace(app.AppSecret) == "" {
		return
	}
	started := time.Now()
	openID, unionID, err := findWechatMPFollowerByQRScene(app, ticket.ID)
	durationMS := time.Since(started).Milliseconds()
	if err != nil {
		slog.WarnContext(ctx, "wechat_mp_qr_scene_reconcile_failed",
			"level", "warn",
			"error", err.Error(),
			"duration_ms", durationMS,
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return
	}
	if openID == "" {
		slog.InfoContext(ctx, "wechat_mp_qr_scene_reconcile",
			"level", "info",
			"outcome", "no_match",
			"duration_ms", durationMS,
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return
	}
	if unionID == "" {
		unionID = wechatUnionIDOf(userID)
	}
	slog.InfoContext(ctx, "wechat_mp_qr_scene_reconcile",
		"level", "info",
		"outcome", "matched",
		"openid_fp", wechatIDFingerprint(openID),
		"unionid_fp", wechatIDFingerprint(unionID),
		"duration_ms", durationMS,
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
	processWechatMPFollowTicket(ctx, app, ticket.ID, openID, unionID)
}

func findWechatMPFollowerByQRScene(app *WeChatAppConfig, tempID string) (openID, unionID string, err error) {
	token, err := getWechatMPAccessToken(app)
	if err != nil {
		return "", "", err
	}
	if token == "" {
		return "", "", wechatMPAPIError{Code: -1, Msg: "empty access_token"}
	}
	next := ""
	seen := 0
	for seen < wechatMPQRSceneReconcileMaxOpenIDs {
		openIDs, nextOpenID, listErr := fetchWechatMPFollowerOpenIDs(token, next)
		if listErr != nil {
			return "", "", listErr
		}
		if len(openIDs) == 0 {
			break
		}
		for i := 0; i < len(openIDs); i += wechatMPUserInfoBatchSize {
			end := i + wechatMPUserInfoBatchSize
			if end > len(openIDs) {
				end = len(openIDs)
			}
			infos, batchErr := fetchWechatMPFollowerBatch(token, openIDs[i:end])
			if batchErr != nil {
				return "", "", batchErr
			}
			if oid, uid, ok := matchFollowerQRScene(infos, tempID); ok {
				return oid, uid, nil
			}
		}
		seen += len(openIDs)
		if nextOpenID == "" || nextOpenID == next {
			break
		}
		next = nextOpenID
	}
	return "", "", nil
}

func fetchWechatMPFollowerOpenIDs(token, nextOpenID string) ([]string, string, error) {
	u := "https://api.weixin.qq.com/cgi-bin/user/get?access_token=" + url.QueryEscape(token)
	if nextOpenID != "" {
		u += "&next_openid=" + url.QueryEscape(nextOpenID)
	}
	body, err := wechatHTTPGet(u)
	if err != nil {
		return nil, "", err
	}
	var parsed struct {
		Data struct {
			OpenID []string `json:"openid"`
		} `json:"data"`
		NextOpenID string `json:"next_openid"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", err
	}
	if parsed.ErrCode != 0 {
		return nil, "", wechatMPAPIError{Code: parsed.ErrCode, Msg: parsed.ErrMsg}
	}
	return parsed.Data.OpenID, strings.TrimSpace(parsed.NextOpenID), nil
}

func fetchWechatMPFollowerBatch(token string, openIDs []string) ([]wechatMPFollowerInfo, error) {
	type item struct {
		OpenID string `json:"openid"`
		Lang   string `json:"lang"`
	}
	list := make([]item, 0, len(openIDs))
	for _, oid := range openIDs {
		list = append(list, item{OpenID: oid, Lang: "zh_CN"})
	}
	payload, err := json.Marshal(map[string]interface{}{"user_list": list})
	if err != nil {
		return nil, err
	}
	body, err := wechatHTTPPostJSON("https://api.weixin.qq.com/cgi-bin/user/info/batchget?access_token="+url.QueryEscape(token), payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		UserInfoList []wechatMPFollowerInfo `json:"user_info_list"`
		ErrCode      int                    `json:"errcode"`
		ErrMsg       string                 `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.ErrCode != 0 {
		return nil, wechatMPAPIError{Code: parsed.ErrCode, Msg: parsed.ErrMsg}
	}
	return parsed.UserInfoList, nil
}
