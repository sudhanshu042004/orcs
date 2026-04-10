package user

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/internal/repository"
)

func GetUser(c *gin.Context) {
	fmt.Println("this is the issue")
	userEmail := c.MustGet("email").(string)

	userData, err := repository.FindUser(userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "couldn't find user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": userData})
	return
}
