package discordclient

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/ChopstickLeg/Discord-Bot-Practice/database"
	"github.com/ChopstickLeg/Discord-Bot-Practice/silence"
	"github.com/bwmarrin/discordgo"
)

func SetupDiscord() {
	clientid := os.Getenv("APP_ID")
	botToken := os.Getenv("BOT_TOKEN")

	discord, err := discordgo.New("Bot " + botToken)
	if err != nil {
		fmt.Println("Error creating discordgo object")
	}

	var commands = []*discordgo.ApplicationCommand{
		{
			Name:        "silence",
			Description: "Silence someone, including in voice and text for the specified amount of time (in minutes)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "username",
					Description: "User to silence",
					Type:        6,
					Required:    true,
				},
				{
					Name:        "duration",
					Description: "Length of silence (in minutes)",
					Type:        4,
					Required:    true,
				},
			},
		},
		{
			Name:        "source",
			Description: "Provides a link to my source code on GitHub",
		},
	}

	discord.ApplicationCommandBulkOverwrite(clientid, "", commands)

	discord.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMembers | discordgo.IntentGuildMessages | discordgo.IntentMessageContent

	discord.AddHandler(ready)

	discord.AddHandler(messageCreate)

	discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		if data.Name == "silence" {
			mute(data.Options[0].UserValue(s).ID, int(data.Options[1].IntValue()), i.GuildID, s, i)
		}
	})

	err = discord.Open()
	if err != nil {
		fmt.Println("Error opening session")
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = discord.Close()
	if err != nil {
		fmt.Println("Error closing session")
	}
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateCustomStatus("Straight jorkin it")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Println("Message received, checking db")
	db := database.GetDB()
	db.DeleteOldSilences()
	silenced := db.IsUserSilenced(m.Author.ID)
	if silenced {
		s.ChannelMessageDelete(m.ChannelID, m.Reference().MessageID)
	}
}

func mute(memberId string, minutes int, guildId string, s *discordgo.Session, i *discordgo.InteractionCreate) {
	user, err := s.User(memberId)
	if err != nil {
		fmt.Println("Error getting User object")
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: user.Mention() + " has been silenced for " + strconv.Itoa(minutes) + " minutes",
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{
					"users",
				},
			},
		},
	})
	db := database.GetDB()
	db.InsertSilence(memberId, guildId, minutes)
	if memberId == "1326247335692341318" {
		fmt.Println("Error, cannot mute bot")
		return
	}
	sil := silence.NewSilence(memberId, minutes, guildId)
	go sil.SilenceUser()
	fmt.Println("User silenced", memberId, minutes)
}
