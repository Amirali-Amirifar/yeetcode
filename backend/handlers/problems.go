package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/Amirali-Amirifar/yeetcode/backend/utils/roles"
	"github.com/gin-gonic/gin"
)

type CreateProblemRequest struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description" binding:"required"`
	TimeLimit   int        `json:"timeLimit" binding:"required"`
	MemoryLimit int        `json:"memoryLimit" binding:"required"`
	Difficulty  string     `json:"difficulty" binding:"required"`
	TestCases   []TestCase `json:"testCases" binding:"required"`
	Status      string     `json:"status"`
}

type TestCase struct {
	Input  string `json:"input" binding:"required"`
	Output string `json:"output" binding:"required"`
}

type UpdateProblemRequest struct {
	Title       string `json:"title"`
	Statement   string `json:"statement"`
	TimeLimit   int    `json:"timeLimit"`
	MemoryLimit int    `json:"memoryLimit"`
	Input       string `json:"input"`
	Output      string `json:"output"`
	Status      string `json:"status"`
}

type PublishProblemRequest struct {
	Status string `json:"status" binding:"required,oneof=draft published"`
}

func CreateProblem(c *gin.Context) {
	userId, _, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate difficulty
	if req.Difficulty != "easy" && req.Difficulty != "medium" && req.Difficulty != "hard" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid difficulty level"})
		return
	}

	question := db.Question{
		Title:       req.Title,
		Statement:   req.Description,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		Difficulty:  req.Difficulty,
		Status:      "draft",
		OwnerId:     userId,
	}

	if err := db.DB.Create(&question).Error; err != nil {
		log.Printf("Error creating question: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create question"})
		return
	}

	// Create test cases
	for _, tc := range req.TestCases {
		testCase := db.TestCase{
			QuestionId: question.Id,
			Input:      tc.Input,
			Output:     tc.Output,
		}

		if err := db.DB.Create(&testCase).Error; err != nil {
			log.Printf("Error creating test case: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create test case"})
			return
		}
	}

	c.JSON(http.StatusCreated, question)
}

func GetProblem(c *gin.Context) {
	id := c.Param("id")
	var question db.Question

	if err := db.DB.Preload("Owner").Preload("TestCases").First(&question, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	c.JSON(http.StatusOK, question)
}

func UpdateProblem(c *gin.Context) {
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var question db.Question

	// First, check if question exists and user has permission
	if err := db.DB.First(&question, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	// Only owner or admin can update
	if question.OwnerId != userId && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	var req UpdateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields if provided
	if req.Title != "" {
		question.Title = req.Title
	}
	if req.Statement != "" {
		question.Statement = req.Statement
	}
	if req.TimeLimit != 0 {
		question.TimeLimit = req.TimeLimit
	}
	if req.MemoryLimit != 0 {
		question.MemoryLimit = req.MemoryLimit
	}
	if req.Status != "" {
		question.Status = req.Status
	}

	if err := db.DB.Save(&question).Error; err != nil {
		log.Printf("Error updating question: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update question"})
		return
	}

	// Update test case if input/output provided
	if req.Input != "" || req.Output != "" {
		var testCase db.TestCase
		if err := db.DB.Where("question_id = ?", question.Id).First(&testCase).Error; err == nil {
			if req.Input != "" {
				testCase.Input = req.Input
			}
			if req.Output != "" {
				testCase.Output = req.Output
			}
			if err := db.DB.Save(&testCase).Error; err != nil {
				log.Printf("Error updating test case: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update test case"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, question)
}

func ListProblems(c *gin.Context) {
	_, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var questions []db.Question
	query := db.DB.Preload("Owner").Preload("TestCases")

	// Filter by status if provided
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	} else {
		// If no status filter, show only published questions to non-admins
		if !roles.HasPermission(role, "view_all_problems") {
			query = query.Where("status = ?", "published")
		}
	}

	// Filter by owner if provided
	if ownerId := c.Query("owner_id"); ownerId != "" {
		query = query.Where("owner_id = ?", ownerId)
	}

	// Order by published date for published questions
	if c.Query("status") == "published" || (!roles.HasPermission(role, "view_all_problems") && c.Query("status") == "") {
		query = query.Order("published_at DESC")
	} else {
		query = query.Order("created_at DESC")
	}

	if err := query.Find(&questions).Error; err != nil {
		log.Printf("Error listing questions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list questions"})
		return
	}

	// Remove sensitive information from response
	for i := range questions {
		if questions[i].Owner != nil {
			questions[i].Owner.Password = ""
		}
	}

	c.JSON(http.StatusOK, questions)
}

func DeleteProblem(c *gin.Context) {
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var question db.Question

	// First, check if question exists and user has permission
	if err := db.DB.First(&question, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	// Only owner or admin can delete
	if question.OwnerId != userId && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	if err := db.DB.Delete(&question).Error; err != nil {
		log.Printf("Error deleting question: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete question"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question deleted successfully"})
}

// ShowCreateProblemPage renders the page for creating a new problem.
func ShowCreateProblemPage(c *gin.Context) {
	isLoggedIn := false
	userId := uint(0)
	var drafts []db.Question

	userInterface, exists := c.Get("user")
	if exists {
		u, ok := userInterface.(db.User)
		if ok && u.Id != 0 {
			isLoggedIn = true
			userId = u.Id
			if err := db.DB.Where("owner_id = ? AND status = ?", userId, "draft").Order("created_at desc").Find(&drafts).Error; err != nil {
				log.Printf("Error fetching user drafts: %v", err)
			}
		}
	}

	data := gin.H{
		"Title":        "Create New Problem",
		"IsLoggedIn":   isLoggedIn,
		"IsSignupPage": false,
		"Drafts":       drafts,
		"Error":        c.Query("error"),
	}

	c.HTML(http.StatusOK, "create_problem.gohtml", data)
}

func PublishProblem(c *gin.Context) {
	_, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if !roles.HasPermission(role, "publish_problems") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	id := c.Param("id")
	var req PublishProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var question db.Question
	if err := db.DB.First(&question, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	// Check if the problem is already in the requested status
	if question.Status == req.Status {
		c.JSON(http.StatusOK, gin.H{
			"message": "Question status already set to " + req.Status,
			"question": gin.H{
				"id":     question.Id,
				"title":  question.Title,
				"status": question.Status,
			},
		})
		return
	}

	// Update status and published_at if being published
	if req.Status == "published" {
		// Check if the question has test cases before publishing
		var testCaseCount int64
		if err := db.DB.Model(&db.TestCase{}).Where("question_id = ?", question.Id).Count(&testCaseCount).Error; err != nil {
			log.Printf("Error checking test cases: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check test cases"})
			return
		}

		if testCaseCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot publish question without test cases"})
			return
		}

		now := time.Now()
		question.PublishedAt = &now
	} else {
		question.PublishedAt = nil
	}
	question.Status = req.Status

	if err := db.DB.Save(&question).Error; err != nil {
		log.Printf("Error updating question status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update question status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Question status updated successfully",
		"question": gin.H{
			"id":          question.Id,
			"title":       question.Title,
			"status":      question.Status,
			"publishedAt": question.PublishedAt,
		},
	})
}

// HandlePublishProblem handles HTTP form submissions for publishing/unpublishing problems
func HandlePublishProblem(c *gin.Context) {
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Check permissions - only admins can publish/unpublish
	if !roles.HasPermission(role, "publish_problems") {
		c.Redirect(http.StatusFound, "/problems")
		return
	}

	// Get problem ID from URL
	problemId := c.Param("id")
	var question db.Question
	if err := db.DB.First(&question, problemId).Error; err != nil {
		c.Redirect(http.StatusFound, "/problems")
		return
	}

	// Get the desired status from form
	status := c.PostForm("status")
	if status != "published" && status != "draft" {
		c.Redirect(http.StatusFound, "/problems/"+problemId)
		return
	}

	// Check if the problem is already in the requested status
	if question.Status == status {
		c.Redirect(http.StatusFound, "/problems/"+problemId)
		return
	}

	// Update status and published_at if being published
	if status == "published" {
		// Check if the question has test cases before publishing
		var testCaseCount int64
		if err := db.DB.Model(&db.TestCase{}).Where("question_id = ?", question.Id).Count(&testCaseCount).Error; err != nil {
			log.Printf("Error checking test cases: %v", err)
			c.Redirect(http.StatusFound, "/problems/"+problemId)
			return
		}

		if testCaseCount == 0 {
			c.Redirect(http.StatusFound, "/problems/"+problemId)
			return
		}

		now := time.Now()
		question.PublishedAt = &now
	} else {
		question.PublishedAt = nil
	}
	question.Status = status

	if err := db.DB.Save(&question).Error; err != nil {
		log.Printf("Error updating question status: %v", err)
		c.Redirect(http.StatusFound, "/problems/"+problemId)
		return
	}

	log.Printf("User %d changed problem %s status to %s", userId, problemId, status)

	// Redirect back to the problem page
	c.Redirect(http.StatusFound, "/problems/"+problemId)
}

// HandleDeleteProblem processes a form submission to delete a problem
func HandleDeleteProblem(c *gin.Context) {
	// Check if user is logged in
	userId, role, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Get problem ID from URL
	problemId := c.Param("id")
	var question db.Question

	// Check if question exists
	if err := db.DB.First(&question, problemId).Error; err != nil {
		// If problem not found, redirect to create problem page
		c.Redirect(http.StatusFound, "/problems/new")
		return
	}

	// Only owner or admin can delete
	if question.OwnerId != userId && role != "admin" {
		c.Redirect(http.StatusFound, "/problems/new")
		return
	}

	// Delete the problem
	if err := db.DB.Delete(&question).Error; err != nil {
		log.Printf("Error deleting question: %v", err)
		c.Redirect(http.StatusFound, "/problems/new")
		return
	}

	// Redirect back to the create problem page
	c.Redirect(http.StatusFound, "/problems/new")
}

// HandleCreateProblem processes a form submission to create a new problem
func HandleCreateProblem(c *gin.Context) {
	// Check if user is logged in
	userId, _, err := CheckValidToken(c.Request)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Parse form data
	title := c.PostForm("title")
	description := c.PostForm("description")
	difficulty := strings.ToLower(c.PostForm("difficulty"))

	// Parse timeLimit and memoryLimit from form
	timeLimit, err := strconv.Atoi(c.PostForm("timeLimit"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "Invalid time limit format",
		})
		return
	}

	memoryLimit, err := strconv.Atoi(c.PostForm("memoryLimit"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "Invalid memory limit format",
		})
		return
	}

	// Parse test cases from JSON string
	testCasesJSON := c.PostForm("testCases")
	var testCases []TestCase
	if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
		c.HTML(http.StatusBadRequest, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "Invalid test cases format: " + err.Error(),
		})
		return
	}

	// Validate form data
	if title == "" || description == "" || len(testCases) == 0 {
		c.HTML(http.StatusBadRequest, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "All fields are required",
		})
		return
	}

	// Validate difficulty
	if difficulty != "easy" && difficulty != "medium" && difficulty != "hard" {
		c.HTML(http.StatusBadRequest, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "Invalid difficulty level",
		})
		return
	}

	// Create the question in the database
	question := db.Question{
		Title:       title,
		Statement:   description,
		TimeLimit:   timeLimit,
		MemoryLimit: memoryLimit,
		Difficulty:  difficulty,
		Status:      "draft",
		OwnerId:     userId,
	}

	// Start a transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		c.HTML(http.StatusInternalServerError, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "Failed to create problem",
		})
		return
	}

	// Create the question
	if err := tx.Create(&question).Error; err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "Failed to create problem",
		})
		return
	}

	// Create test cases
	for _, tc := range testCases {
		testCase := db.TestCase{
			QuestionId: question.Id,
			Input:      tc.Input,
			Output:     tc.Output,
		}
		if err := tx.Create(&testCase).Error; err != nil {
			tx.Rollback()
			c.HTML(http.StatusInternalServerError, "create_problem.gohtml", gin.H{
				"Title":        "Create New Problem",
				"IsLoggedIn":   true,
				"IsSignupPage": false,
				"Error":        "Failed to create test cases",
			})
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.HTML(http.StatusInternalServerError, "create_problem.gohtml", gin.H{
			"Title":        "Create New Problem",
			"IsLoggedIn":   true,
			"IsSignupPage": false,
			"Error":        "Failed to create problem",
		})
		return
	}

	// Redirect to the problem page
	c.Redirect(http.StatusFound, "/problems/"+fmt.Sprintf("%d", question.Id))
}
