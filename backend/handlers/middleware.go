package handlers

import (
	"log"
	"net/http"
	"net/url"

	"github.com/Amirali-Amirifar/yeetcode/backend/utils/roles"
	"github.com/gin-gonic/gin"
)

func isLoggedInMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get("isLoggedIn"); !ok {
			c.Set("IsLoggedIn", isLoggedIn(c))
		}
		c.Next()
	}
}

func authorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println(c.Request.URL.String())
		if c.MustGet("IsLoggedIn").(bool) {
			c.Next()
		} else {
			c.Redirect(http.StatusFound, "/login?err="+url.QueryEscape("Please log in again"))
		}
	}
}

// RequireRole creates a middleware that checks if the user has the required role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, role, err := CheckValidToken(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Set("userId", userId)
		c.Set("userRole", role)
		c.Next()
	}
}

// RequirePermission creates a middleware that checks if the user has the required permission
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, role, err := CheckValidToken(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		if !roles.HasPermission(role, permission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Set("userId", userId)
		c.Set("userRole", role)
		c.Next()
	}
}
