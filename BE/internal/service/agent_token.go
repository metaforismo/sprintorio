package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
)

var ErrInvalidAgentToken = errors.New("name must be 1 to 100 characters, scope read, write or full, and expiry 1 to 365 days")

type CreateAgentTokenRequest struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`
	ExpiresInDays *int   `json:"expires_in_days"`
}
type CreatedAgentToken struct {
	Token  string             `json:"token"`
	Record *domain.AgentToken `json:"record"`
}
type AgentTokenService struct {
	repo *repository.AgentTokenRepository
}

func NewAgentTokenService(repo *repository.AgentTokenRepository) *AgentTokenService {
	return &AgentTokenService{repo: repo}
}
func AgentTokenHash(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
func (s *AgentTokenService) Create(ctx context.Context, workspaceID, userID uuid.UUID, req CreateAgentTokenRequest) (*CreatedAgentToken, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Scope == "" {
		req.Scope = "read"
	}
	days := 30
	if req.ExpiresInDays != nil {
		days = *req.ExpiresInDays
	}
	if utf8.RuneCountInString(req.Name) < 1 || utf8.RuneCountInString(req.Name) > 100 || (req.Scope != "read" && req.Scope != "write" && req.Scope != "full") || days < 1 || days > 365 {
		return nil, ErrInvalidAgentToken
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	raw := "spr_" + base64.RawURLEncoding.EncodeToString(secret)
	token := &domain.AgentToken{ID: uuid.New(), WorkspaceID: workspaceID, UserID: userID, Name: req.Name, Prefix: raw[:12], TokenHash: AgentTokenHash(raw), Scope: req.Scope, ExpiresAt: time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)}
	if err := s.repo.Create(ctx, token); err != nil {
		return nil, err
	}
	return &CreatedAgentToken{Token: raw, Record: token}, nil
}
func (s *AgentTokenService) List(ctx context.Context, w, u uuid.UUID) ([]domain.AgentToken, error) {
	return s.repo.List(ctx, w, u)
}
func (s *AgentTokenService) Revoke(ctx context.Context, id, w, u uuid.UUID) (bool, error) {
	return s.repo.Revoke(ctx, id, w, u)
}
func (s *AgentTokenService) Authenticate(ctx context.Context, raw string) (*domain.AgentToken, error) {
	if len(raw) != 47 || !strings.HasPrefix(raw, "spr_") {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw[4:])
	if err != nil || len(decoded) != 32 {
		return nil, nil
	}
	token, err := s.repo.FindActive(ctx, AgentTokenHash(raw))
	if err != nil || token == nil {
		return token, err
	}
	if token.LastUsedAt == nil || time.Since(*token.LastUsedAt) > 15*time.Minute {
		_ = s.repo.Touch(ctx, token.ID)
	}
	return token, nil
}
