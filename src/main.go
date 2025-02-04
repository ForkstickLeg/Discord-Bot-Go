package main

import (
	"fmt"

	"github.com/ChopstickLeg/Discord-Bot-Practice/src/database"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/discordclient"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	db := database.CreateDB("discordbot")
	defer db.Close()

	discordclient.SetupDiscord()
}
