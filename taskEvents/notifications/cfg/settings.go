package cfg

import (
	"fmt"
	"log"
	"strings"
	"time"

	"confload"
)

// EmailSettings holds SMTP config from sync-generated fragment.
type EmailSettings struct {
	Host          string
	Port          int
	User          string
	Password      string
	DefaultFrom   string
	UseSSL        bool
	Timeout       time.Duration
	LocalHostname string
}

// AliyunSMSSettings holds Aliyun SMS config from sms.aliyun.
type AliyunSMSSettings struct {
	AccessKeyID              string
	AccessKeySecret          string
	RegionID                 string
	SignName                 string
	TemplateCodeNotification string
}

// SMSSettings holds SMS provider config.
type SMSSettings struct {
	Provider string
	Aliyun   AliyunSMSSettings
}

// Settings loaded from sync-generated conf/events/domain-events/{email,sms}.yaml.
// OPT-20260806-057: django.yaml 片段退役（真源 conf/core/{email,sms}.yaml）。
type Settings struct {
	Email EmailSettings
	SMS   SMSSettings
}

type emailYAML struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	HostUser     string `yaml:"host_user"`
	HostPassword string `yaml:"host_password"`
	DefaultFrom  string `yaml:"default_from"`
	UseSSL       bool   `yaml:"use_ssl"`
	Timeout      int    `yaml:"timeout"`
}

type smsYAML struct {
	Provider string `yaml:"provider"`
	Aliyun   struct {
		AccessKeyID              string `yaml:"access_key_id"`
		AccessKeySecret          string `yaml:"access_key_secret"`
		RegionID                 string `yaml:"region_id"`
		SignName                 string `yaml:"sign_name"`
		TemplateCodeNotification string `yaml:"template_code_notification"`
	} `yaml:"aliyun"`
}

type fragmentYAML struct {
	Email emailYAML `yaml:"email"`
	SMS   smsYAML   `yaml:"sms"`
}

// LoadSettings reads email/sms from sync-generated fragments
// conf/events/domain-events/{email,sms}.yaml (no cross-service direct reads).
// ADR-0054：必须经 ReadAppFragment 叠 conf-local 同相对路径，否则 host_password 空串
// 会让 SMTP AUTH 535，而 Kafka 入队成功仍让个人资料页显示「验证码已发送」。
// OPT-20260806-057: django.yaml → email.yaml + sms.yaml（真源 conf/core/{email,sms}.yaml）。
func LoadSettings(root string) (Settings, error) {
	var raw fragmentYAML
	if err := confload.ReadAppFragment(root, "events/domain-events", "email.yaml", &raw); err != nil {
		return Settings{}, fmt.Errorf("read email fragment: %w", err)
	}
	e := raw.Email
	var smsRaw fragmentYAML
	if err := confload.ReadAppFragment(root, "events/domain-events", "sms.yaml", &smsRaw); err == nil {
		raw.SMS = smsRaw.SMS
	}
	timeout := time.Duration(e.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	port := e.Port
	if port == 0 {
		port = 465
	}
	if strings.TrimSpace(e.HostPassword) == "" {
		log.Printf("[notifications] SMTP host_password empty after conf-local overlay host=%s user=%q — EMAIL_SENT AUTH will fail", e.Host, e.HostUser)
	} else {
		log.Printf("[notifications] SMTP loaded host=%s user=%q password_configured=true", e.Host, e.HostUser)
	}
	return Settings{
		Email: EmailSettings{
			Host:          e.Host,
			Port:          port,
			User:          e.HostUser,
			Password:      e.HostPassword,
			DefaultFrom:   firstNonEmpty(e.DefaultFrom, e.HostUser),
			UseSSL:        e.UseSSL,
			Timeout:       timeout,
			LocalHostname: "localhost",
		},
		SMS: SMSSettings{
			Provider: raw.SMS.Provider,
			Aliyun: AliyunSMSSettings{
				AccessKeyID:              raw.SMS.Aliyun.AccessKeyID,
				AccessKeySecret:          raw.SMS.Aliyun.AccessKeySecret,
				RegionID:                 firstNonEmpty(raw.SMS.Aliyun.RegionID, "cn-hangzhou"),
				SignName:                 raw.SMS.Aliyun.SignName,
				TemplateCodeNotification: raw.SMS.Aliyun.TemplateCodeNotification,
			},
		},
	}, nil
}

// LoadSettingsFromConfig uses monorepo root discovered by confload.FindMonorepoRoot.
func LoadSettingsFromConfig() (Settings, string, error) {
	root, err := confload.FindMonorepoRoot()
	if err != nil {
		return Settings{}, "", err
	}
	s, err := LoadSettings(root)
	return s, root, err
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
