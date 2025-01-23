package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ChopstickLeg/Discord-Bot-Practice/src/database"
	discordapiclient "github.com/ChopstickLeg/Discord-Bot-Practice/src/discord-api-client"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/structs"
	"github.com/ChopstickLeg/Discord-Bot-Practice/src/websocketclient"
	"github.com/joho/godotenv"
)

var clientid string
var botToken string

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	db := database.CreateDB("discordbot")
	defer db.Close()

	clientid = os.Getenv("APP_ID")
	botToken = os.Getenv("BOT_TOKEN")

	silenceOptions := structs.Command{
		Name:        "user",
		Description: "Silence someone, including in voice and text for the specified amount of time (in minutes)",
		Type:        1,
		Options: []structs.Command{
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
	}

	output := setupCommand("silence", "Use this command to totally silence someone. Specify the amount of time (in minutes), default is 1", &silenceOptions)

	fmt.Println(output.ID)

	gatewayURL := getWSUrl()

	client := websocketclient.NewWebsocketClient(gatewayURL)

	client.Connect(gatewayURL)

	select {}
}

func setupCommand(name string, description string, options *structs.Command, commandType ...int) structs.Command {
	var data structs.Command
	key := []string{"Content-Type", "Authorization"}
	value := []string{"application/json", "Bot " + botToken}

	if len(commandType) > 0 && options == nil {
		data = structs.Command{
			Name:        name,
			Description: description,
			Type:        commandType[1],
		}
	} else if options != nil && len(commandType) > 0 {
		data = structs.Command{
			Name:        name,
			Description: description,
			Type:        commandType[1],
			Options:     []structs.Command{*options},
		}
	} else if options == nil && len(commandType) == 0 {
		data = structs.Command{
			Name:        name,
			Description: description,
		}
	} else {
		data = structs.Command{
			Name:        name,
			Description: description,
			Options:     []structs.Command{*options},
		}
	}
	ac := discordapiclient.NewApiCall("https://discord.com/api/v10/applications/" + clientid + "/commands")
	ac.AddHeader(key, value)
	ac.AddBody(data)

	commands := ac.MakePostCall()

	var output structs.Command
	err := json.Unmarshal(commands, &output)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}
	return output
}

func getWSUrl() string {
	key := []string{"Content-Type", "Authorization"}
	value := []string{"application/json", "Bot " + botToken}

	ac := discordapiclient.NewApiCall("https://discord.com/api/v10/gateway")
	ac.AddHeader(key, value)

	body := ac.MakeGetCall()

	var output structs.GatewayResponse
	err := json.Unmarshal(body, &output)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}
	return output.URL
}
