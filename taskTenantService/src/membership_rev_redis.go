//go:build redis

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var membershipRevClient *redis.Client

func initMembershipRevRedis() {
	if membershipRevClient != nil {
		return
	}
	host := firstNonEmpty(
		os.Getenv("TENANT_REDIS_HOST"),
		os.Getenv("INFRA_HOST"),
	)
	if host == "" {
		return
	}
	port := firstNonEmpty(os.Getenv("TENANT_REDIS_PORT"), "6379")
	db := 0
	if v := os.Getenv("TENANT_REDIS_DB"); v != "" {
		db, _ = strconv.Atoi(v)
	}
	membershipRevClient = redis.NewClient(&redis.Options{
		Addr:         host + ":" + port,
		DB:           db,
		DialTimeout:  2 * time.Second,
		WriteTimeout: 1 * time.Second,
	})
	log.Printf("[taskTenantService] membership_rev redis connected %s/%d", host+":"+port, db)
}

func init() {
	initMembershipRevRedis()
}

func incrMembershipRev(userID string) {
	if membershipRevClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	key := "membership_rev:" + userID
	if err := membershipRevClient.Incr(ctx, key).Err(); err != nil {
		log.Printf("[taskTenantService] membership_rev incr failed user_id=%s: %v", userID, err)
		return
	}
	membershipRevClient.Expire(ctx, key, 30*time.Minute)
}

func lookupMembershipRev(userID string) (int64, error) {
	if membershipRevClient == nil {
		return 0, fmt.Errorf("redis not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return membershipRevClient.Get(ctx, "membership_rev:"+userID).Int64()
}

// incrGroupMembersRev bumps the membership revision for every member of groupID.
// 组级变更（组角色、资源组授权/回收）会影响组内全部成员的 PDP 权限集合，
// 需逐一 incr 其 rev 才能让 taskAuth 侧 PDP 缓存按 rev 失效。
func incrGroupMembersRev(groupID string) {
	if groupID == "" || membershipRevClient == nil {
		return
	}
	rows, err := db.Query(`SELECT DISTINCT user_id FROM tenant_company_group_member WHERE group_id=?`, groupID)
	if err != nil {
		log.Printf("[taskTenantService] membership_rev group members query failed group_id=%s: %v", groupID, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil && uid != "" {
			incrMembershipRev(uid)
		}
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
