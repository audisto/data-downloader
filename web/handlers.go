package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func homeHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", nil)
	// c.String(http.StatusOK, "Home")
}

func loginHandler(c *gin.Context) {
	// c.HTML(http.StatusOK, "login.html", nil)
	c.String(http.StatusOK, "login")
}
