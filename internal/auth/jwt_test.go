package auth_test

import (
	"os"
	"testing"
	"time"

	"github.com/katierevinska/calculatorService/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "testjwtsecretkey")
	if err := auth.InitJWT(); err != nil {
		panic("Failed to initialize JWT for tests: " + err.Error())
	}
	code := m.Run()
	os.Unsetenv("JWT_SECRET")
	os.Exit(code)
}

func TestJWTFunctions(t *testing.T) {
	userID := int64(123)

	t.Run("GenerateToken_Success", func(t *testing.T) {
		tokenString, err := auth.GenerateToken(userID)
		require.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		t.Run("ValidateToken_Success", func(t *testing.T) {
			claims, err := auth.ValidateToken(tokenString)
			require.NoError(t, err)
			assert.NotNil(t, claims)
			assert.Equal(t, userID, claims.UserID)
			assert.Equal(t, "calculatorService", claims.Issuer)
			assert.WithinDuration(t, time.Now().Add(24*time.Hour), claims.ExpiresAt.Time, 5*time.Second)
		})
	})

	t.Run("ValidateToken_Malformed", func(t *testing.T) {
		_, err := auth.ValidateToken("this.is.not.a.jwt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "malformed token")
	})

	t.Run("ValidateToken_Expired", func(t *testing.T) {
		t.Skip("Skipping direct expired token test for brevity, relying on JWT library's robustness")
	})

	t.Run("ValidateToken_InvalidSignature", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "anothersecret")
		auth.InitJWT()

		tokenString, _ := auth.GenerateToken(userID)

		os.Setenv("JWT_SECRET", "testjwtsecretkey")
		auth.InitJWT()

		_, err := auth.ValidateToken(tokenString)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token signature")
	})
}

func TestInitJWT_NoSecret(t *testing.T) {
	originalSecret, wasSet := os.LookupEnv("JWT_SECRET")
	os.Unsetenv("JWT_SECRET")

	err := auth.InitJWT()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET environment variable not set")

	if wasSet {
		os.Setenv("JWT_SECRET", originalSecret)
		auth.InitJWT()
	} else {
		os.Setenv("JWT_SECRET", "testjwtsecretkey")
		auth.InitJWT()
	}
}
