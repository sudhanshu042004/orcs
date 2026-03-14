package repository

import (
	"fmt"

	"github.com/sudhanshu042004/orcs/database"
)

func CreateUser(username string, name string, email string, avatar string, repoUrl string) (int64, error) {
	res, err := database.DB.Exec("INSERT INTO users(username,name,email,avatar,repoUrl) VALUES($1,$2,$3,$4,$5) RETURNING id", username, name, email, avatar, repoUrl)
	if err != nil {
		fmt.Printf("error while creating user %s\n", err)
		return -1, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		fmt.Printf("error while creating user %s\n", err)
		return -1, err
	}
	return id, nil
}
