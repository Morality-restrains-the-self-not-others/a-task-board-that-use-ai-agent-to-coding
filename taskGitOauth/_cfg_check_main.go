package main
import (
  "fmt"
  "os"
  infra "taskGitOauth/infrastructure"
)
func main() {
  wd, _ := os.Getwd()
  _ = wd
  root, err := infra.FindMonorepoRoot()
  if err != nil { fmt.Println("root err", err); os.Exit(1) }
  cfg, err := infra.LoadConfig(root)
  if err != nil { fmt.Println("cfg err", err); os.Exit(1) }
  fmt.Printf("GithubOutboundProxy=%q ClientID=%q\n", cfg.GithubOutboundProxy, cfg.GithubClientID)
}
