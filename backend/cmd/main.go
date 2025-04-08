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
	router.LoadHTMLGlob("./templates/**/*")

	//router.GET("/login", func(c *gin.Context) {
	//	c.HTML(http.StatusOK, "login", gin.H{
	//		"title": "Login",
	//		"page":  "Login",
	//	})
	//})
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home", gin.H{
			"title": "Home",
			"page":  "home",
		})
	})

	router.Static("/static", "./static")
	router.Run(":8080")
}
