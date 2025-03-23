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

	db := database.CreateDB()
	defer db.Close()

	discordclient.SetupDiscord()
}

func isRunningLocally() bool {
	env := os.Getenv("APP_ENV")
	fmt.Println("Environment: ", env)
	return env == "" || env == "local"
}
