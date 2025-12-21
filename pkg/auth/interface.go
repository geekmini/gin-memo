package auth

//go:generate mockgen -destination=mocks/mock_jwt.go -package=mocks gin-sample/pkg/auth TokenManager

// TokenManager defines the interface for JWT token operations.
type TokenManager interface {
	// GenerateToken creates a new JWT token for a user.
	GenerateToken(userID string) (string, error)
	// ValidateToken parses and validates a JWT token, returning the claims if valid.
	ValidateToken(tokenString string) (*Claims, error)
}

// Ensure JWTManager implements TokenManager interface
var _ TokenManager = (*JWTManager)(nil)
