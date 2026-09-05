package cloudserverstartauto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

type mockNetwork struct{}

func (mockNetwork) Provision(ctx context.Context, accessKey, secretKey string, in aliyun.AutoNetworkInput) (aliyun.AutoNetworkResult, error) {
	_ = ctx
	_ = accessKey
	_ = secretKey
	out := aliyun.AutoNetworkResult{
		VPCID:           in.ExistingVPCID,
		VSwitchID:       in.ExistingVSwitchID,
		SecurityGroupID: in.ExistingSGID,
	}
	if in.AutoVPC {
		out.VPCID = "vpc-mock"
	}
	if in.AutoVSwitch {
		out.VSwitchID = "vsw-mock"
	}
	if in.AutoSG {
		out.SecurityGroupID = "sg-mock"
	}
	return out, nil
}

type capturePub struct {
	events []string
	last   map[string]interface{}
}

func (p *capturePub) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	_ = ctx
	_ = key
	p.events = append(p.events, eventType)
	p.last = data
	return nil
}

func TestDispatchAutoCreateChainsStarted(t *testing.T) {
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	repo := saas.New("")
	pub := &capturePub{}
	h := &Handler{Repo: repo, Publisher: pub, Network: mockNetwork{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"auto_create_vpc": true, "auto_create_vswitch": true, "auto_create_security_group": true,
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_START_AUTO",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_START_AUTO", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if len(pub.events) == 0 || pub.events[len(pub.events)-1] != "CLOUD_SERVER_STARTED" {
		t.Fatalf("events=%v", pub.events)
	}
	if pub.last["vpc_id"] != "vpc-mock" || pub.last["vswitch_id"] != "vsw-mock" || pub.last["security_group_id"] != "sg-mock" {
		t.Fatalf("next data=%v", pub.last)
	}
	if pub.last["auto_create_vpc"] != false {
		t.Fatalf("flags not cleared: %v", pub.last)
	}
}

func TestDispatchAutoCreateMissingAuthorizationID(t *testing.T) {
	// 回归（rule 41）：go vet 非恒定格式字符串修复（fmt.Errorf(msg)→fmt.Errorf("%s", msg)），
	// 断言该错误路径消息原样透传。修复前本包因 vet 构建失败，任何单测均为红。
	h := &Handler{Repo: saas.New(""), Publisher: &capturePub{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "company_id": 100,
		"cloud_platform_type": "aliyun", "region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"auto_create_vpc": true,
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_START_AUTO",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_START_AUTO", Data: data},
	})
	if out != domain.DispatchPermanent || err == nil || err.Error() != "缺少 authorization_id" {
		t.Fatalf("expected permanent missing authorization_id error, got out=%v err=%v", out, err)
	}
}

func TestDispatchStartAutoMockPlatformSucceedsWithoutChain(t *testing.T) {
	pub := &capturePub{}
	h := &Handler{Publisher: pub, Network: mockNetwork{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-mock", "company_id": 100,
		"cloud_platform_type": "mock", "region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_START_AUTO",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_START_AUTO", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("mock platform must not DLT, outcome %v", out)
	}
	for _, ev := range pub.events {
		if ev == "CLOUD_SERVER_STARTED" {
			t.Fatal("mock platform must not chain CLOUD_SERVER_STARTED")
		}
	}
}
