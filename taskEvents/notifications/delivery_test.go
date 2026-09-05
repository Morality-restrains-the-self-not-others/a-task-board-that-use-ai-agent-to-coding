package notifications_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskEvents/domain"
	"taskEvents/notifications"
	"taskEvents/notifications/cfg"
)

func TestLocalDeliveryEmailSent(t *testing.T) {
	ln, addr := startSMTPServer(t)
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host:          host,
			Port:          port,
			User:          "user",
			Password:      "pass",
			DefaultFrom:   "from@example.com",
			UseSSL:        false,
			LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "账号激活 - SaaS平台",
		"message":        "plain body",
		"html_message":   "<p>html</p>",
		"from_email":     "from@example.com",
		"recipient_list": []string{"user@example.com"},
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestLocalDeliveryEmailSentSMTP535Permanent(t *testing.T) {
	ln, addr := startSMTPServerAuthReply(t, "535 Login fail. Account is abnormal, login frequency limited")
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host:          host,
			Port:          port,
			User:          "user",
			Password:      "pass",
			DefaultFrom:   "from@example.com",
			UseSSL:        false,
			LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "密码重置 - SaaS平台",
		"message":        "plain body",
		"html_message":   "<p>html</p>",
		"from_email":     "from@example.com",
		"recipient_list": []string{"ruandao@example.com"},
		"template_name":  "password_reset",
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k535"},
	})
	if err == nil {
		t.Fatal("expected SMTP 535 error")
	}
	if !strings.Contains(err.Error(), "535") {
		t.Fatalf("err=%v, want 535", err)
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v, want DispatchPermanent (535 must not retry)", out)
	}
}

func TestLocalDeliveryInvitationWithMessage(t *testing.T) {
	ln, addr := startSMTPServer(t)
	defer ln.Close()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: host, Port: port, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}
	custom := "欢迎加入我们团队，一起高效协作。"
	data, _ := json.Marshal(map[string]interface{}{
		"company_name":    "测试公司",
		"email":           "new.member@example.com",
		"invitation_url":  "http://localhost:4000/join/?token=abc",
		"expiration_days": 7,
		"invite_method":   "email",
		"message":         custom,
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "INVITATION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "INVITATION_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestLocalDeliveryInvitationMissingCompanyNameFallsBack(t *testing.T) {
	ln, addr := startSMTPServer(t)
	defer ln.Close()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: host, Port: port, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}
	// 回归 OPT-20260809-015：旧发布方/存量事件只带 company_id 不带 company_name，
	// 消费端须兜底渲染而非永久卡「投递中」。
	data, _ := json.Marshal(map[string]interface{}{
		"company_id":     "874176608758427648",
		"email":          "member@example.com",
		"invitation_url": "http://localhost:4000/join/?token=abc",
		"invite_method":  "email",
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "INVITATION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "INVITATION_CREATED", Data: data},
	})
	if err != nil {
		t.Fatalf("missing company_name must not fail, got %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v want DispatchSuccess", out)
	}
}

func TestLocalDeliveryInvitationMissingURLPermanentCallbacksFailed(t *testing.T) {
	// 模拟 taskTenantService delivery-callback 端点，记录 failed 回调。
	var called bool
	var gotStatus string
	cb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/tenant/invitations/delivery-callback/inv1/" {
			called = true
			var body struct {
				Status       string `json:"status"`
				ErrorMessage string `json:"error_message"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotStatus = body.Status
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer cb.Close()

	t.Setenv("TASK_TENANT_INTERNAL_URL", cb.URL)

	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: "127.0.0.1", Port: 9999, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}
	// 回归 OPT-20260809-015：永久失败（缺 invitation_url）须回调 failed，避免前端永远看到「投递中」。
	data, _ := json.Marshal(map[string]interface{}{
		"invitation_id": "inv1",
		"company_name":  "测试公司",
		"email":         "member@example.com",
		"invite_method": "email",
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "INVITATION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "INVITATION_CREATED", Data: data},
	})
	if err == nil {
		t.Fatal("expected permanent error for missing invitation_url")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v want DispatchPermanent", out)
	}
	if !called {
		t.Fatal("expected failed delivery callback to taskTenantService")
	}
	if gotStatus != "failed" {
		t.Fatalf("callback status %q want failed", gotStatus)
	}
}

func TestLocalDeliveryEmailSentWithTemplate(t *testing.T) {
	ln, addr := startSMTPServer(t)
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host:          host,
			Port:          port,
			User:          "user",
			Password:      "pass",
			DefaultFrom:   "from@example.com",
			UseSSL:        false,
			LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}

	resetURL := "http://localhost:4000/auth/reset-password/token123/"
	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "密码重置 - SaaS平台",
		"from_email":     "from@example.com",
		"recipient_list": []string{"user@example.com"},
		"template_name":  "password_reset",
		"context": map[string]interface{}{
			"reset_url": resetURL,
		},
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestLocalDeliveryEmailSentEmptyBodyPermanent(t *testing.T) {
	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: "127.0.0.1", Port: 9999, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "empty",
		"recipient_list": []string{"user@example.com"},
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k3"},
	})
	if err == nil {
		t.Fatal("expected permanent error for empty body")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v want permanent", out)
	}
}

func TestLocalDeliveryEmailSentMissingTemplateContextPermanent(t *testing.T) {
	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: "127.0.0.1", Port: 9999, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "密码重置 - SaaS平台",
		"recipient_list": []string{"user@example.com"},
		"template_name":  "password_reset",
		"context":        map[string]interface{}{},
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k4"},
	})
	if err == nil {
		t.Fatal("expected permanent error for missing reset_url")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v want permanent", out)
	}
}

func TestLocalDeliveryEmailSentTemplateFillsMissingHTML(t *testing.T) {
	ln, addr := startSMTPServer(t)
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: host, Port: port, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := json.Marshal(map[string]interface{}{
		"subject":        "密码重置 - SaaS平台",
		"recipient_list": []string{"user@example.com"},
		"message":        "plain only",
		"template_name":  "password_reset",
		"context": map[string]interface{}{
			"reset_url": "http://localhost:4000/auth/reset-password/tok/",
		},
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "EMAIL_SENT",
		Envelope:  domain.EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "k5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func startSMTPServer(t *testing.T) (net.Listener, string) {
	return startSMTPServerAuthReply(t, "235 ok")
}

func startSMTPServerAuthReply(t *testing.T, authReply string) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSMTP(conn, authReply)
		}
	}()
	return ln, ln.Addr().String()
}

func handleSMTP(conn net.Conn, authReply string) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writeLine(conn, "220 localhost ESMTP")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO") || strings.HasPrefix(cmd, "HELO"):
			writeLine(conn, "250-localhost")
			writeLine(conn, "250 AUTH PLAIN")
		case strings.HasPrefix(cmd, "AUTH"):
			writeLine(conn, authReply)
			if strings.HasPrefix(authReply, "535") {
				return
			}
		case strings.HasPrefix(cmd, "MAIL FROM"):
			writeLine(conn, "250 ok")
		case strings.HasPrefix(cmd, "RCPT TO"):
			writeLine(conn, "250 ok")
		case cmd == "DATA":
			writeLine(conn, "354 go")
			for {
				l, _ := reader.ReadString('\n')
				if strings.TrimSpace(l) == "." {
					break
				}
			}
			writeLine(conn, "250 queued")
		case cmd == "QUIT":
			writeLine(conn, "221 bye")
			return
		default:
			writeLine(conn, "250 ok")
		}
	}
}

func writeLine(conn net.Conn, msg string) {
	_, _ = conn.Write([]byte(msg + "\r\n"))
}

// 租户邀请投递回调：SMTP 发送成功后回调 taskTenantService（delivered）
func TestLocalDeliveryTenantInvitationCallbackDelivered(t *testing.T) {
	var gotBody, gotPath string
	cbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	defer cbSrv.Close()
	t.Setenv("TASK_TENANT_INTERNAL_URL", cbSrv.URL)

	ln, addr := startSMTPServer(t)
	defer ln.Close()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: host, Port: port, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]interface{}{
		"company_name":    "回调测试公司",
		"email":           "cb.member@example.com",
		"invitation_url":  "http://localhost:4000/tenant/c1/people/join/?token=cb1",
		"expiration_days": 7,
		"invite_method":   "email",
		"invitation_id":   "inv-9001",
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "INVITATION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "INVITATION_CREATED", Data: data},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("dispatch out=%v err=%v", out, err)
	}
	if gotPath != "/api/internal/tenant/invitations/delivery-callback/inv-9001/" {
		t.Fatalf("unexpected callback path: %s", gotPath)
	}
	if !strings.Contains(gotBody, `"status":"delivered"`) {
		t.Fatalf("expected delivered status in callback body, got %s", gotBody)
	}
}

// 租户邀请投递回调：无 invitation_id（非租户邀请事件）→ 不回调
func TestLocalDeliveryTenantInvitationNoCallbackWithoutID(t *testing.T) {
	called := false
	cbSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer cbSrv.Close()
	t.Setenv("TASK_TENANT_INTERNAL_URL", cbSrv.URL)

	ln, addr := startSMTPServer(t)
	defer ln.Close()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	settings := cfg.Settings{
		Email: cfg.EmailSettings{
			Host: host, Port: port, User: "u", Password: "p",
			DefaultFrom: "from@example.com", UseSSL: false, LocalHostname: "localhost",
		},
	}
	delivery, err := notifications.NewLocalDelivery(settings)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]interface{}{
		"company_name":    "无ID公司",
		"email":           "noid@example.com",
		"invitation_url":  "http://localhost:4000/join/?token=noid",
		"expiration_days": 7,
		"invite_method":   "email",
	})
	out, err := delivery.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "INVITATION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "INVITATION_CREATED", Data: data},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("dispatch out=%v err=%v", out, err)
	}
	if called {
		t.Fatal("expected no callback for event without invitation_id")
	}
}
