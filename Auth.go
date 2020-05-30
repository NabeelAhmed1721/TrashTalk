package main

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Auth middleware
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		if session.Get("auth") != true {
			c.Redirect(302, "/login")
		} else {
			// c.Redirect(302, "/dashboard")
			c.Next()
		}
	}
}
