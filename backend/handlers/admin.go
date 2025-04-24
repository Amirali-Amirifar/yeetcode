package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/gin-gonic/gin"
)

// UpdateUserRoleRequest defines the request format for updating a user's role
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin user"`
}

// ListUsers returns a list of all users in the system (admin access only)
func ListUsers(c *gin.Context) {
	var users []db.User
	if err := db.DB.Find(&users).Error; err != nil {
		log.Printf("Error listing users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	// Remove sensitive information before sending response
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, users)
}

// UpdateUserRole changes a user's role between 'admin' and 'user' (admin access only)
func UpdateUserRole(c *gin.Context) {
	currentUserId, _, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	targetUserId := c.Param("id")
	var req UpdateUserRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert targetUserId to uint for comparison
	var targetUserIdUint uint
	_, err = fmt.Sscanf(targetUserId, "%d", &targetUserIdUint)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Prevent admins from removing their own admin privileges
	if currentUserId == targetUserIdUint && req.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot remove your own admin privileges"})
		return
	}

	var user db.User
	if err := db.DB.First(&user, targetUserId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user role
	user.Role = req.Role
	if err := db.DB.Save(&user).Error; err != nil {
		log.Printf("Error updating user role: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User role updated successfully",
		"user": gin.H{
			"id":   user.Id,
			"role": user.Role,
		},
	})
}
