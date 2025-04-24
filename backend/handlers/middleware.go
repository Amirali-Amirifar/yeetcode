package handlers

import (
	"log"
	"net/http"
	"net/url"

	"github.com/Amirali-Amirifar/yeetcode/backend/utils/roles"
	"github.com/gin-gonic/gin"
)

// isLoggedInMiddleware checks if a user is logged in and sets the appropriate context variable
func isLoggedInMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is logged in
		isLoggedIn := isLoggedIn(c)
		c.Set("IsLoggedIn", isLoggedIn)

		// Set user information in context if logged in
		if isLoggedIn {
			userId, role, err := CheckValidToken(c.Request)
			if err == nil {
				c.Set("userId", userId)
				c.Set("userRole", role)
			}
		}

		c.Next()
	}
}

// authorizationMiddleware redirects to login if user is not logged in
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
			// Return HTTP 401 Unauthorized for missing or invalid tokens
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"details": "Valid authentication is required",
			})
			c.Abort()
			return
		}

		if role != requiredRole {
			// Return HTTP 403 Forbidden when the user doesn't have the required role
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"details": "You don't have permission to access this resource",
			})
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
			// Return HTTP 401 Unauthorized for missing or invalid tokens
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"details": "Valid authentication is required",
			})
			c.Abort()
			return
		}

		if !roles.HasPermission(role, permission) {
			// Return HTTP 403 Forbidden when the user doesn't have the required permission
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"details": "You don't have permission to perform this action",
			})
			c.Abort()
			return
		}

		c.Set("userId", userId)
		c.Set("userRole", role)
		c.Next()
	}
}
