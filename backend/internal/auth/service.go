package auth

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/internal/repository"
	"github.com/sudhanshu042004/orcs/pkg/config"
	"github.com/sudhanshu042004/orcs/types"
)

// json type
type jsonType struct {
	Login      string `json:"login"`
	Avatar_url string `json:"avatar_url"`
	Repos_url  string `json:"repos_url"`
	Email      string `json:"email"`
	Name       string `json:"name"`

	Extra map[string]interface{} `json:"-"`
}

// json parse
func (c *jsonType) jsonParse(data []byte) error {
	type jsonTypeAlias jsonType
	alias := (*jsonTypeAlias)(c)
	if err := json.Unmarshal(data, alias); err != nil {
		//	fmt.Printf("error occured while parsing data %s",err)
		return err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	delete(raw, "username")
	delete(raw, "name")
	delete(raw, "email")
	delete(raw, "avatar_url")
	delete(raw, "repo_url")
	c.Extra = raw

	return nil

}

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

	//github token generation
	token, err := githubCon.Exchange(context.Background(), code)
	if err != nil {
		return
	}

	//requesting userdata
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

	//json conversion
	var data jsonType
	json.Unmarshal(userData, &data)
	log.Printf("user data %s\n", userData)

	//check existing user
	existingUser, err := repository.FindUser(data.Email)
	if err != nil {
		log.Fatalf("%s", err.Error())
		c.AbortWithStatusJSON(500, "Server went up while finding user!!")
		return
	}

	if existingUser != (types.User{}) {
		config.SetCookie(existingUser.Id, existingUser.Name, c)
		return
	}

	//user creation
	newUser, err := repository.CreateUser(data.Login, data.Name, data.Email, data.Avatar_url, data.Repos_url)
	if err != nil {
		c.AbortWithStatusJSON(500, "Something went wrong!!")
		return
	}

	config.SetCookie(newUser.Id, newUser.Email, c)

	return
}
