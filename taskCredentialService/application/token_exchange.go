package application

import (
	"fmt"
	"log"
	"strings"
	"time"

	"taskCredentialService/domain"
)

// ExchangeRefresh exchanges a bootstrap access token for a refresh token.
// First success invalidates access and persists refresh. Subsequent calls that
// prove possession of the original (or still-valid) access return the existing
// refresh (idempotent) so a recreated container can self-heal without a local
// container_refresh_token.json. Unknown access still gets TOKEN_EXCHANGE_ALREADY_DONE
// and must not receive the refresh secret.
func (s *TokenService) ExchangeRefresh(accessToken, businessAPIEndpoint string, scope domain.TaskScope) (string, error) {
	token, err := s.ValidateToken(accessToken, scope)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok && (de.Code == "TOKEN_NOT_FOUND" || de.Code == "TOKEN_EXPIRED") {
			existing, findErr := s.tokenRepo.FindByScope(scope.TenantID, scope.WorkspaceID, scope.TaskID, scope.CommentID)
			if findErr == nil && existing != nil {
				if rt := strings.TrimSpace(existing.ContainerRefreshToken); rt != "" {
					if s.presentedAccessProvesExistingRefresh(accessToken, existing) {
						s.touchBusinessAPIEndpoint(existing.ID, businessAPIEndpoint)
						log.Printf("[task-credential-service] exchange-refresh: IDEMPOTENT task=%s", existing.TaskID)
						return rt, nil
					}
					return "", domain.ErrTokenExchangeAlreadyDone
				}
			}
		}
		return "", err
	}
	if rt := strings.TrimSpace(token.ContainerRefreshToken); rt != "" {
		s.touchBusinessAPIEndpoint(token.ID, businessAPIEndpoint)
		log.Printf("[task-credential-service] exchange-refresh: IDEMPOTENT current-access task=%s", token.TaskID)
		return rt, nil
	}
	refreshToken := generateToken()
	if err := s.tokenRepo.UpdateRefreshToken(token.ID, refreshToken); err != nil {
		return "", fmt.Errorf("exchange refresh: update refresh token: %w", err)
	}
	if err := s.tokenRepo.UpdateAccessToken(token.ID, "", ""); err != nil {
		return "", fmt.Errorf("exchange refresh: clear access token: %w", err)
	}
	s.touchBusinessAPIEndpoint(token.ID, businessAPIEndpoint)
	s.recordAudit(token, "exchange_refresh", "", nil)
	log.Printf("[task-credential-service] exchange-refresh: OK task=%s", token.TaskID)
	return refreshToken, nil
}

// RefreshAccess uses a refresh token to obtain a new access token.
func (s *TokenService) RefreshAccess(refreshToken string, scope domain.TaskScope) (string, string, error) {
	token, err := s.tokenRepo.FindByRefreshToken(refreshToken)
	if err != nil || token == nil {
		return "", "", domain.ErrTokenNotFound
	}
	if token.CompanyID != scope.TenantID || token.WorkspaceID != scope.WorkspaceID || token.TaskID != scope.TaskID {
		return "", "", domain.ErrScopeMismatch
	}
	if domain.CommentConflicts(token.CommentID, scope.CommentID) {
		return "", "", domain.ErrScopeMismatch
	}
	newAccessToken := generateToken()
	expiresAt := time.Now().Add(1 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	if err := s.tokenRepo.UpdateAccessToken(token.ID, newAccessToken, expiresAt); err != nil {
		return "", "", fmt.Errorf("refresh access: update access token: %w", err)
	}
	s.recordAudit(token, "refresh_access", "", nil)
	log.Printf("[task-credential-service] refresh-access: OK task=%s", token.TaskID)
	return newAccessToken, expiresAt, nil
}

func (s *TokenService) touchBusinessAPIEndpoint(tokenID, businessAPIEndpoint string) {
	if strings.TrimSpace(businessAPIEndpoint) == "" {
		return
	}
	if err := s.tokenRepo.UpdateBusinessAPIEndpoint(tokenID, businessAPIEndpoint); err != nil {
		log.Printf("[task-credential-service] WARN exchange refresh: update business_api_endpoint failed: %v", err)
	}
}

// presentedAccessProvesExistingRefresh reports whether the caller holds the
// bootstrap or current access for this row (hash match against token_issued /
// exchange_refresh audit, or plaintext match of the still-stored access).
func (s *TokenService) presentedAccessProvesExistingRefresh(accessToken string, token *domain.ContainerToken) bool {
	presented := strings.TrimSpace(accessToken)
	if presented == "" || token == nil {
		return false
	}
	if cur := strings.TrimSpace(token.ContainerAccessToken); cur != "" && cur == presented {
		return true
	}
	presentedHash := sha256Hex(presented)
	events, err := s.auditRepo.FindByTaskID(token.TaskID, 50)
	if err != nil {
		log.Printf("[task-credential-service] exchange-refresh: audit lookup failed task=%s: %v", token.TaskID, err)
		return false
	}
	for _, ev := range events {
		if ev.EventType != "token_issued" && ev.EventType != "exchange_refresh" {
			continue
		}
		if domain.CommentConflicts(token.CommentID, ev.CommentID) {
			continue
		}
		if ev.AccessTokenSHA256 != "" && ev.AccessTokenSHA256 == presentedHash {
			return true
		}
	}
	return false
}
