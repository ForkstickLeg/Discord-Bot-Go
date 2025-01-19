package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

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
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        int       `json:"type,omitempty"`
	Options     []Command `json:"options,omitempty"`
	Required    bool      `json:"required,omitempty"`
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
var writeMutex sync.Mutex

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	clientid = os.Getenv("APP_ID")
	botToken = os.Getenv("BOT_TOKEN")

	silenceOptions := Command{
		Name:        "user",
		Description: "Silence someone, including in voice and text for the specified amount of time (in minutes)",
		Type:        1,
		Options: []Command{
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

	gatewayURL = getWSUrl()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		fmt.Println("Goroutine setup conn started")
		defer fmt.Println("Goroutine setup conn ended")
		conn, err := setupWebSocket(ctx, gatewayURL)
		if err != nil {
			fmt.Println("Error setting up connection")
			cancel()
			return
		}
		defer conn.Close(websocket.StatusGoingAway, "ending connection")
		readMessages(ctx, cancel, conn)
	}()
	waitForShutdown(cancel)

	fmt.Println("shutdown complete")
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

func setupWebSocket(ctx context.Context, websocketURL string) (*websocket.Conn, error) {
	if websocketURL == "" {
		return nil, fmt.Errorf("websocket URL is empty")
	} else {
		websocketURL = websocketURL + "?v=10&encoding=json"
	}

	conn, _, err := websocket.Dial(ctx, websocketURL, nil)
	if err != nil {
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
		err = safeWrite(conn, websocket.MessageText, msgJSON)
		if err != nil {
			fmt.Println("Error sending message:", err)
		}
	}

	return conn, nil
}

func readMessages(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	for {
		if !conn || conn.readyState != websocket.OPEN {
			console.error
		}
		//need to find a way to stop this loop on error
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled")
			return
		default:
			_, message, err := conn.Read(context.Background())
			if err != nil {
				fmt.Println("Error reading message:", err)
				conn.Close(websocket.StatusNormalClosure, "reconnect")
				reconnect(cancel)
				return
			}
			handleGatewayMessage(ctx, cancel, conn, string(message))
		}
	}
}

func reconnect(cancel context.CancelFunc) {
	cancel()
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		conn, err := setupWebSocket(ctx, resumeGatewayUrl)
		if err == nil {
			fmt.Println("Reconnected to WebSocket")
			go readMessages(ctx, cancel, conn)
			break
		}
		fmt.Println("Error reconnecting to WebSocket:", err)
		time.Sleep(5 * time.Second)
	}
}

func handleGatewayMessage(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, message string) {
	var msg Message
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		fmt.Println("Error unmarshalling message: ", err)
		fmt.Println("Message: ", message)
		return
	}
	select {
	case <-ctx.Done():
		fmt.Println("Context closed")
		return

	default:
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
				err = safeWrite(conn, websocket.MessageText, sendMessageJSON)
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
			go func() {
				fmt.Println("Goroutine heartbeat message case 10 started")
				defer fmt.Println("Goroutine heartbeat message case 10 ended")
				time.Sleep(time.Duration(randomSleep) * time.Millisecond)
				err = safeWrite(conn, websocket.MessageText, sendMessageJSON)
				if err != nil {
					fmt.Println("Error sending message: ", err)
					return
				}
				fmt.Println("Heartbeat message sent")
			}()
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
			err = safeWrite(conn, websocket.MessageText, sendMessageJSON)
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
			go func() {
				fmt.Println("Goroutine heartbeat message case 11 started")
				defer fmt.Println("Goroutine heartbeat message case 11 ended")
				time.Sleep(time.Duration(heartbeatInterval) * time.Millisecond)
				err = safeWrite(conn, websocket.MessageText, sendMessageJSON)
				if err != nil {
					fmt.Println("Error sending message: ", err)
					return
				}
				fmt.Println("Heartbeat message sent")
			}()
		case 7:
			conn.Close(websocket.StatusNormalClosure, "reconnect")
			reconnect(cancel)
		case 0:
			//This is where the payload will come from
			sequenceNum = *msg.S
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
			case "GUILD_CREATE":
				fmt.Println("Joined guild")
			default:
				fmt.Println("unknown message received\n" + message)
			}
		case 9:
			if msg.D == "true" {
				conn.Close(websocket.StatusNormalClosure, "reconnect")
				reconnect(cancel)
			} else {
				fmt.Println("Invalid session error")
				conn.Close(websocket.StatusNormalClosure, "reconnect")
				setupWebSocket(ctx, gatewayURL)
			}
		}
	}
}

func safeWrite(conn *websocket.Conn, messageType websocket.MessageType, data []byte) error {
	writeMutex.Lock()
	defer writeMutex.Unlock()
	return conn.Write(context.Background(), messageType, data)
}

func cleanupAndExit(response *http.Response) {
	if response != nil {
		response.Body.Close()
	}
	os.Exit(1)
}

func setupCommand(name string, description string, options *Command, commandType ...int) Command {
	var data Command
	key := []string{"Content-Type", "Authorization"}
	value := []string{"application/json", "Bot " + botToken}

	if len(commandType) > 0 && options == nil {
		data = Command{
			Name:        name,
			Description: description,
			Type:        commandType[1],
		}
	} else if options != nil && len(commandType) > 0 {
		data = Command{
			Name:        name,
			Description: description,
			Type:        commandType[1],
			Options:     []Command{*options},
		}
	} else if options == nil && len(commandType) == 0 {
		data = Command{
			Name:        name,
			Description: description,
		}
	} else {
		data = Command{
			Name:        name,
			Description: description,
			Options:     []Command{*options},
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
	err = json.Unmarshal(rawDataBytes, &interaction)
	if err != nil {
		fmt.Println("Error unmarshalling message: ", err)
		fmt.Println("Message: ", message)
		return
	}
	var data InteractionData
	rawData, ok = interaction.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error asserting baseMsg.D to map[string]interface{}")
		return
	}
	rawDataBytes, err = json.Marshal(rawData)
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
	switch data.Name {
	case "silence":
		silence(1, 1)
	}
}
func silence(memberId int, minutes int) {
	fmt.Println("User silenced", memberId, minutes)
}

func waitForShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Received shutdown signal")
	cancel() // Cancel the context
}
