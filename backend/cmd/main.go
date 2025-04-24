package main

import (
	"fmt"
	"github.com/Amirali-Amirifar/yeetcode/backend/config"
	"github.com/Amirali-Amirifar/yeetcode/backend/db"
	"github.com/Amirali-Amirifar/yeetcode/backend/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	config.GetConfig()
	db.Init()

	router := gin.Default()

	router.LoadHTMLGlob("templates/**/*")
	router.Static("/static", "./static")

	handlers.InitHandlers(router)

	router.Run(fmt.Sprintf(":%s", config.GetConfig().ServerPort))
}
