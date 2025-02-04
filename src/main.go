package main

import (
	"log"
	"os"

	"github.com/ChopstickLeg/Discord-Bot-Practice/src/database"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/discordclient"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	App_id := os.Getenv("APP_ID")
	log.Println(App_id)

	db := database.CreateDB("discordbot")
	defer db.Close()

	discordclient.SetupDiscord()
}
