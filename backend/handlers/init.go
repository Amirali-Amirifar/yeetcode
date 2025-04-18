package handlers

import (
	"net/http"

	"github.com/Amirali-Amirifar/yeetcode/backend/utils/jwt"
	"github.com/gin-gonic/gin"
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
		cookie, err := c.Cookie("session_token")
		if err != nil || !isValidSession(cookie) {
			c.Redirect(http.StatusFound, "/signup")
			return
		}
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
			"title": "Signup",
			"page":  "Signup",
		})
	})

	router.GET("/problems", func(c *gin.Context) {
		c.HTML(http.StatusOK, "problems.gohtml", gin.H{
			"title": "Problems",
			"page":  "Problems",
		})
	})

	router.GET("/problems/:problem", func(c *gin.Context) {
		c.HTML(http.StatusOK, "problem.gohtml", gin.H{
			"title": "Problem",
			"page":  "Problem",
		})
	})
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
	router.POST("/api/problems", func(c *gin.Context) {})
	router.POST("/api/problems/:problem", func(c *gin.Context) {})
}
