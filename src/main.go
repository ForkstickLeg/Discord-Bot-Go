package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

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

func makeCall(apiUrl string, method string, key []string, value []string, body ...string) []byte {
	var requestBody string
	if len(body) > 0 {
		requestBody = body[0]
	} else {
		requestBody = "{}"
	}
	req, err := http.NewRequest(method, apiUrl, strings.NewReader(requestBody))
	if err != nil {
		fmt.Println("Error creating request")
		os.Exit(1)
	}

	for i := 0; i < len(key); i++ {
		req.Header.Add(key[i], value[i])
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		cleanupAndExit(response)
	}
	defer response.Body.Close()

	output, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	return output
}

func cleanupAndExit(response *http.Response) {
	if response != nil {
		response.Body.Close()
	}
	os.Exit(1)
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

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		os.Exit(1)
	}

	commands := makeCall("https://discord.com/api/v10/applications/"+clientid+"/commands", "POST", key, value, string(jsonData))

	var output structs.Command
	err = json.Unmarshal(commands, &output)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}
	return output
}

func getWSUrl() string {
	key := []string{"Content-Type", "Authorization"}
	value := []string{"application/json", "Bot " + botToken}

	body := makeCall("https://discord.com/api/v10/gateway", "GET", key, value)

	var output structs.GatewayResponse
	err := json.Unmarshal(body, &output)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}
	return output.URL
}
