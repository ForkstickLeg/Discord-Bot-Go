package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/net/websocket"
)

type Interaction struct {
	Type int `json:"type"`
}

type GatewayResponse struct {
	URL string `json:"url"`
}

type Command struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var botToken string
var clientid string
var publicKey string
var gatewayURL string

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	clientid = os.Getenv("APP_ID")
	botToken = os.Getenv("BOT_TOKEN")
	publicKey = os.Getenv("PUBLIC_KEY")

	output := setupCommand("silence", "Use this command to totally silence someone. Specify the amount of time (in minutes), default is 1")

	gatewayURL = getWSUrl()

	fmt.Printf("ID: %s\nName: %s\nDescription:%s", output.ID, output.Name, output.Description)

	http.HandleFunc("/ack", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Request acknowledged")
	})

	http.HandleFunc("/", postHandler)

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("HTTP server error", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Println("Received signal:", sig)
	fmt.Println("Shutting down gracefully")
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

func setupWebSocket(websocketURL string, userId int) {
	conn, err := websocket.Dial(websocketURL, "", "http://localhost/")
	if err != nil {
		fmt.Println("Error connecting to WebSocket:", err)
		return
	}
	defer conn.Close()

	for {
		var message string
		err := websocket.Message.Receive(conn, &message)
		if err != nil {
			fmt.Println("Error reading message:", err)
			return
		}
		fmt.Printf("Received message: %s\n", message)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		signature := r.Header.Get("X-Signature-Ed25519")
		timestamp := r.Header.Get("X-Signature-Timestamp")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		message := append([]byte(timestamp), body...)
		sig, err := hex.DecodeString(signature)
		if err != nil {
			http.Error(w, "Invalid signature format", http.StatusBadRequest)
			return
		}

		pubKey, err := hex.DecodeString(publicKey)
		if err != nil {
			http.Error(w, "Invalid public key format", http.StatusInternalServerError)
			return
		}

		if !ed25519.Verify(pubKey, message, sig) {
			http.Error(w, "Invalid request signature", http.StatusUnauthorized)
			return
		}

		var interaction Interaction
		if err := json.Unmarshal(body, &interaction); err != nil {
			http.Error(w, "Error unmarshalling request body", http.StatusBadRequest)
			return
		}

		if interaction.Type == 1 {
			response := map[string]int{"type": 1}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		if interaction.Type == 2 {
			response := map[string]interface{}{"type": 4, "data": map[string]string{"content": "User has been silenced"}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
			//TODO: actually call the function to handle muting the person
		}

		response := map[string]int{"type": 1}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func cleanupAndExit(response *http.Response) {
	if response != nil {
		response.Body.Close()
	}
	os.Exit(1)
}

func setupCommand(name string, description string, commandType ...int) Command {
	var data map[string]string
	key := []string{"Content-Type", "Authorization"}
	value := []string{"application/json", "Bot " + botToken}

	if len(commandType) > 0 {
		data = map[string]string{
			"name":        name,
			"description": description,
			"type":        string(rune(commandType[1])),
		}
	} else {
		data = map[string]string{
			"name":        name,
			"description": description,
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		os.Exit(1)
	}

	commands := makeCall("https://discord.com/api/v10/applications/"+clientid+"/commands", "POST", key, value, string(jsonData))

	var output Command
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

	var output GatewayResponse
	err := json.Unmarshal(body, &output)
	if err != nil {
		fmt.Println("Error unmarshalling response")
	}
	return output.URL
}

func silence(memberId int, minutes int) {

}
