package main

import (
	"fmt"
	"github.com/Amirali-Amirifar/yeetcode/backend/config"
	"github.com/Amirali-Amirifar/yeetcode/backend/handlers"
	"html/template"
	"time"

	"github.com/gin-gonic/gin"
)

func formatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d%02d/%02d", year, month, day)
}

func main() {
	// Initialize config
	config.GetConfig()

	router := gin.Default()
	router.SetFuncMap(template.FuncMap{
		"formatAsDate": formatAsDate,
		"isLoggedIn": func() bool {
			fmt.Println("isLoggedIn")
			return true
		},
	})

	router.LoadHTMLGlob("templates/**/*")

	handlers.InitHandlers(router)

	router.Static("/static", "./static")
	router.Run(fmt.Sprintf(":%s", config.GetConfig().ServerPort))
}
