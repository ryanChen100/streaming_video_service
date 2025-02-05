package middlewares

import (
	t_token "streaming_video_service/pkg/token"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	//QueryToken token in query name
	QueryToken = "auth"

	//TokenMemberID get member form token, set c.locals name
	TokenMemberID = "MemberID"
	//TokenRole get role form token, set c.locals name
	TokenRole = "role"
)

// JWTMiddleware validates JWT in the Authorization header
func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		tokenStr := c.Query(QueryToken)

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenStr, &t_token.Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Unexpected signing method")
			}
			return t_token.JWTSecret, nil
		})

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Extract claims and pass them to the context
		if claims, ok := token.Claims.(*t_token.Claims); ok && token.Valid {
			c.Locals(TokenMemberID, claims.MemberID)
			c.Locals(TokenRole, claims.Role)
		} else {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		return c.Next()
	}
}
