package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"sort"
	"strings"
)

// seedOidcBootstrapClients idempotently creates all OIDC bootstrap clients.
func seedOidcBootstrapClients() error {
	for _, c := range cfg.OidcBootstrapClients {
		redirectURIs := []string{c.RedirectURI}
		urisJSON, err := json.Marshal(redirectURIs)
		if err != nil {
			return err
		}
		if err := ensureOidcClient(
			c.ClientID,
			c.ClientSecret,
			c.ClientID, // use clientId as display name
			string(urisJSON),
		); err != nil {
			return err
		}
		log.Printf("[taskAuth] oidc bootstrap: client %s ready", c.ClientID)
	}
	log.Printf("[taskAuth] oidc bootstrap: seeded %d clients checksum=%s",
		len(cfg.OidcBootstrapClients), oidcBootstrapClientChecksum())
	return nil
}

// oidcBootstrapClientChecksum hashes sorted bootstrap client_id values (never secrets).
func oidcBootstrapClientChecksum() string {
	ids := make([]string, 0, len(cfg.OidcBootstrapClients))
	for _, c := range cfg.OidcBootstrapClients {
		id := strings.TrimSpace(c.ClientID)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	return hex.EncodeToString(sum[:])
}
