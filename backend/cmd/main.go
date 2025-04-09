package main

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func formatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d%02d/%02d", year, month, day)
}

func main() {
	router := gin.Default()
	router.SetFuncMap(template.FuncMap{
		"formatAsDate": formatAsDate,
		"isLoggedIn": func() bool {
			fmt.Println("isLoggedIn")
			return true
		},
	})

	router.LoadHTMLGlob("templates/**/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.gohtml", gin.H{
			"title": "Home",
			"page":  "home",
		})
	})
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.gohtml", gin.H{
			"title": "Login",
			"page":  "Login",
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

	router.Static("/static", "./static")
	router.Run(":8080")
}
