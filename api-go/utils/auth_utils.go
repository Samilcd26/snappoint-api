package utils

import (
	"github.com/gin-gonic/gin"
)

type UserClaims struct {
	UserID uint     `json:"user_id"`
	Roles  []string `json:"roles"`
}

type contextKey string

const UserContextKey contextKey = "user"

func GetUser(c *gin.Context) *UserClaims {
	user, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil
	}
	if userClaims, ok := user.(*UserClaims); ok {
		return userClaims
	}
	return nil
}
