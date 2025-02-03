package discordclient

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

func SetupDiscord() {
	//clientid := os.Getenv("APP_ID")
	botToken := os.Getenv("BOT_TOKEN")

	discord, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Println("Error creating discordgo object")
	}

	discord.Identify.Intents = discordgo.IntentGuilds
	discord.Identify.Intents |= discordgo.IntentGuildMembers
	discord.Identify.Intents |= discordgo.IntentGuildMessages
	discord.Identify.Intents |= discordgo.IntentMessageContent

}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateCustomStatus("Straight jorkin it")
}

func messageCreate()
