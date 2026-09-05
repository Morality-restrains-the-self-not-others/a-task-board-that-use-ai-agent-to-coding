package saas

import (
	"testing"

	"taskEvents/internal/saastest"
)

func TestPersonalCompanyName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"张三", "张三的公司"},
		{"  软刀  ", "软刀的公司"},
		{"", DefaultPersonalCompanyName},
		{"   ", DefaultPersonalCompanyName},
		{DefaultPersonalCompanyName, DefaultPersonalCompanyName},
		{"软刀的公司", "软刀的公司"},
		{"softknife", "softknife的公司"},
	}
	for _, tc := range cases {
		got := PersonalCompanyName(tc.in)
		if got != tc.want {
			t.Errorf("PersonalCompanyName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolvePersonalCompanyName(t *testing.T) {
	cases := []struct {
		username, nickname, want string
	}{
		{"evtuser", "", "evtuser的公司"},
		{"", "软刀", "软刀的公司"},
		{DefaultPersonalCompanyName, "软刀", "软刀的公司"},
		{"", "", DefaultPersonalCompanyName},
		{"张三的公司", "ignored", "张三的公司"},
	}
	for _, tc := range cases {
		got := ResolvePersonalCompanyName(tc.username, tc.nickname)
		if got != tc.want {
			t.Errorf("ResolvePersonalCompanyName(%q, %q) = %q, want %q",
				tc.username, tc.nickname, got, tc.want)
		}
	}
}

// TestCreateCompanyEmptyUsernameUsesNeutralName — OPT-20260806-035/20260810-020：
// username 为空且无个人昵称时兜底使用中性默认名「我的公司」，而非雪花数字 userID。
func TestCreateCompanyEmptyUsernameUsesNeutralName(t *testing.T) {
	if DefaultPersonalCompanyName != "我的公司" {
		t.Fatalf("DefaultPersonalCompanyName = %q, want 我的公司", DefaultPersonalCompanyName)
	}
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := New("")
	created, err := repo.CreateCompanyForUser("849095291104686080", "")
	if err != nil {
		t.Fatalf("create with empty username: %v", err)
	}
	if created.Name != DefaultPersonalCompanyName {
		t.Fatalf("empty username company name = %q, want %q", created.Name, DefaultPersonalCompanyName)
	}
	// 公司名不应是雪花数字 ID
	if created.Name == "849095291104686080" {
		t.Fatalf("company name must not be the numeric userID")
	}
}

// TestCreateCompanyMemberNameUsesPersonalNickname — 创建者 member_name 必须是个人昵称，
// 绝不能写成公司名「我的公司」（人员管理页「公司成员名称」列回归）。
func TestCreateCompanyMemberNameUsesPersonalNickname(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.ProfileNicknames["849095291104686080"] = "软刀"
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := New("")
	created, err := repo.CreateCompanyForUser("849095291104686080", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Name != "软刀的公司" {
		t.Fatalf("company name = %q, want 软刀的公司", created.Name)
	}
	if len(mux.CompanyMembers) != 1 {
		t.Fatalf("expected 1 member, got %+v", mux.CompanyMembers)
	}
	mem := mux.CompanyMembers[0]
	if mem.MemberName != "软刀" {
		t.Fatalf("member_name = %q, want 软刀 (personal nickname), not company name", mem.MemberName)
	}
	if mem.MemberName == DefaultPersonalCompanyName {
		t.Fatalf("member_name must not be default company name %q", DefaultPersonalCompanyName)
	}
}

// TestEnsureCreatorMemberDoesNotOverwriteNickname — 自愈不得用公司名覆盖已有昵称。
func TestEnsureCreatorMemberDoesNotOverwriteNickname(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.Companies["u-heal"] = saastest.Company{ID: 42, Name: DefaultPersonalCompanyName}
	mux.CompanyMembers = []saastest.CompanyMemberEntry{{
		ID: "m-existing", UserID: "u-heal", CompanyID: "42", IsAdmin: true, MemberName: "已有昵称",
	}}
	mux.ProfileNicknames["u-heal"] = "个人昵称"
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := New("")
	if err := repo.EnsureCreatorMember(42, "u-heal"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if len(mux.CompanyMembers) != 1 {
		t.Fatalf("must not insert duplicate member, got %+v", mux.CompanyMembers)
	}
	if mux.CompanyMembers[0].MemberName != "已有昵称" {
		t.Fatalf("existing member_name overwritten: got %q", mux.CompanyMembers[0].MemberName)
	}
}

// TestCreateCompanyUsernamePreserved — 非空 username 生成「{user}的公司」。
func TestCreateCompanyUsernamePreserved(t *testing.T) {
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := New("")
	created, err := repo.CreateCompanyForUser("849095291104686080", "softknife")
	if err != nil {
		t.Fatalf("create with username: %v", err)
	}
	if created.Name != "softknife的公司" {
		t.Fatalf("company name = %q, want softknife的公司", created.Name)
	}
}
