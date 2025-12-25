package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefreshTokenGenerator(t *testing.T) {
	t.Run("creates generator", func(t *testing.T) {
		gen := NewRefreshTokenGenerator()

		assert.NotNil(t, gen)
	})
}

func TestRefreshTokenGenerator_Generate(t *testing.T) {
	gen := NewRefreshTokenGenerator()

	t.Run("generates valid token format", func(t *testing.T) {
		token, familyID, err := gen.Generate()

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotEmpty(t, familyID)

		// Token format: rt_{familyID}_{random}
		parts := strings.Split(token, "_")
		assert.Len(t, parts, 3)
		assert.Equal(t, "rt", parts[0])
		assert.Equal(t, familyID, parts[1])
		assert.Len(t, parts[1], 16) // familyID is 16 hex chars (8 bytes)
		assert.Len(t, parts[2], 32) // random is 32 hex chars (16 bytes)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, familyID1, _ := gen.Generate()
		token2, familyID2, _ := gen.Generate()

		assert.NotEqual(t, token1, token2)
		assert.NotEqual(t, familyID1, familyID2)
	})

	t.Run("familyID matches token prefix", func(t *testing.T) {
		token, familyID, _ := gen.Generate()

		extractedFamilyID, err := gen.ExtractFamilyID(token)
		require.NoError(t, err)
		assert.Equal(t, familyID, extractedFamilyID)
	})
}

func TestRefreshTokenGenerator_GenerateWithFamily(t *testing.T) {
	gen := NewRefreshTokenGenerator()

	t.Run("generates token with given family ID", func(t *testing.T) {
		familyID := "1234567890abcdef" // 16 hex chars

		token, err := gen.GenerateWithFamily(familyID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify token contains the family ID
		parts := strings.Split(token, "_")
		assert.Len(t, parts, 3)
		assert.Equal(t, "rt", parts[0])
		assert.Equal(t, familyID, parts[1])
	})

	t.Run("generates different tokens for same family", func(t *testing.T) {
		familyID := "1234567890abcdef"

		token1, _ := gen.GenerateWithFamily(familyID)
		token2, _ := gen.GenerateWithFamily(familyID)

		assert.NotEqual(t, token1, token2)
		// But both should have same family ID
		assert.True(t, strings.Contains(token1, familyID))
		assert.True(t, strings.Contains(token2, familyID))
	})
}

func TestRefreshTokenGenerator_ExtractFamilyID(t *testing.T) {
	gen := NewRefreshTokenGenerator()

	t.Run("extracts family ID from valid token", func(t *testing.T) {
		token := "rt_1234567890abcdef_fedcba0987654321fedcba0987654321"

		familyID, err := gen.ExtractFamilyID(token)

		require.NoError(t, err)
		assert.Equal(t, "1234567890abcdef", familyID)
	})

	t.Run("extracts family ID from generated token", func(t *testing.T) {
		token, expectedFamilyID, _ := gen.Generate()

		familyID, err := gen.ExtractFamilyID(token)

		require.NoError(t, err)
		assert.Equal(t, expectedFamilyID, familyID)
	})

	t.Run("returns error for invalid prefix", func(t *testing.T) {
		token := "xx_1234567890abcdef_fedcba0987654321fedcba0987654321"

		_, err := gen.ExtractFamilyID(token)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token format")
	})

	t.Run("returns error for wrong number of parts", func(t *testing.T) {
		_, err := gen.ExtractFamilyID("rt_only_two")

		assert.Error(t, err)
		// Note: This actually has 3 parts, let's test with actual wrong count
	})

	t.Run("returns error for too few parts", func(t *testing.T) {
		_, err := gen.ExtractFamilyID("rt_onlyonepart")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token format")
	})

	t.Run("returns error for invalid family ID length", func(t *testing.T) {
		token := "rt_short_fedcba0987654321fedcba0987654321"

		_, err := gen.ExtractFamilyID(token)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID length")
	})

	t.Run("returns error for empty token", func(t *testing.T) {
		_, err := gen.ExtractFamilyID("")

		assert.Error(t, err)
	})

	t.Run("returns error for non-hex family ID", func(t *testing.T) {
		// Valid structure (16 chars) but family ID contains non-hex characters (g, h, i, j)
		token := "rt_ghij567890abcdef_fedcba0987654321fedcba0987654321"

		_, err := gen.ExtractFamilyID(token)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID format: must be hex")
	})
}

func TestRefreshTokenGenerator_Hash(t *testing.T) {
	gen := NewRefreshTokenGenerator()

	t.Run("returns consistent hash for same token", func(t *testing.T) {
		token := "rt_1234567890abcdef_fedcba0987654321fedcba0987654321"

		hash1 := gen.Hash(token)
		hash2 := gen.Hash(token)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("returns different hashes for different tokens", func(t *testing.T) {
		token1 := "rt_1234567890abcdef_fedcba0987654321fedcba0987654321"
		token2 := "rt_1234567890abcdef_fedcba0987654321fedcba0987654322"

		hash1 := gen.Hash(token1)
		hash2 := gen.Hash(token2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("returns 64 character hex hash (SHA-256)", func(t *testing.T) {
		token := "rt_1234567890abcdef_fedcba0987654321fedcba0987654321"

		hash := gen.Hash(token)

		assert.Len(t, hash, 64) // SHA-256 produces 32 bytes = 64 hex chars
	})

	t.Run("hash is valid hex string", func(t *testing.T) {
		token := "rt_1234567890abcdef_fedcba0987654321fedcba0987654321"

		hash := gen.Hash(token)

		// All characters should be hex digits
		for _, c := range hash {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
				"character %c is not a hex digit", c)
		}
	})
}

func TestRefreshTokenGenerator_CompareHashes(t *testing.T) {
	gen := NewRefreshTokenGenerator()

	t.Run("returns true for identical hashes", func(t *testing.T) {
		hash := gen.Hash("rt_1234567890abcdef_fedcba0987654321fedcba0987654321")

		result := gen.CompareHashes(hash, hash)

		assert.True(t, result)
	})

	t.Run("returns false for different hashes", func(t *testing.T) {
		hash1 := gen.Hash("token1")
		hash2 := gen.Hash("token2")

		result := gen.CompareHashes(hash1, hash2)

		assert.False(t, result)
	})

	t.Run("returns false for partial match", func(t *testing.T) {
		hash := gen.Hash("token")
		partialHash := hash[:len(hash)-1] + "x"

		result := gen.CompareHashes(hash, partialHash)

		assert.False(t, result)
	})

	t.Run("returns true for hashes of same token", func(t *testing.T) {
		token, _, _ := gen.Generate()
		hash1 := gen.Hash(token)
		hash2 := gen.Hash(token)

		result := gen.CompareHashes(hash1, hash2)

		assert.True(t, result)
	})
}

func TestRefreshTokenGenerator_Interface(t *testing.T) {
	t.Run("implements RefreshTokenGenerator interface", func(t *testing.T) {
		var _ RefreshTokenGenerator = (*refreshTokenGenerator)(nil)
	})
}

func BenchmarkRefreshTokenGenerator_Generate(b *testing.B) {
	gen := NewRefreshTokenGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = gen.Generate()
	}
}

func BenchmarkRefreshTokenGenerator_Hash(b *testing.B) {
	gen := NewRefreshTokenGenerator()
	token := "rt_1234567890abcdef_fedcba0987654321fedcba0987654321"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.Hash(token)
	}
}

func BenchmarkRefreshTokenGenerator_CompareHashes(b *testing.B) {
	gen := NewRefreshTokenGenerator()
	hash1 := gen.Hash("token1")
	hash2 := gen.Hash("token2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.CompareHashes(hash1, hash2)
	}
}
