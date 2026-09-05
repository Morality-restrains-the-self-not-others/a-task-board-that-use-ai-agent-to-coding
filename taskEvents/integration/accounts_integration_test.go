//go:build integration

package integration_test

import (
	"testing"

	"taskEvents/config"
	"taskEvents/domain"
	"taskEvents/integration"
	"taskEvents/internal/handlers/usercreated"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

func setupUserCreatedRepo(t *testing.T) *saas.Repository {
	t.Helper()
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	t.Cleanup(srv.Close)
	return saas.New("")
}

func TestUserCreatedRedisRoundTrip(t *testing.T) {
	host, port, db := integration.RedisAvailable(t)
	repo := setupUserCreatedRepo(t)
	cfg, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	stream := cfg.StreamKey
	if stream == "" {
		stream = "domain-events:all"
	}
	pub := publish.NewRedisStreamPublisher(host, port, db, stream)
	defer pub.Close()

	h := &usercreated.Handler{Repo: repo, Publisher: pub}
	out := integration.DispatchOnce(
		t,
		"USER_CREATED",
		map[string]interface{}{"user_id": "9001", "username": "intuser"},
		"9001",
		"user_created",
		h,
		nil,
	)
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	_, name, ok, err := repo.CompanyByCreator("9001")
	if err != nil || !ok || name != "intuser" {
		t.Fatalf("company name=%q ok=%v err=%v", name, ok, err)
	}
}

func TestUserCreatedIdempotent(t *testing.T) {
	host, port, dbNum := integration.RedisAvailable(t)
	repo := setupUserCreatedRepo(t)
	cfg, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	stream := cfg.StreamKey
	if stream == "" {
		stream = "domain-events:all"
	}
	pub := publish.NewRedisStreamPublisher(host, port, dbNum, stream)
	defer pub.Close()
	h := &usercreated.Handler{Repo: repo, Publisher: pub}
	data := map[string]interface{}{"user_id": "9002", "username": "idemuser"}
	out1 := integration.DispatchOnce(t, "USER_CREATED", data, "9002", "user_created", h, nil)
	out2 := integration.DispatchOnce(t, "USER_CREATED", data, "9002", "user_created", h, nil)
	if out1 != domain.DispatchSuccess || out2 != domain.DispatchSuccess {
		t.Fatalf("outcomes %v %v", out1, out2)
	}
	if _, _, ok, _ := repo.CompanyByCreator("9002"); !ok {
		t.Fatal("expected company")
	}
}
