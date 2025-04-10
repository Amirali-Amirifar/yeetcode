package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func loginUser(c *gin.Context) {
	var username = c.PostForm("email")
	var password = c.PostForm("password")
	fmt.Println(username, password)
	c.Redirect(http.StatusFound, "/login?err=wrong-credentials")
}
