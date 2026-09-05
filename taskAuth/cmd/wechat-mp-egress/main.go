package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"confload"
	"taskAuth/wechatmpegress"
	"tracelog"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:8030", "listen addr host:port")
	flag.Parse()

	secret := strings.TrimSpace(os.Getenv("WECHAT_MP_EGRESS_INTERNAL_SECRET"))
	if secret == "" {
		// Optional conf overlay when running inside a full monorepo / deploy root.
		if root, err := confload.FindConfigRoot(); err == nil {
			var wrap struct {
				Host   string `yaml:"host"`
				Port   int    `yaml:"port"`
				Secret string `yaml:"internalSecret"`
			}
			if err := confload.ReadAppConfig(root, "infra/wechat-mp-egress", &wrap); err == nil {
				secret = strings.TrimSpace(wrap.Secret)
				if *listen == "0.0.0.0:8030" && wrap.Port > 0 {
					host := wrap.Host
					if host == "" {
						host = "0.0.0.0"
					}
					*listen = fmt.Sprintf("%s:%d", host, wrap.Port)
				}
			}
		}
	}
	if secret == "" {
		log.Fatal("wechat-mp-egress: set WECHAT_MP_EGRESS_INTERNAL_SECRET (or conf-local internalSecret)")
	}
	srv := &wechatmpegress.Server{Secret: secret}
	log.Printf("[wechat-mp-egress] listen %s allowlist=%s", *listen, wechatmpegress.AllowedHost)
	if err := tracelog.ListenAndServe(*listen, srv.Handler()); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
