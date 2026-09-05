package consumer

import (
	"log"
	"sync"

	"taskEvents/broker"
	"taskEvents/config"
	notifcfg "taskEvents/notifications/cfg"
	"taskEvents/notifications/smtp"
)

var dltAlertOnce sync.Once

// ensureDLTEmailAlert wires a throttled SMTP alerter for DLT reports. The
// cooldown window is shared across consumer processes via Redis when available,
// otherwise it degrades to a process-local window. Failures are non-fatal:
// missing email config only disables alerts.
func ensureDLTEmailAlert(cfg config.Config) {
	dltAlertOnce.Do(func() {
		root, err := config.FindMonorepoRoot()
		if err != nil {
			log.Printf("[dlt-alert] disabled: find monorepo root: %v", err)
			return
		}
		settings, err := notifcfg.LoadSettings(root)
		if err != nil {
			log.Printf("[dlt-alert] disabled: load email settings: %v", err)
			return
		}
		if settings.Email.Host == "" {
			log.Printf("[dlt-alert] disabled: empty SMTP host")
			return
		}
		to := broker.ResolveDLTAlertEmail()
		cooldown := broker.ResolveDLTAlertCooldown()
		sender := smtp.NewSender(settings.Email)
		store := broker.NewRedisDLTCooldownStore(cfg.RedisHost, cfg.RedisPort, cfg.RedisDB, "")
		broker.SetDLTAlerter(broker.NewThrottledDLTAlerter(broker.DLTAlertConfig{
			To:            to,
			From:          settings.Email.DefaultFrom,
			Cooldown:      cooldown,
			Sender:        sender,
			CooldownStore: store,
		}))
		log.Printf("[dlt-alert] enabled to=%s cooldown=%s smtp=%s:%d store=redis:%s:%d/%d",
			to, cooldown, settings.Email.Host, settings.Email.Port,
			cfg.RedisHost, cfg.RedisPort, cfg.RedisDB)
	})
}
