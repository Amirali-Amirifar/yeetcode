package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/Amirali-Amirifar/yeetcode/backend/utils/jwt"
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
	router.GET("/problems/:problem", func(c *gin.Context) {
		c.HTML(http.StatusOK, "problem.gohtml", gin.H{
			"title":      "Problem",
			"page":       "Problem",
			"IsLoggedIn": isLoggedIn(c),
		})
	})

	// Admin dashboard route
	router.GET("/admin", func(c *gin.Context) {
		_, role, err := CheckValidToken(c.Request)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		if role != "admin" {
			c.Redirect(http.StatusFound, "/")
			return
		}

		c.HTML(http.StatusOK, "admin.gohtml", gin.H{
			"title":      "Admin Dashboard",
			"page":       "Admin",
			"IsLoggedIn": true,
			"UserRole":   role,
		})
	})
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
		c.HTML(http.StatusOK, "problems.gohtml", gin.H{
			"title":      "Problems",
			"page":       "Problems",
			"IsLoggedIn": isLoggedIn(c),
		})
	})

	router.GET("/problems/new", ShowCreateProblemPage)
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
