package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/joho/godotenv"
)

type GatewayResponse struct {
	URL string `json:"url"`
}

type Message struct {
	Op int         `json:"op"`
	D  interface{} `json:"d"`
	S  *int        `json:"s,omitempty"`
	T  *string     `json:"t,omitempty"`
}

type HelloMessageData struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type IdentifyMessageData struct {
	Token      string `json:"token"`
	Properties Props  `json:"properties"`
	Intents    int    `json:"intents"`
}

type Props struct {
	Os      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type ReadyPayload struct {
	SessionId        string `json:"session_id"`
	ResumeGatewayURL string `json:"resume_gateway_url"`
}

type Command struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Interaction struct {
	Data interface{} `json:"data"`
}

type InteractionData struct {
	Name string `json:"name"`
}

var botToken string
var clientid string
var gatewayURL string
var heartbeatInterval int
var intents int = 1<<0 | 1<<1 | 1<<9 | 1<<15
var resumeGatewayUrl string
var sessionId string
var sequenceNum int

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	clientid = os.Getenv("APP_ID")
	botToken = os.Getenv("BOT_TOKEN")

	output := setupCommand("silence", "Use this command to totally silence someone. Specify the amount of time (in minutes), default is 1")

	gatewayURL = getWSUrl()

	go func() {
		conn, err := setupWebSocket(gatewayURL)
		if err != nil {
			fmt.Println("Error setting up connection")
			return
		}
		readMessages(conn)
	}()
	fmt.Printf("ID: %s\nName: %s\nDescription:%s", output.ID, output.Name, output.Description)

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

func setupWebSocket(websocketURL string) (*websocket.Conn, error) {
	if websocketURL == "" {
		return nil, fmt.Errorf("websocket URL is empty")
	} else {
		websocketURL = websocketURL + "?v=10&encoding=json"
	}

	const maxRetries = 3
	const retryDelay = 3 * time.Second

	var conn *websocket.Conn
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, _, err = websocket.Dial(context.Background(), websocketURL, nil)
		if err == nil {
			fmt.Println("Connected to WebSocket")
			break
		}
		fmt.Printf("Error connecting to WebSocket (attempt %d/%d): %v\n", i+1, maxRetries, err)
		time.Sleep(retryDelay)
	}

	if err != nil {
		fmt.Println("Failed to connect to WebSocket after multiple attempts")
		return nil, err
	}

	if len(sessionId) != 0 {
		msg := Message{
			Op: 6,
			D: map[string]interface{}{
				"token":      botToken,
				"session_id": sessionId,
				"seq":        sequenceNum,
			},
		}
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshalling data")
		}
		fmt.Println(string(msgJSON))
		err = conn.Write(context.Background(), websocket.MessageText, msgJSON)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}

	return conn, nil
}

func readMessages(conn *websocket.Conn) {
	for {
		_, message, err := conn.Read(context.Background())
		if err != nil {
			fmt.Println("Error reading message:", err)
			conn.Close(websocket.StatusNormalClosure, "reconnect")
			reconnect()
			return
		}
		handleGatewayMessage(conn, string(message))
	}
}

func reconnect() {
	var conn *websocket.Conn
	var err error
	for {
		conn, err = setupWebSocket(resumeGatewayUrl)
		if err == nil {
			fmt.Println("Reconnected to WebSocket")
			readMessages(conn)
			break
		}
		fmt.Println("Error reconnecting to WebSocket:", err)
		time.Sleep(5 * time.Second) // Wait before retrying
	}
}

func handleGatewayMessage(conn *websocket.Conn, message string) {
	var msg Message
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		fmt.Println("Error unmarshalling message: ", err)
		fmt.Println("Message: ", message)
		return
	}
	switch msg.Op {
	case 10:
		//Hello message
		var data HelloMessageData
		rawData, ok := msg.D.(map[string]interface{})
		if !ok {
			fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
			return
		}
		rawDataBytes, err := json.Marshal(rawData)
		if err != nil {
			fmt.Println("Error marshalling rawData to bytes: ", err)
			return
		}
		err = json.Unmarshal(rawDataBytes, &data)
		if err != nil {
			fmt.Println("Error unmarshalling message: ", err)
			fmt.Println("Message: ", message)
			return
		}
		if len(sessionId) == 0 {

			sendIdentifyMessage := Message{
				Op: 2,
				D: IdentifyMessageData{
					Token: botToken,
					Properties: Props{
						Os:      runtime.GOOS,
						Browser: "CSL's Discord App",
						Device:  "CSL's Discord App",
					},
					Intents: intents,
				},
			}
			sendMessageJSON, err := json.Marshal(sendIdentifyMessage)
			if err != nil {
				fmt.Println("Error marshalling message: ", err)
				return
			}
			err = conn.Write(context.Background(), websocket.MessageText, sendMessageJSON)
			if err != nil {
				fmt.Println("Error sending message: ", err)
				return
			}
			fmt.Println("Identify message sent")

		}
		heartbeatInterval = data.HeartbeatInterval
		randomSleep := rand.Intn(heartbeatInterval)
		sendMessage := Message{
			Op: 1,
			D:  &sequenceNum,
		}
		sendMessageJSON, err := json.Marshal(sendMessage)
		if err != nil {
			fmt.Println("Error marshalling message: ", err)
			return
		}
		time.Sleep(time.Duration(randomSleep) * time.Millisecond)
		err = conn.Write(context.Background(), websocket.MessageText, sendMessageJSON)
		if err != nil {
			fmt.Println("Error sending message: ", err)
			return
		}
		fmt.Println("Heartbeat message sent")
	case 1:
		//Heartbeat message, requires immediate heartbeat return
		sendMessage := Message{
			Op: 1,
		}
		sendMessageJSON, err := json.Marshal(sendMessage)
		if err != nil {
			fmt.Println("Error marshalling message: ", err)
			return
		}
		err = conn.Write(context.Background(), websocket.MessageText, sendMessageJSON)
		if err != nil {
			fmt.Println("Error sending message: ", err)
			return
		}
	case 11:
		fmt.Println("Heartbeat ACK received")
		sendMessage := Message{
			Op: 1,
			D:  &sequenceNum,
		}
		sendMessageJSON, err := json.Marshal(sendMessage)
		if err != nil {
			fmt.Println("Error marshalling message: ", err)
			return
		}
		time.Sleep(time.Duration(heartbeatInterval) * time.Millisecond)
		err = conn.Write(context.Background(), websocket.MessageText, sendMessageJSON)
		if err != nil {
			fmt.Println("Error sending message: ", err)
			return
		}
		fmt.Println("Heartbeat message sent")
	case 7:
		conn.Close(websocket.StatusNormalClosure, "reconnect")
		setupWebSocket(resumeGatewayUrl)
	case 0:
		//This is where the payload will come from
		switch *msg.T {
		case "READY":
			var data ReadyPayload
			rawData, ok := msg.D.(map[string]interface{})
			if !ok {
				fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
				return
			}
			rawDataBytes, err := json.Marshal(rawData)
			if err != nil {
				fmt.Println("Error marshalling rawData to bytes: ", err)
				return
			}
			err = json.Unmarshal(rawDataBytes, &data)
			if err != nil {
				fmt.Println("Error unmarshalling message: ", err)
				fmt.Println("Message: ", message)
				return
			}
			sequenceNum = *msg.S
			sessionId = data.SessionId
			resumeGatewayUrl = data.ResumeGatewayURL
		case "INTERACTION_CREATE":
			handleInteraction(msg)
		case "RESUMED":
			fmt.Println("Session Resumed")
		default:
			fmt.Println("unknown message received\n" + message)
		}
	case 9:
		if msg.D == "true" {
			conn.Close(websocket.StatusNormalClosure, "reconnect")
			setupWebSocket(resumeGatewayUrl)
		} else {
			fmt.Println("Invalid session error")
			conn.Close(websocket.StatusNormalClosure, "reconnect")
			setupWebSocket(gatewayURL)
		}
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

func handleInteraction(message Message) {
	var interaction Interaction
	parseDataTo(&interaction, message)
	var data InteractionData
	rawData, ok := interaction.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
		return
	}
	rawDataBytes, err := json.Marshal(rawData)
	if err != nil {
		fmt.Println("Error marshalling rawData to bytes: ", err)
		return
	}
	err = json.Unmarshal(rawDataBytes, &data)
	if err != nil {
		fmt.Println("Error unmarshalling message: ", err)
		fmt.Println("Message: ", message)
		return
	}
	switch InteractionData.Name {
		case "silence":

	}
}

func parseDataTo(returnedObject interface{}, message Message) {
	rawData, ok := message.D.(map[string]interface{})
	if !ok {
		fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
		return
	}
	rawDataBytes, err := json.Marshal(rawData)
	if err != nil {
		fmt.Println("Error marshalling rawData to bytes: ", err)
		return
	}
	err = json.Unmarshal(rawDataBytes, returnedObject)
	if err != nil {
		fmt.Println("Error unmarshalling message: ", err)
		fmt.Println("Message: ", message)
		return
	}
}

func silence(memberId int, minutes int) {

}
