package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("successfully hashes password", func(t *testing.T) {
		password := "mysecretpassword"

		hash, err := HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
		// Verify it's a valid bcrypt hash
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		assert.NoError(t, err)
	})

	t.Run("generates different hashes for same password", func(t *testing.T) {
		password := "testpassword"

		hash1, err1 := HashPassword(password)
		hash2, err2 := HashPassword(password)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2, "bcrypt should generate unique salts")
	})

	t.Run("hashes empty password", func(t *testing.T) {
		hash, err := HashPassword("")

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("returns error for password exceeding 72 bytes", func(t *testing.T) {
		// bcrypt has a 72-byte limit
		longPassword := string(make([]byte, 100))

		_, err := HashPassword(longPassword)

		assert.Error(t, err, "bcrypt should reject passwords over 72 bytes")
	})

	t.Run("hashes password at 72 byte limit", func(t *testing.T) {
		// 72 bytes is the max for bcrypt
		maxPassword := string(make([]byte, 72))

		hash, err := HashPassword(maxPassword)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("hashes password with special characters", func(t *testing.T) {
		password := "p@$$w0rd!#$%^&*()_+-=[]{}|;':\",./<>?"

		hash, err := HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		assert.NoError(t, err)
	})

	t.Run("hashes password with unicode characters", func(t *testing.T) {
		password := "密码🔐日本語"

		hash, err := HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		assert.NoError(t, err)
	})
}

func TestCheckPassword(t *testing.T) {
	t.Run("returns nil for matching password", func(t *testing.T) {
		password := "correctpassword"
		hash, _ := HashPassword(password)

		err := CheckPassword(password, hash)

		assert.NoError(t, err)
	})

	t.Run("returns error for wrong password", func(t *testing.T) {
		password := "correctpassword"
		hash, _ := HashPassword(password)

		err := CheckPassword("wrongpassword", hash)

		assert.Error(t, err)
		assert.Equal(t, bcrypt.ErrMismatchedHashAndPassword, err)
	})

	t.Run("returns error for empty password against hash", func(t *testing.T) {
		password := "somepassword"
		hash, _ := HashPassword(password)

		err := CheckPassword("", hash)

		assert.Error(t, err)
	})

	t.Run("returns error for invalid hash format", func(t *testing.T) {
		err := CheckPassword("password", "notavalidhash")

		assert.Error(t, err)
	})

	t.Run("returns error for empty hash", func(t *testing.T) {
		err := CheckPassword("password", "")

		assert.Error(t, err)
	})

	t.Run("case sensitive password comparison", func(t *testing.T) {
		password := "MyPassword"
		hash, _ := HashPassword(password)

		err := CheckPassword("mypassword", hash)

		assert.Error(t, err, "password comparison should be case-sensitive")
	})

	t.Run("handles whitespace in password", func(t *testing.T) {
		password := "  password with spaces  "
		hash, _ := HashPassword(password)

		// Exact match should work
		err := CheckPassword("  password with spaces  ", hash)
		assert.NoError(t, err)

		// Trimmed should fail
		err = CheckPassword("password with spaces", hash)
		assert.Error(t, err)
	})
}

func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword("benchmarkpassword")
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	hash, _ := HashPassword("benchmarkpassword")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CheckPassword("benchmarkpassword", hash)
	}
}
