package middleware

import (
	"strings"
	"time"

	"apartment-backend/internal/config"
	"apartment-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID uuid.UUID       `json:"user_id"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

func AuthRequired(cfg config.JWTConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := extractToken(c)
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse("Missing or invalid authorization token"))
		}

		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
			}
			return []byte(cfg.AccessSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse("Invalid or expired token"))
		}

		c.Locals("userID", claims.UserID)
		c.Locals("userEmail", claims.Email)
		c.Locals("userRole", claims.Role)

		return c.Next()
	}
}

func RoleRequired(roles ...models.UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(models.UserRole)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse("Unauthorized"))
		}

		for _, role := range roles {
			if userRole == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Insufficient permissions"))
	}
}

func GenerateAccessToken(cfg config.JWTConfig, user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(cfg.AccessExpiry)
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.AccessSecret))
	return tokenString, expiresAt, err
}

func GenerateRefreshToken(cfg config.JWTConfig, userID uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().Add(cfg.RefreshExpiry)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.RefreshSecret))
	return tokenString, expiresAt, err
}

func ParseRefreshToken(cfg config.JWTConfig, tokenString string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.RefreshSecret), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid refresh token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token subject")
	}

	return userID, nil
}

func ParseAccessToken(cfg config.JWTConfig, tokenString string, claims *JWTClaims) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
		}
		return []byte(cfg.AccessSecret), nil
	})
}

func extractToken(c *fiber.Ctx) string {
	// Check Authorization header
	authHeader := c.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Check query param (for WebSocket)
	if token := c.Query("token"); token != "" {
		return token
	}

	return ""
}

func GetUserID(c *fiber.Ctx) uuid.UUID {
	userID, _ := c.Locals("userID").(uuid.UUID)
	return userID
}

func GetUserRole(c *fiber.Ctx) models.UserRole {
	role, _ := c.Locals("userRole").(models.UserRole)
	return role
}
