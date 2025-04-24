package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"math"

	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/Amirali-Amirifar/yeetcode/backend/utils/jwt"
	"github.com/Amirali-Amirifar/yeetcode/backend/utils/roles"
	"github.com/gin-gonic/gin"
)

// InitHandlers initializes all route handlers for the application
func InitHandlers(router *gin.Engine) {
	router.Use(gin.Logger())
	if gin.Mode() == gin.ReleaseMode {
		router.Use(gin.Recovery())
	}

	router.SetFuncMap(template.FuncMap{})
	router.Use(isLoggedInMiddleware())

	initUnauthorizedTemplates(router)
	initAuthHandlers(router)
	router.Use(authorizationMiddleware())
	initAuthorizedTemplates(router)
	initAuthorizedHandlers(router)
}

// initAuthorizedTemplates sets up routes for authorized pages
func initAuthorizedTemplates(router *gin.Engine) {
	router.GET("/problems/:id", func(c *gin.Context) {
		problemId := c.Param("id")

		// Fetch the problem data
		var problem db.Question
		if err := db.DB.Preload("Owner").Preload("TestCases").First(&problem, problemId).Error; err != nil {
			// If problem not found, redirect to problems list
			c.Redirect(http.StatusFound, "/problems")
			return
		}

		// Get user info for permissions
		userId, role, _ := CheckValidToken(c.Request)

		// Only show published problems to non-owners and non-admins
		if problem.Status != "published" && problem.OwnerId != userId && role != "admin" {
			c.Redirect(http.StatusFound, "/problems")
			return
		}

		// Remove sensitive owner information
		if problem.Owner != nil {
			problem.Owner.Password = ""
		}

		// Fetch recent submissions for this problem
		var submissions []db.Submission
		if err := db.DB.Preload("User").Where("question_id = ?", problem.Id).Order("created_at DESC").Limit(10).Find(&submissions).Error; err != nil {
			log.Printf("Error fetching submissions: %v", err)
			submissions = []db.Submission{}
		}

		c.HTML(http.StatusOK, "problem.gohtml", gin.H{
			"title":       problem.Title,
			"page":        "Problem",
			"IsLoggedIn":  isLoggedIn(c),
			"UserRole":    role,
			"Problem":     problem,
			"Submissions": submissions,
		})
	})

	// Admin dashboard route
	router.GET("/admin", ShowAdminDashboard)

	// User role management route
	router.POST("/admin/users/:id/toggle-role", HandleToggleUserRole)

	// Publish draft problem route
	router.POST("/admin/drafts/:id/publish", HandlePublishDraft)
}

// isLoggedIn checks if the current request has a valid session
func isLoggedIn(c *gin.Context) bool {
	cookie, err := c.Cookie("session_token")
	return err == nil && isValidSession(cookie)
}

// initUnauthorizedTemplates sets up routes for pages that don't require authentication
func initUnauthorizedTemplates(router *gin.Engine) {
	renderLogin := func(c *gin.Context) {
		params := c.Request.URL.Query()
		val := params.Get("err")
		unscaped, err := url.QueryUnescape(val)
		if err != nil {
			log.Fatalf("Error while unescaping login form: %v", err)
		}
		c.HTML(http.StatusOK, "login.gohtml", gin.H{
			"title":        "Login",
			"page":         "Login",
			"err":          unscaped,
			"IsLoggedIn":   isLoggedIn(c),
			"IsSignupPage": false,
		})
	}

	renderSignUp := func(c *gin.Context) {
		params := c.Request.URL.Query()
		val := params.Get("err")
		unscaped, err := url.QueryUnescape(val)
		if err != nil {
			log.Fatalf("Error while unescaping login form: %v", err)
		}
		c.HTML(http.StatusOK, "signup.gohtml", gin.H{
			"title":        "Signup",
			"page":         "Signup",
			"IsSignupPage": true,
			"err":          unscaped,
			"IsLoggedIn":   isLoggedIn(c),
		})
	}

	router.GET("/", func(c *gin.Context) {
		_, role, _ := CheckValidToken(c.Request)
		c.HTML(http.StatusOK, "home.gohtml", gin.H{
			"title":      "Home",
			"page":       "home",
			"IsLoggedIn": isLoggedIn(c),
			"UserRole":   role,
		})
	})

	router.GET("/login", renderLogin)
	router.POST("/login", renderLogin)
	router.GET("/signup", renderSignUp)
	router.POST("/signup", renderSignUp)

	router.GET("/problems", func(c *gin.Context) {
		// Extract query parameters
		difficulty := c.DefaultQuery("difficulty", "")
		searchTerm := c.DefaultQuery("search", "")
		page := 1
		if pageStr := c.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		// Get user role for permissions
		_, role, _ := CheckValidToken(c.Request)

		// Build database query
		query := db.DB.Preload("Owner")

		// Always show only published problems to non-admins
		if !roles.HasPermission(role, "view_all_problems") {
			query = query.Where("status = ?", "published")
		}

		// Filter by difficulty if provided
		if difficulty != "" {
			query = query.Where("LOWER(difficulty) = LOWER(?)", difficulty)
		}

		// Filter by search term if provided
		if searchTerm != "" {
			query = query.Where("title ILIKE ? OR statement ILIKE ?", "%"+searchTerm+"%", "%"+searchTerm+"%")
		}

		// Count total problems for pagination
		var totalCount int64
		query.Model(&db.Question{}).Count(&totalCount)

		// Calculate pagination
		const problemsPerPage = 10
		totalPages := int(math.Ceil(float64(totalCount) / float64(problemsPerPage)))
		if page > totalPages && totalPages > 0 {
			page = totalPages
		}
		offset := (page - 1) * problemsPerPage

		// Get paginated problems
		var problems []db.Question
		query = query.Order("published_at DESC").Offset(offset).Limit(problemsPerPage)
		if err := query.Find(&problems).Error; err != nil {
			log.Printf("Error fetching problems: %v", err)
			c.HTML(http.StatusOK, "problems.gohtml", gin.H{
				"title":        "Problems",
				"page":         "Problems",
				"IsLoggedIn":   isLoggedIn(c),
				"UserRole":     role,
				"ErrorMessage": "Failed to fetch problems",
			})
			return
		}

		// Format problems for display
		var formattedProblems []gin.H
		for _, problem := range problems {
			// Format the difficulty with first letter capitalized
			difficulty := problem.Difficulty
			if len(difficulty) > 0 {
				difficulty = strings.ToUpper(difficulty[:1]) + difficulty[1:]
			}

			// Add each problem
			formattedProblems = append(formattedProblems, gin.H{
				"ID":         problem.Id,
				"Title":      problem.Title,
				"Difficulty": difficulty,
				"Status":     "Unsolved", // Default status
			})
		}

		c.HTML(http.StatusOK, "problems.gohtml", gin.H{
			"title":       "Problems",
			"page":        "Problems",
			"IsLoggedIn":  isLoggedIn(c),
			"UserRole":    role,
			"Problems":    formattedProblems,
			"SearchTerm":  searchTerm,
			"Difficulty":  difficulty,
			"CurrentPage": page,
			"TotalPages":  totalPages,
			"TotalCount":  totalCount,
			"HasPrevPage": page > 1,
			"HasNextPage": page < totalPages,
			"PrevPage":    page - 1,
			"NextPage":    page + 1,
		})
	})

	router.GET("/problems/new", ShowCreateProblemPage)

	// Add route for deleting problems via form submission
	router.POST("/problems/:id/delete", HandleDeleteProblem)

	// Add route for creating problems via form submission
	router.POST("/problems/create", HandleCreateProblem)

	// Add routes for submissions
	router.GET("/problems/:id/submit", ShowSubmitProblemPage)
	router.POST("/problems/:id/submit", HandleSubmitSolution)
	router.GET("/submissions", ShowUserSubmissionsPage)
	router.GET("/submissions/:id", ShowSubmissionPage)
}

// isValidSession validates the given session token
func isValidSession(cookie string) bool {
	_, _, err := jwt.ParseToken(cookie)
	return err == nil
}

type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

// initAuthHandlers sets up authentication-related API routes
func initAuthHandlers(router *gin.Engine) {
	router.POST("/api/login", LoginHandler)
	router.POST("/api/signup", SignUpHandler)
	router.POST("/api/logout", LogoutHandler)
	router.GET("/api/logout", LogoutHandler)
	router.GET("/api/auth/current-user", GetCurrentUser)
}

// initAuthorizedHandlers sets up API routes that require authentication
func initAuthorizedHandlers(router *gin.Engine) {
	// Problem routes with role-based permissions
	problemRoutes := router.Group("/api/problems")
	{
		// All authenticated users can create problems
		problemRoutes.POST("", CreateProblem)

		// All authenticated users can view problems
		problemRoutes.GET("", ListProblems)
		problemRoutes.GET("/:id", GetProblem)

		// Only users with edit permission can update problems
		problemRoutes.PUT("/:id", RequirePermission("edit_problems"), UpdateProblem)

		// Only users with delete permission can delete problems
		problemRoutes.DELETE("/:id", RequirePermission("delete_problems"), DeleteProblem)

		// Only admins can publish problems
		problemRoutes.PUT("/:id/publish", RequirePermission("publish_problems"), PublishProblem)
	}

	// Admin routes
	adminRoutes := router.Group("/api/admin")
	{
		// Only admins can access these routes
		adminRoutes.Use(RequireRole("admin"))
		adminRoutes.GET("/users", ListUsers)
		adminRoutes.PUT("/users/:id/role", UpdateUserRole)
	}
}
