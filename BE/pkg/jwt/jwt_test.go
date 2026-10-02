package jwt

import (
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRefreshTokensAreUniqueAndValidateForSameUser(t *testing.T) {
	userID := uuid.New()
	seenTokens, seenIDs := map[string]bool{}, map[string]bool{}
	for i := 0; i < 100; i++ {
		token, expiresAt, err := GenerateRefreshToken(userID, "test-signing-secret")
		require.NoError(t, err)
		require.False(t, seenTokens[token], "separate sessions must not share a refresh token")
		seenTokens[token] = true
		claims, err := ValidateToken(token, "test-signing-secret")
		require.NoError(t, err)
		require.Equal(t, userID, claims.UserID)
		tokenID, err := uuid.Parse(claims.ID)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, tokenID)
		require.False(t, seenIDs[claims.ID])
		seenIDs[claims.ID] = true
		require.Equal(t, expiresAt.Unix(), claims.ExpiresAt.Unix())
		_, err = ValidateToken(token, "incorrect-secret")
		require.Error(t, err)
	}
}
func TestValidateTokenAcceptsLegacyRefreshTokenWithoutID(t *testing.T) {
	userID := uuid.New()
	claims := Claims{UserID: userID, RegisteredClaims: jwtlib.RegisteredClaims{IssuedAt: jwtlib.NewNumericDate(time.Now()), ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour))}}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte("legacy-secret"))
	require.NoError(t, err)
	parsed, err := ValidateToken(token, "legacy-secret")
	require.NoError(t, err)
	require.Equal(t, userID, parsed.UserID)
	require.Empty(t, parsed.ID)
}
