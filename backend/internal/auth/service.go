package auth

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/pkg/config"
)

func GithubLogin(c *gin.Context) {
	url := config.AppConfig.Config.AuthCodeURL("randomstate")
	c.Status(303)
	c.Redirect(303, url)
}

func GithubCallback(c *gin.Context) {
	state := c.Query("state")
	if state != "randomstate" {
		return
	}
	code := c.Query("code")
	githubCon := config.GithubConfig()

	token, err := githubCon.Exchange(context.Background(), code)
	if err != nil {
		return
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "BEARER "+token.AccessToken)
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	userData, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	log.Printf("user %s", userData)
	c.Redirect(301, "http://localhost:5173")
}
