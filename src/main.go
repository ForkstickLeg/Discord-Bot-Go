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
	Scope string `json:"scope"`
	URL   string `json:"url"`
}

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	clientid := os.Getenv("APP_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	apiUrl := "https://discord.com/api/v10"

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "identify connections bot")
	data.Set("client_id", clientid)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequest("POST", apiUrl+"/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}

	req, err = http.NewRequest("GET", apiUrl+"/gateway", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client = &http.Client{}
	response, err = client.Do(req)
	if err != nil {
		fmt.Println("Error getting gateway")
		return
	}
	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}

	fmt.Println(tokenResponse.URL)
}
