package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/Amirali-Amirifar/yeetcode/backend/scheduler"
	"github.com/gin-gonic/gin"
)

// SubmissionStatus represents the possible statuses of a submission
var SubmissionStatus = struct {
	Pending      string
	CompileError string
	WrongAnswer  string
	MemoryLimit  string
	TimeLimit    string
	RuntimeError string
	Ok           string
}{
	Pending:      "pending",
	CompileError: "compile_error",
	WrongAnswer:  "wrong_answer",
	MemoryLimit:  "memory_limit",
	TimeLimit:    "time_limit",
	RuntimeError: "runtime_error",
	Ok:           "ok",
}

// CreateSubmissionRequest defines the request format for creating a submission
type CreateSubmissionRequest struct {
	Code       string `json:"code" binding:"required"`
	QuestionId uint   `json:"questionId" binding:"required"`
}

// CreateSubmission handles the API endpoint for creating a new submission
func CreateSubmission(c *gin.Context) {
	userId, _, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if question exists and is published
	var question db.Question
	if err := db.DB.First(&question, req.QuestionId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	if question.Status != "published" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot submit to unpublished question"})
		return
	}

	// Create submission
	submission := db.Submission{
		Code:       req.Code,
		Status:     SubmissionStatus.Pending,
		QuestionId: req.QuestionId,
		UserId:     userId,
	}

	if err := db.DB.Create(&submission).Error; err != nil {
		log.Printf("Error creating submission: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create submission"})
		return
	}

	c.JSON(http.StatusCreated, submission)
}

func ListSubmissions(c *gin.Context) {
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var submissions []db.Submission
	query := db.DB.Preload("Question").Preload("User")

	// Require user_id to be provided
	queryUserId := c.Query("user_id")
	if queryUserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id query parameter is required"})
		return
	}

	// Convert user_id to uint
	var queryUserIdUint uint
	if _, err := fmt.Sscanf(queryUserId, "%d", &queryUserIdUint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	// Only allow user to query their own submissions unless admin
	if queryUserIdUint != userId && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	query = query.Where("user_id = ?", queryUserIdUint)

	// Optional: Filter by question
	if c.Query("question_id") != "" {
		query = query.Where("question_id = ?", c.Query("question_id"))
	}

	// Optional: Filter by status
	if c.Query("status") != "" {
		query = query.Where("status = ?", c.Query("status"))
	}

	query = query.Order("created_at DESC")

	if err := query.Find(&submissions).Error; err != nil {
		log.Printf("Error listing submissions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list submissions"})
		return
	}

	// Remove sensitive information
	for i := range submissions {
		if submissions[i].User != nil {
			submissions[i].User.Password = ""
		}
	}

	c.JSON(http.StatusOK, submissions)
}

// GetSubmission handles the API endpoint for getting a single submission
func GetSubmission(c *gin.Context) {
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var submission db.Submission

	if err := db.DB.Preload("Question").Preload("User").First(&submission, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Only the submission owner or an admin can view a submission
	if submission.UserId != userId && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	// Remove sensitive information
	if submission.User != nil {
		submission.User.Password = ""
	}

	c.JSON(http.StatusOK, submission)
}

// UpdateSubmissionStatus handles the API endpoint for updating a submission status
// This would typically be called by the judging service
func UpdateSubmissionStatus(c *gin.Context) {
	// This endpoint would be protected by a service API key in a production system
	// For this demo, we'll just check if the user is an admin
	_, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	id := c.Param("id")
	var submission db.Submission

	if err := db.DB.First(&submission, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Parse request
	var req struct {
		Status string `json:"status" binding:"required"`
		Output string `json:"output"`
		Error  string `json:"error"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update submission status
	submission.Status = req.Status
	submission.Output = req.Output
	submission.Error = req.Error
	now := time.Now()
	submission.ProcessedAt = &now

	if err := db.DB.Save(&submission).Error; err != nil {
		log.Printf("Error updating submission: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update submission"})
		return
	}

	c.JSON(http.StatusOK, submission)
}

// ShowSubmissionPage renders the submission view page
func ShowSubmissionPage(c *gin.Context) {
	// Check if user is logged in
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	submissionId := c.Param("id")
	var submission db.Submission

	// Load submission with related data
	if err := db.DB.Preload("Question").Preload("User").First(&submission, submissionId).Error; err != nil {
		c.Redirect(http.StatusFound, "/problems")
		return
	}

	// Only the submission owner or an admin can view the submission
	if submission.UserId != userId && role != "admin" {
		c.Redirect(http.StatusFound, "/problems")
		return
	}

	data := gin.H{
		"Title":        "Submission Details",
		"IsLoggedIn":   true,
		"UserRole":     role,
		"IsSignupPage": false,
		"Submission":   submission,
	}

	c.HTML(http.StatusOK, "submission.gohtml", data)
}

// ShowSubmitProblemPage renders the page for submitting a solution to a problem
func ShowSubmitProblemPage(c *gin.Context) {
	// Check if user is logged in
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	problemId := c.Param("id")
	var problem db.Question

	// Load problem with test cases
	if err := db.DB.Preload("Owner").First(&problem, problemId).Error; err != nil {
		c.Redirect(http.StatusFound, "/problems")
		return
	}

	// Only published problems can be submitted unless user is admin
	if problem.Status != "published" && role != "admin" && problem.OwnerId != userId {
		c.Redirect(http.StatusFound, "/problems")
		return
	}

	// Get user's previous submissions for this problem
	var submissions []db.Submission
	if err := db.DB.Where("user_id = ? AND question_id = ?", userId, problem.Id).Order("created_at DESC").Limit(5).Find(&submissions).Error; err != nil {
		log.Printf("Error fetching previous submissions: %v", err)
		submissions = []db.Submission{} // Empty array if error
	}

	data := gin.H{
		"Title":         "Submit Solution",
		"IsLoggedInaaa": true,
		"UserRole":      role,
		"IsSignupPage":  false,
		"Problem":       problem,
		"Submissions":   submissions,
	}

	c.HTML(http.StatusOK, "submit.gohtml", data)
}

func HandleSubmitSolution(c *gin.Context) {
	// Check if user is logged in
	userId, _, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Get problem ID and code from form
	problemId := c.Param("id")
	code := c.PostForm("code")

	if code == "" {
		c.Redirect(http.StatusFound, "/problems/"+problemId+"/submit?error=Code+cannot+be+empty")
		return
	}

	// Check if problem exists
	var problem db.Question
	if err := db.DB.First(&problem, problemId).Error; err != nil {
		c.Redirect(http.StatusFound, "/problems")
		return
	}

	// Create submission record using enum
	submission := db.Submission{
		Code:       code,
		Status:     string(scheduler.StatusPending), // Use enum, convert to string if necessary
		QuestionId: problem.Id,
		UserId:     userId,
	}
	if err := db.DB.Create(&submission).Error; err != nil {
		log.Printf("Error creating submission: %v", err)
		c.Redirect(http.StatusFound, "/problems/"+problemId+"/submit?error=Failed+to+create+submission")
		return
	}

	// Redirect to the submissions page
	c.Redirect(http.StatusFound, "/submissions/"+fmt.Sprintf("%d", submission.Id))
}

// ShowUserSubmissionsPage renders the page showing a user's submissions
func ShowUserSubmissionsPage(c *gin.Context) {
	// Check if user is logged in
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Get all submissions for the user
	var submissions []db.Submission
	query := db.DB.Where("user_id = ?", userId).Preload("Question").Order("created_at DESC")

	if err := query.Find(&submissions).Error; err != nil {
		log.Printf("Error listing submissions: %v", err)
		submissions = []db.Submission{} // Empty array if error
	}

	data := gin.H{
		"Title":        "My Submissions",
		"IsLoggedIn":   true,
		"UserRole":     role,
		"IsSignupPage": false,
		"Submissions":  submissions,
	}

	c.HTML(http.StatusOK, "user_submissions.gohtml", data)
}
