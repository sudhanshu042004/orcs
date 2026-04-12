package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/pkg/config"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("orcsAuth")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unauthorized access"})
			return
		}
		userInfo, err := config.VerifyToken(cookie)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong!!"})
			fmt.Printf("error came up %s", err)
			return
		}
		c.Set("email", userInfo.Email)
		c.Set("id", userInfo.Id)

		c.Next()
	}
}
