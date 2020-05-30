package main

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AlrAuth middleware
func AlrAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		if session.Get("auth") == true {
			c.Redirect(302, "/dashboard")
		} else {
			c.Next()
		}
	}
}
