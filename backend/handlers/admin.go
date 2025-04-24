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

	// No change needed if role is already set to the requested value
	if user.Role == req.Role {
		c.JSON(http.StatusOK, gin.H{
			"message": "Role already set to " + req.Role,
			"user": gin.H{
				"id":   user.Id,
				"role": user.Role,
			},
		})
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

// ShowAdminDashboard renders the admin dashboard page with users and draft problems
func ShowAdminDashboard(c *gin.Context) {
	// Check if user is logged in and is an admin
	_, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if role != "admin" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// Get tab from query parameter, default to "users"
	tab := c.DefaultQuery("tab", "users")

	// Get all users
	var users []db.User
	if err := db.DB.Find(&users).Error; err != nil {
		log.Printf("Error listing users: %v", err)
		users = []db.User{} // Empty array if error
	}

	log.Printf("Found %d users", len(users))
	for i, user := range users {
		log.Printf("User %d: ID=%d, Username=%s, Role=%s", i, user.Id, user.Username, user.Role)
	}

	// Remove sensitive information
	for i := range users {
		users[i].Password = ""
	}

	// Get all draft problems
	var drafts []db.Question
	if err := db.DB.Where("status = ?", "draft").Preload("Owner").Find(&drafts).Error; err != nil {
		log.Printf("Error listing drafts: %v", err)
		drafts = []db.Question{} // Empty array if error
	}

	log.Printf("Found %d drafts", len(drafts))
	for i, draft := range drafts {
		ownerName := "Unknown"
		if draft.Owner != nil {
			ownerName = draft.Owner.Username
		}
		log.Printf("Draft %d: ID=%d, Title=%s, Owner=%s", i, draft.Id, draft.Title, ownerName)
	}

	// Remove sensitive information from owners
	for i := range drafts {
		if drafts[i].Owner != nil {
			drafts[i].Owner.Password = ""
		}
	}

	c.HTML(http.StatusOK, "admin.gohtml", gin.H{
		"Title":        "Admin Dashboard",
		"IsLoggedIn":   true,
		"UserRole":     role,
		"IsSignupPage": false,
		"ActiveTab":    tab,
		"Users":        users,
		"Drafts":       drafts,
	})
}

// HandleToggleUserRole processes a form submission to change a user's role
func HandleToggleUserRole(c *gin.Context) {
	// Check if user is logged in and is an admin
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if role != "admin" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// Get target user ID from URL
	targetUserId := c.Param("id")
	var targetUserIdUint uint
	_, err = fmt.Sscanf(targetUserId, "%d", &targetUserIdUint)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin?tab=users")
		return
	}

	// Prevent admins from changing their own role
	if userId == targetUserIdUint {
		c.Redirect(http.StatusFound, "/admin?tab=users")
		return
	}

	// Get the user
	var user db.User
	if err := db.DB.First(&user, targetUserId).Error; err != nil {
		c.Redirect(http.StatusFound, "/admin?tab=users")
		return
	}

	// Toggle role between admin and user
	if user.Role == "admin" {
		user.Role = "user"
	} else {
		user.Role = "admin"
	}

	// Save changes
	if err := db.DB.Save(&user).Error; err != nil {
		log.Printf("Error updating user role: %v", err)
		c.Redirect(http.StatusFound, "/admin?tab=users")
		return
	}

	// Redirect back to admin page with users tab active
	c.Redirect(http.StatusFound, "/admin?tab=users")
}

// HandlePublishDraft processes a form submission to publish a draft problem
func HandlePublishDraft(c *gin.Context) {
	// Check if user is logged in and is an admin
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if role != "admin" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// Get problem ID from URL
	problemId := c.Param("id")

	// Get the problem
	var problem db.Question
	if err := db.DB.First(&problem, problemId).Error; err != nil {
		c.Redirect(http.StatusFound, "/admin?tab=drafts")
		return
	}

	// Make sure the problem is a draft
	if problem.Status != "draft" {
		c.Redirect(http.StatusFound, "/admin?tab=drafts")
		return
	}

	// Log the action
	log.Printf("Admin user %d is publishing problem %s", userId, problemId)

	// Publish the problem
	problem.Status = "published"
	if err := db.DB.Save(&problem).Error; err != nil {
		log.Printf("Error publishing problem: %v", err)
		c.Redirect(http.StatusFound, "/admin?tab=drafts")
		return
	}

	// Redirect back to admin page with drafts tab active
	c.Redirect(http.StatusFound, "/admin?tab=drafts")
}
