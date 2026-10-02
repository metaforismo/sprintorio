package service

import (
	"context"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/metaforismo/sprintorio/BE/internal/dto"
	"github.com/metaforismo/sprintorio/BE/internal/repository"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestAuthRegistrationImmediateLoginAndRotationPersistDistinctRefreshTokens(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	db, err := sqlx.Connect("pgx", databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	svc := NewAuthService(repository.NewUserRepository(db), repository.NewRefreshTokenRepository(db), "auth-regression-test-secret")
	email := "refresh-" + uuid.NewString() + "@example.test"
	password := "RegressionPassword123"
	registered, _, registeredToken, err := svc.Register(ctx, dto.RegisterRequest{Email: email, Name: "Refresh Tester", Password: password})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id=$1`, registered.ID) })
	loggedIn, _, loginToken, err := svc.Login(ctx, dto.LoginRequest{Email: email, Password: password})
	require.NoError(t, err)
	require.Equal(t, registered.ID, loggedIn.ID)
	require.NotEqual(t, registeredToken, loginToken)
	_, rotatedToken, err := svc.RefreshTokens(ctx, loginToken)
	require.NoError(t, err)
	require.NotEqual(t, loginToken, rotatedToken)
	require.NotEqual(t, registeredToken, rotatedToken)
	var tokenCount int
	require.NoError(t, db.Get(&tokenCount, `SELECT COUNT(*) FROM refresh_tokens WHERE user_id=$1`, registered.ID))
	require.Equal(t, 2, tokenCount, "registration session and rotated login session remain distinct")
}
