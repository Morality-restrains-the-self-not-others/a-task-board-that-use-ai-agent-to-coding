package smtp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"taskEvents/notifications/cfg"
)

// Sender delivers email via SMTP (SSL on 465 or STARTTLS).
type Sender struct {
	cfg cfg.EmailSettings
}

func NewSender(c cfg.EmailSettings) *Sender {
	return &Sender{cfg: c}
}

// Send delivers a multipart/alternative message when htmlBody is non-empty.
func (s *Sender) Send(from string, to []string, subject, textBody, htmlBody string) error {
	return s.SendWithHeaders(from, to, subject, textBody, htmlBody, nil)
}

// SendWithHeaders is Send plus extra RFC 5322 headers (e.g. List-Unsubscribe).
func (s *Sender) SendWithHeaders(from string, to []string, subject, textBody, htmlBody string, extraHeaders map[string]string) error {
	if len(to) == 0 {
		return fmt.Errorf("empty recipient list")
	}
	if from == "" {
		from = s.cfg.DefaultFrom
	}
	if from == "" {
		return fmt.Errorf("empty from address")
	}
	msg := buildMessage(from, to, subject, textBody, htmlBody, extraHeaders)
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	localName := s.cfg.LocalHostname
	if localName == "" {
		localName = "localhost"
	}

	dialer := &net.Dialer{Timeout: s.cfg.Timeout}
	if s.cfg.UseSSL {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.cfg.Host})
		if err != nil {
			return err
		}
		defer conn.Close()
		return s.sendClient(conn, s.cfg.Host, localName, from, to, msg)
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	return s.sendClient(conn, s.cfg.Host, localName, from, to, msg)
}

func (s *Sender) sendClient(conn net.Conn, host, localName, from string, to []string, msg []byte) error {
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Hello(localName); err != nil {
		return err
	}
	if s.cfg.User != "" {
		auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, host)
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

func buildMessage(from string, to []string, subject, textBody, htmlBody string, extraHeaders map[string]string) []byte {
	boundary := "taskEvents-notifications-boundary"
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	for k, v := range extraHeaders {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		buf.WriteString(k + ": " + v + "\r\n")
	}
	buf.WriteString("MIME-Version: 1.0\r\n")
	if htmlBody != "" {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		buf.WriteString("\r\n")
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(textBody)
		buf.WriteString("\r\n")
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		buf.WriteString(htmlBody)
		buf.WriteString("\r\n")
		buf.WriteString("--" + boundary + "--\r\n")
	} else {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(textBody)
	}
	return buf.Bytes()
}
