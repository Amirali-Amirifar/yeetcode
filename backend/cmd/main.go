package main

import (
	"fmt"
	"html/template"
	"time"

	"github.com/Amirali-Amirifar/yeetcode/backend/config"
	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/Amirali-Amirifar/yeetcode/backend/handlers"

	"github.com/gin-gonic/gin"
)

func formatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d%02d/%02d", year, month, day)
}

func main() {
	config.GetConfig()
	db.Init()

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
