package repository

import (
	"database/sql"
	"fmt"

	"github.com/sudhanshu042004/orcs/database"
	"github.com/sudhanshu042004/orcs/types"
)

func CreateUser(username string, name string, email string, avatar string, repoUrl string) (types.JwtPayload, error) {
	var userId int64
	q := `INSERT INTO users(username,name,email,avatar,repo_url) VALUES($1,$2,$3,$4,$5) RETURNING id`
	err := database.DB.QueryRow(q, username, name, email, avatar, repoUrl).Scan(&userId)
	if err != nil {
		fmt.Printf("error while creating user %s\n", err.Error())
		return types.JwtPayload{}, err
	}

	return types.JwtPayload{Id: userId, Email: email}, nil
}

func FindUser(email string) (types.User, error) {
	var existingUser types.User
	q := `SELECT username,name,avatar,id,repo_url,email FROM users WHERE email = $1`

	err := database.DB.QueryRow(q, email).Scan(&existingUser.Username, &existingUser.Name, &existingUser.Avatar, &existingUser.Id, &existingUser.RepoUrl, &existingUser.Email)
	if err == sql.ErrNoRows {
		return types.User{}, nil
	}
	if err != nil {
		return types.User{}, err
	}

	return existingUser, nil
}
