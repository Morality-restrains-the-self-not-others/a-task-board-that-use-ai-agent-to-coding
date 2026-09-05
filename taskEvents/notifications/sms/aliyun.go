package sms

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"taskEvents/notifications/cfg"
)

// AliyunSender sends SMS via dysmsapi (dev mock when config incomplete).
type AliyunSender struct {
	cfg cfg.AliyunSMSSettings
}

func NewAliyunSender(c cfg.AliyunSMSSettings) *AliyunSender {
	return &AliyunSender{cfg: c}
}

func (a *AliyunSender) configured() bool {
	return a.cfg.AccessKeyID != "" &&
		a.cfg.AccessKeySecret != "" &&
		a.cfg.SignName != "" &&
		a.cfg.TemplateCodeNotification != ""
}

// SendNotification mirrors Saas_email SMSService.send_notification dev behavior.
func (a *AliyunSender) SendNotification(phone, content string, templateParams map[string]string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return fmt.Errorf("empty phone")
	}
	if !a.configured() {
		log.Printf("[notifications-sms] mock send to %s: %s", phone, content)
		return nil
	}
	// Production path: use Aliyun template API (content logged; template drives body).
	params := url.Values{}
	params.Set("PhoneNumbers", phone)
	params.Set("SignName", a.cfg.SignName)
	params.Set("TemplateCode", a.cfg.TemplateCodeNotification)
	if len(templateParams) > 0 {
		raw, _ := json.Marshal(templateParams)
		params.Set("TemplateParam", string(raw))
	}
	endpoint := fmt.Sprintf("https://dysmsapi.aliyuncs.com/?Action=SendSms&Version=2017-05-25&RegionId=%s&Format=JSON&%s",
		url.QueryEscape(a.cfg.RegionID), params.Encode())
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("aliyun sms http %d: %s", resp.StatusCode, string(body))
	}
	log.Printf("[notifications-sms] aliyun response: %s", string(body))
	return nil
}
