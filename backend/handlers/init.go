package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/Amirali-Amirifar/yeetcode/backend/utils/jwt"
	"github.com/gin-gonic/gin"
)

func InitHandlers(router *gin.Engine) {
	router.Use(gin.Logger())
	if gin.Mode() == gin.ReleaseMode {
		router.Use(gin.Recovery())
	}

	router.SetFuncMap(template.FuncMap{})
	router.Use(isLoggedInMiddleware())

	initUnauthorizedTemplates(router)
	initUnauthorizedHandlers(router)
	router.Use(authorizationMiddleware())
	initAuthorizedTemplates(router)

}

func initAuthorizedTemplates(router *gin.Engine) {

	router.GET("/problems/:problem", func(c *gin.Context) {
		c.HTML(http.StatusOK, "problem.gohtml", gin.H{
			"title":      "Problem",
			"page":       "Problem",
			"IsLoggedIn": isLoggedIn(c),
		})
	})
}

func isLoggedIn(c *gin.Context) bool {
	cookie, err := c.Cookie("session_token")
	return err == nil && isValidSession(cookie)
}

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
		c.HTML(http.StatusOK, "home.gohtml", gin.H{
			"title":      "Home",
			"page":       "home",
			"IsLoggedIn": isLoggedIn(c),
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

func isValidSession(cookie string) bool {
	_, _, err := jwt.ParseToken(cookie)
	return err == nil
}

type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func initUnauthorizedHandlers(router *gin.Engine) {
	router.POST("/api/login", LoginHandler)
	router.POST("/api/signup", SignUpHandler)
	router.POST("/api/logout", LogoutHandler)
	router.GET("/api/logout", LogoutHandler)

	// Problem routes
	router.POST("/api/problems", CreateProblem)
	router.GET("/api/problems", ListProblems)
	router.GET("/api/problems/:id", GetProblem)
	router.PUT("/api/problems/:id", UpdateProblem)
	router.DELETE("/api/problems/:id", DeleteProblem)
}
