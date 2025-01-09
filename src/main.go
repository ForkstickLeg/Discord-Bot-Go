package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type TokenResponse struct {
	Token string `json:"access_token"`
	URL   string `json:"url"`
}

type Command struct {
	ID string `json:"id"`
}

var oauthToken string
var wslUrl string

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	clientid := os.Getenv("APP_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	botToken := os.Getenv("BOT_TOKEN")

	key := []string{"Content-Type"}
	value := []string{"application/x-www-form-urlencoded"}

	oauthToken = getToken(clientid, clientSecret, key, value)

	wslUrl = getWSUrl(key, value)

	fmt.Println(wslUrl + "\n" + oauthToken)

	key = append(key, "Authorization")
	value = append(value, "Bot "+botToken)

	commands := makeCall("https://discord.com/api/v10/applications/"+clientid+"/commands", "GET", key, value)

	fmt.Println("Raw response:", string(commands))

	var output []Command
	err = json.Unmarshal(commands, &output)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}

	for _, out := range output {
		fmt.Printf("ID: %s\n", out.ID)
	}
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

func getToken(clientid string, clientSecret string, key []string, value []string) string {

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "identify connections bot")
	data.Set("client_id", clientid)
	data.Set("client_secret", clientSecret)

	body := makeCall("https://discord.com/api/v10/oauth2/token", "POST", key, value, data.Encode())

	var tokenResponse TokenResponse
	err := json.Unmarshal(body, &tokenResponse)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}

	return tokenResponse.Token
}

func getWSUrl(key []string, value []string) string {
	body := makeCall("https://discord.com/api/v10/gateway", "GET", key, value)

	var tokenResponse TokenResponse
	err := json.Unmarshal(body, &tokenResponse)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}
	return tokenResponse.URL
}
