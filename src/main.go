package main

import (
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/database"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/discordclient"
)

func main() {
	db := database.CreateDB("discordbot")
	defer db.Close()

	discordclient.SetupDiscord()
}
