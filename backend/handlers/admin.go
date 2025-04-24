package handlers

import (
	"log"
	"net/http"

	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/gin-gonic/gin"
)

type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin user"`
}

func ListUsers(c *gin.Context) {
	var users []db.User
	if err := db.DB.Find(&users).Error; err != nil {
		log.Printf("Error listing users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	// Remove sensitive information
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, users)
}

func UpdateUserRole(c *gin.Context) {
	userId := c.Param("id")
	var req UpdateUserRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user db.User
	if err := db.DB.First(&user, userId).Error; err != nil {
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
