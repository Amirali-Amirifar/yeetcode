package handlers

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/url"
)

func isLoggedInMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get("isLoggedIn"); !ok {
			c.Set("IsLoggedIn", isLoggedIn(c))
		}
		c.Next()
	}
}

func authorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println(c.Request.URL.String())
		if c.MustGet("IsLoggedIn").(bool) {
			c.Next()
		} else {
			c.Redirect(http.StatusFound, "/login?err="+url.QueryEscape("Please log in again"))
		}
	}
}
