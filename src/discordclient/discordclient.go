package discordclient

import (
	"fmt"
	"os"

	"github.com/ChopstickLeg/Discord-Bot-Practice/src/database"
	"github.com/bwmarrin/discordgo"
)

func SetupDiscord() {
	//clientid := os.Getenv("APP_ID")
	botToken := os.Getenv("BOT_TOKEN")

	discord, err := discordgo.New("Bot " + botToken)
	if err != nil {
		fmt.Println("Error creating discordgo object")
	}

	registerCommands(discord)

	discord.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMembers | discordgo.IntentGuildMessages | discordgo.IntentMessageContent

	discord.AddHandler(ready)

	discord.AddHandler(messageCreate)

}

func registerCommands(s *discordgo.Session) {

}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateCustomStatus("Straight jorkin it")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	db := database.GetDB()
	db.DeleteOldSilences()
	db.IsUserSilenced(m.Author.ID)
}
