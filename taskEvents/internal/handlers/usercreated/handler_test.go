package usercreated

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

const testUserID = "849095291104686080"

func TestDispatchCreatesCompany(t *testing.T) {
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": "evtuser"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	_, name, ok, err := repo.CompanyByCreator(testUserID)
	if err != nil || !ok || name != "evtuser的公司" {
		t.Fatalf("company %+v ok=%v err=%v", name, ok, err)
	}
}

func TestDispatchIdempotentWhenCompanyExists(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.Companies[testUserID] = saastest.Company{ID: 100, Name: "existing"}
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": "evtuser"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

// TestDispatchRetriesWhenMemberRowFails — OPT-20260806-013：公司已创建但
// 成员行写入失败（瞬时 500）→ 必须返回 DispatchRetryable（事件重试），
// 而非静默成功留下「用户有公司无成员、权限全缺」状态。
func TestDispatchRetriesWhenMemberRowFails(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.FailMembersPost = true
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": "evtuser"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err == nil {
		t.Fatal("expected retryable error when member row creation fails")
	}
	if out != domain.DispatchRetryable {
		t.Fatalf("expected DispatchRetryable outcome, got %v", out)
	}
}

// TestDispatchSelfHealsMemberRow — OPT-20260806-013：公司已存在分支重试时
// 幂等确保成员行（公司存在但成员缺失的自愈路径）。
func TestDispatchSelfHealsMemberRow(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.Companies[testUserID] = saastest.Company{ID: 100, Name: "existing"}
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": "evtuser"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	// 成员行必须已写入（自愈）
	found := false
	for _, mem := range mux.CompanyMembers {
		if mem.UserID == testUserID && mem.CompanyID == "100" && mem.IsAdmin {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("member row should be ensured by self-heal, got %+v", mux.CompanyMembers)
	}
}

// TestDispatchEmptyUsernameCreatesDefaultCompany — 手机号/微信注册 USER_CREATED
// 故意不带 username（避免手机号成为公司名）。handler 不得以
// missing username 永久失败进 DLT；无个人昵称时回退「我的公司」。
func TestDispatchEmptyUsernameCreatesDefaultCompany(t *testing.T) {
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{
		"user_id":  testUserID,
		"phone":    "+8618959264502",
		"username": "",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatalf("empty username must not fail permanently: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	_, name, ok, err := repo.CompanyByCreator(testUserID)
	if err != nil || !ok {
		t.Fatalf("company ok=%v err=%v", ok, err)
	}
	if name != saas.DefaultPersonalCompanyName {
		t.Fatalf("company name = %q, want %q", name, saas.DefaultPersonalCompanyName)
	}
}

// TestDispatchEmptyUsernameUsesNickname — 空 username 但 profile 有个人昵称时，
// 公司名为「{昵称}的公司」，member_name 仍为昵称本身。
func TestDispatchEmptyUsernameUsesNickname(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.ProfileNicknames[testUserID] = "软刀"
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{
		"user_id":  testUserID,
		"username": "",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatalf("empty username with nickname must succeed: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	_, name, ok, err := repo.CompanyByCreator(testUserID)
	if err != nil || !ok {
		t.Fatalf("company ok=%v err=%v", ok, err)
	}
	if name != "软刀的公司" {
		t.Fatalf("company name = %q, want 软刀的公司", name)
	}
	if len(mux.CompanyMembers) != 1 || mux.CompanyMembers[0].MemberName != "软刀" {
		t.Fatalf("member_name must stay personal nickname, got %+v", mux.CompanyMembers)
	}
}

// TestDispatchSkipsCompanyForPlatformStaff — 平台角色（super_admin/employee）
// 注册后不自动创建公司。
func TestDispatchSkipsCompanyForPlatformStaff(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.PlatformRoles[testUserID] = []string{"super_admin"}
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": "platformuser"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	_, _, ok, err := repo.CompanyByCreator(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("platform staff must NOT have a company auto-created")
	}
}

// TestDispatchSkipsCompanyForPlatformEmployee — employee 角色同样跳过。
func TestDispatchSkipsCompanyForPlatformEmployee(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.PlatformRoles[testUserID] = []string{"employee"}
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &noopPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{"user_id": testUserID, "username": "staffuser"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	_, _, ok, err := repo.CompanyByCreator(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("platform employee must NOT have a company auto-created")
	}
}

type noopPublisher struct{}

func (n *noopPublisher) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	return nil
}
