// Package auth provides authentication utilities.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

// RefreshTokenGenerator generates and validates refresh tokens.
type RefreshTokenGenerator interface {
	// Generate creates a new refresh token with a new family ID.
	Generate() (token string, familyID string, err error)
	// GenerateWithFamily creates a new token within an existing family.
	GenerateWithFamily(familyID string) (string, error)
	// ExtractFamilyID parses the family ID from a token.
	ExtractFamilyID(token string) (string, error)
	// Hash returns the SHA-256 hash of a token.
	Hash(token string) string
	// CompareHashes securely compares two token hashes.
	CompareHashes(hash1, hash2 string) bool
}

type refreshTokenGenerator struct{}

// NewRefreshTokenGenerator creates a new RefreshTokenGenerator.
func NewRefreshTokenGenerator() RefreshTokenGenerator {
	return &refreshTokenGenerator{}
}

// Generate creates a new refresh token in format: rt_{familyID}_{random}
// - familyID: 16-character hex string (8 bytes)
// - random: 32-character hex string (16 bytes)
func (g *refreshTokenGenerator) Generate() (string, string, error) {
	familyID, err := g.generateRandomHex(8)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate family ID: %w", err)
	}

	random, err := g.generateRandomHex(16)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random part: %w", err)
	}

	token := fmt.Sprintf("rt_%s_%s", familyID, random)
	return token, familyID, nil
}

// GenerateWithFamily creates a new refresh token with an existing family ID.
func (g *refreshTokenGenerator) GenerateWithFamily(familyID string) (string, error) {
	random, err := g.generateRandomHex(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate random part: %w", err)
	}

	token := fmt.Sprintf("rt_%s_%s", familyID, random)
	return token, nil
}

// ExtractFamilyID parses the family ID from a refresh token.
func (g *refreshTokenGenerator) ExtractFamilyID(token string) (string, error) {
	parts := strings.Split(token, "_")
	if len(parts) != 3 || parts[0] != "rt" {
		return "", fmt.Errorf("invalid refresh token format")
	}
	if len(parts[1]) != 16 {
		return "", fmt.Errorf("invalid family ID length")
	}
	// Validate hex characters
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", fmt.Errorf("invalid family ID format: must be hex")
	}
	return parts[1], nil
}

// Hash returns the SHA-256 hash of the token as a hex string.
func (g *refreshTokenGenerator) Hash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// CompareHashes securely compares two token hashes using constant-time comparison.
func (g *refreshTokenGenerator) CompareHashes(hash1, hash2 string) bool {
	return subtle.ConstantTimeCompare([]byte(hash1), []byte(hash2)) == 1
}

// generateRandomHex generates a random hex string of specified byte length.
func (g *refreshTokenGenerator) generateRandomHex(byteLen int) (string, error) {
	bytes := make([]byte, byteLen)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
