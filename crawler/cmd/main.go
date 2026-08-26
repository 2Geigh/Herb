package main

import (
	"log"

	"github.com/2Geigh/Herb/crawler/internal/database"
)

func main() {
	err := database.InitializeDB()
	if err != nil {
		log.Fatalf("connect to database failed: %v", err)
	}
}
