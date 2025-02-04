package main

import (
	"fmt"
	"os"

	"github.com/ChopstickLeg/Discord-Bot-Practice/src/database"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/discordclient"
	"github.com/joho/godotenv"
)

func main() {
	if isRunningLocally() {
		err := godotenv.Load("../.env")
		if err != nil {
			fmt.Println("Error loading env files")
		}
		fmt.Println("Running locally, .env loaded.")
	}

	db := database.CreateDB(os.Getenv("DB_PATH"))
	defer db.Close()

	discordclient.SetupDiscord()
}

func isRunningLocally() bool {
	env := os.Getenv("APP_ENV")
	return env == "" || env == "local"
}
