package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// --- SMTP fallback (direct send when Kafka is unavailable) ---

// smtpConfig holds SMTP settings for direct email delivery fallback.
type smtpConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	UseSSL      bool
	Timeout     time.Duration
	DefaultFrom string
}

func loadSMTPConfig() smtpConfig {
	// 默认值 ← sync 片段 conf/auth/task-auth/email.yaml（loadEmailConfigFromSyncedFragment）
	out := smtpConfig{
		Host:        "smtp.qq.com",
		Port:        465,
		UseSSL:      true,
		Timeout:     10 * time.Second,
		User:        strings.TrimSpace(cfg.EmailHostUser),
		Password:    strings.TrimSpace(cfg.EmailHostPassword),
		DefaultFrom: strings.TrimSpace(cfg.EmailDefaultFrom),
	}
	if h := strings.TrimSpace(cfg.EmailHost); h != "" {
		out.Host = h
	}
	if cfg.EmailPort > 0 {
		out.Port = cfg.EmailPort
	}
	if cfg.EmailTimeoutSec > 0 {
		out.Timeout = time.Duration(cfg.EmailTimeoutSec) * time.Second
	}
	// EmailUseSSL：YAML 缺省 false 与「未同步」无法区分；仅当已加载 user/host 时采用 YAML 值
	if cfg.EmailHostUser != "" || cfg.EmailHost != "" {
		out.UseSSL = cfg.EmailUseSSL
	}
	// 环境变量最高优先（运维覆盖 / 测试注入）
	if v := strings.TrimSpace(os.Getenv("EMAIL_HOST")); v != "" {
		out.Host = v
	}
	if v := strings.TrimSpace(os.Getenv("EMAIL_PORT")); v != "" {
		fmt.Sscanf(v, "%d", &out.Port)
	}
	if v := strings.TrimSpace(os.Getenv("EMAIL_HOST_USER")); v != "" {
		out.User = v
	}
	if v := strings.TrimSpace(os.Getenv("EMAIL_HOST_PASSWORD")); v != "" {
		out.Password = v
	}
	if v := strings.TrimSpace(os.Getenv("EMAIL_USE_SSL")); v == "false" || v == "0" {
		out.UseSSL = false
	} else if v == "true" || v == "1" {
		out.UseSSL = true
	}
	if v := strings.TrimSpace(os.Getenv("EMAIL_DEFAULT_FROM")); v != "" {
		out.DefaultFrom = v
	}
	if out.DefaultFrom == "" {
		out.DefaultFrom = out.User
	}
	return out
}

// sendEmailSMTP delivers an email directly via SMTP (SSL on 465).
func sendEmailSMTP(from string, to []string, subject, textBody, htmlBody string) error {
	cfg := loadSMTPConfig()
	if cfg.Host == "" {
		return fmt.Errorf("SMTP host not configured")
	}
	if from == "" {
		from = cfg.DefaultFrom
	}
	if from == "" {
		return fmt.Errorf("empty from address")
	}
	if len(to) == 0 {
		return fmt.Errorf("empty recipient list")
	}

	msg := buildSMTPMessage(from, to, subject, textBody, htmlBody)
	// net.JoinHostPort 对 IPv6 主机（如 ::1）自动加方括号，fmt "%s:%d" 会拼出非法地址
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	dialer := &net.Dialer{Timeout: cfg.Timeout}
	if cfg.UseSSL {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: cfg.Host})
		if err != nil {
			return fmt.Errorf("SMTP TLS dial %s: %w", addr, err)
		}
		defer conn.Close()
		return smtpSend(conn, cfg.Host, from, to, msg, cfg.User, cfg.Password)
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("SMTP dial %s: %w", addr, err)
	}
	defer conn.Close()
	return smtpSend(conn, cfg.Host, from, to, msg, cfg.User, cfg.Password)
}

func smtpSend(conn net.Conn, host, from string, to []string, msg []byte, user, password string) error {
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return err
	}
	if user != "" {
		auth := smtp.PlainAuth("", user, password, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(strings.TrimSpace(rcpt)); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// buildSMTPMessage constructs an RFC 2822 email message.
func buildSMTPMessage(from string, to []string, subject, textBody, htmlBody string) []byte {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	if htmlBody != "" {
		boundary := "taskauth-email-boundary"
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		buf.WriteString("\r\n")
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(textBody)
		buf.WriteString("\r\n--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		buf.WriteString(htmlBody)
		buf.WriteString("\r\n--" + boundary + "--\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(textBody)
	}
	return []byte(buf.String())
}
