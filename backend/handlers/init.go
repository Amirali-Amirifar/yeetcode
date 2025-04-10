package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func InitHandlers(router *gin.Engine) {
	router.Use(gin.Logger())
	if gin.Mode() == gin.ReleaseMode {
		router.Use(gin.Recovery())
	}

	initTemplateHandlers(router)
	initUnauthorizedHandlers(router)
}

func initTemplateHandlers(router *gin.Engine) {
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.gohtml", gin.H{
			"title": "Home",
			"page":  "home",
		})
	})

	router.GET("/login", func(c *gin.Context) {
		params := c.Request.URL.Query()
		err := params.Get("err")

		c.HTML(http.StatusOK, "login.gohtml", gin.H{
			"title": "Login",
			"page":  "Login",
			"err":   err,
		})
	})

	router.GET("/signup", func(c *gin.Context) {
		c.HTML(http.StatusOK, "signup.gohtml", gin.H{
			"title": "Login",
			"page":  "Login",
		})
	})

	router.GET("/problems", func(c *gin.Context) {
		c.HTML(http.StatusOK, "problems.gohtml", gin.H{
			"title": "Login",
			"page":  "Login",
		})
	})

	router.GET("/problems/:problem", func(c *gin.Context) {
		c.HTML(http.StatusOK, "problem.gohtml", gin.H{
			"title": "Login",
			"page":  "Login",
		})
	})
}

type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func initUnauthorizedHandlers(router *gin.Engine) {
	router.POST("/api/login", func(c *gin.Context) {
		loginUser(c)
		return
	})
	router.POST("/api/signup", func(c *gin.Context) {})
	router.POST("/api/logout", func(c *gin.Context) {})
	router.POST("/api/problems", func(c *gin.Context) {})
	router.POST("/api/problems/:problem", func(c *gin.Context) {})
}
