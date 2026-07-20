package database

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/lib/pq"

	"github.com/joho/godotenv"
)

var DB *sql.DB

func ConnectDb() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error occured during loading .env")
	}
	host := os.Getenv("HOST")
	port, _ := strconv.Atoi(os.Getenv("PORT")) // don't forget to convert int since port is int type.
	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DBNAME")
	pass := os.Getenv("PASSWORD")

	psqlSetup := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable", host, port, user, dbname, pass)
	fmt.Println(psqlSetup)
	db, err := sql.Open("postgres", psqlSetup)
	if err != nil {
		slog.Error("couldn't connect to the datbase")
		log.Fatal("database failure", err)
	} else {
		DB = db
		fmt.Println("Successfully connected to database!")
		runMigrations(db)
	}
}

func runMigrations(db *sql.DB) {
	files, err := filepath.Glob("database/migrations/*.sql")
	if err != nil {
		slog.Warn("Failed to locate migration files", "error", err)
		return
	}
	sort.Strings(files)

	for _, file := range files {
		if !strings.HasSuffix(file, ".up.sql") {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			slog.Warn("Failed to read migration file", "file", file, "error", err)
			continue
		}

		_, err = db.Exec(string(content))
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			slog.Error("Migration execution failed", "file", file, "error", err)
			log.Fatal("Fatal migration error", err)
		}
		fmt.Printf("Migration successfully applied: %s\n", file)
	}
}
