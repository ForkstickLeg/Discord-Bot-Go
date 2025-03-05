package main

import (
	"fmt"
	"os"

	"github.com/ChopstickLeg/Discord-Bot-Practice/database"
	"github.com/ChopstickLeg/Discord-Bot-Practice/discordclient"
	"github.com/joho/godotenv"
)

func main() {
	if isRunningLocally() {
		err := godotenv.Load("./.env")
		if err != nil {
			fmt.Println("Error loading env files: ", err)
			return
		}
		fmt.Println("Running locally, .env loaded.")
	}

	fmt.Println(os.Getenv("APP_ENV"))
	db := database.CreateDB()
	defer db.Close()

	discordclient.SetupDiscord()
}

func isRunningLocally() bool {
	env := os.Getenv("APP_ENV")
	return env == "" || env == "local"
}
