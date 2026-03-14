package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

var DB *sql.DB

func ConnectDb() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error occured during loading .env")
	}
	dburl := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dburl)
	if err != nil {
		fmt.Println("couldn't connect to the datbase")
	} else {
		DB = db
		fmt.Println("Successfully connected to database!")
	}
}
