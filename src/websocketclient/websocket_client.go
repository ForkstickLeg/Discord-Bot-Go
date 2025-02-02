package websocketclient

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ChopstickLeg/Discord-Bot-Practice/src/structs"
	"github.com/gorilla/websocket"
)

type WebsocketClient struct {
	URL               string
	Connection        *websocket.Conn
	ReconnectURL      string
	HeartbeatInterval int
	listenChan        chan interface{}
	SequenceNum       int
	SessionId         string
	token             string
	mutex             sync.Mutex
	reconnecting      bool
	retryCount        int
	maxRetries        int
	reconnectDelay    time.Duration
	maxDelay          time.Duration
}

func NewWebsocketClient(url string) *WebsocketClient {
	return &WebsocketClient{
		URL:            url + "/?v=10&encoding=json",
		reconnecting:   false,
		retryCount:     0,
		maxRetries:     5,
		reconnectDelay: 1 * time.Second,
		maxDelay:       30 * time.Second,
		token:          os.Getenv("BOT_TOKEN"),
	}
}

func (ws *WebsocketClient) Connect(url string) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println("Error connecting: ", err)
	}

	ws.Connection = conn
	ws.retryCount = 0
	fmt.Println("Websocket Connected")

	ws.Connection.SetCloseHandler(func(code int, text string) error {
		return nil
	})

	_, msg, err := ws.Connection.ReadMessage()
	if err != nil {
		fmt.Println("Error reading Hello message")
	}

	var message structs.Message
	err = json.Unmarshal(msg, &message)
	if err != nil {
		fmt.Println("Error unmarshalling Hello message")
	}

	if message.Op != 10 {
		fmt.Println("Expected opcode 10, got opcode ", message.Op)
		return
	}

	fmt.Println("Received opcode 10")

	rawMsgData, err := json.Marshal(message.D)
	if err != nil {
		fmt.Println("Error marshalling raw message data")
	}

	var helloMessage structs.HelloMessageData
	err = json.Unmarshal(rawMsgData, &helloMessage)
	if err != nil {
		fmt.Println("Error unmarshalling raw message data")
	}

	ws.HeartbeatInterval = helloMessage.HeartbeatInterval

	ws.SequenceNum = *message.S
	if ws.SessionId == "" {
		sendIdentifyMessage := structs.Message{
			Op: 2,
			D: structs.IdentifyMessageData{
				Token: ws.token,
				Properties: structs.Props{
					Os:      runtime.GOOS,
					Browser: "CSL's Discord App",
					Device:  "CSL's Discord App",
				},
				Intents: intents,
			},
		}
		sendMessageJSON, err := json.Marshal(sendIdentifyMessage)
		if err != nil {
			fmt.Println("Error marshalling Identify message: ", err)
			return
		}
		err = ws.SendMessage(sendMessageJSON)
		if err != nil {
			fmt.Println("Error sending Identify message: ", err)
			return
		}
		fmt.Println("Identify message sent")
	} else {
		sendResumeMessage := structs.Message{
			Op: 6,
			D: map[string]interface{}{
				"token":      "Bot " + ws.token,
				"session_id": ws.SessionId,
				"seq":        ws.SequenceNum,
			},
		}
		sendMessageJSON, err := json.Marshal(sendResumeMessage)
		if err != nil {
			fmt.Println("Error marshalling Resume message: ", err)
			return
		}
		err = ws.SendMessage(sendMessageJSON)
		if err != nil {
			fmt.Println("Error sending Resume message: ", err)
			return
		}
		fmt.Println("Resume message sent")
	}

	_, msg, err = ws.Connection.ReadMessage()
	if err != nil {
		fmt.Println("Error reading Ready/Resumed message")
	}

	err = json.Unmarshal(msg, &message)
	if err != nil {
		fmt.Println("Error Unmarshalling ready/resumed message")
	}

	rawMsgData, err = json.Marshal(message.D)
	if err != nil {
		fmt.Println("Error marshalling ready/resumed message data")
	}

	ws.SequenceNum = *message.S

	if *message.T == "READY" {
		var readyMessageData structs.ReadyPayload
		err = json.Unmarshal(rawMsgData, &readyMessageData)
		if err != nil {
			fmt.Println("Error unmarshalling ready payload")
		}
		ws.SessionId = readyMessageData.SessionId
		ws.ReconnectURL = readyMessageData.ResumeGatewayURL
		fmt.Println("Ready message received")
	} else if *message.T == "Resumed" {
		fmt.Println("Resumed")
	} else {
		fmt.Println("Unknown message received")
		fmt.Println(message)
	}

	go ws.heartbeat()
	go ws.ReadMessage()
}

func (ws *WebsocketClient) heartbeat() {
	fmt.Println("Started heartbeat ticker")
	ticker := time.NewTicker(time.Duration(ws.HeartbeatInterval) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		select {
		default:
			ws.mutex.Lock()
			sendMessage := structs.Message{
				Op: 1,
				D:  &ws.SequenceNum,
			}
			sendMessageJSON, err := json.Marshal(sendMessage)
			if err != nil {
				fmt.Println("Error marshalling message: ", err)
				return
			}
			ws.SendMessage(sendMessageJSON)
		case <-ws.listenChan:
			return
		}
	}
}

func (ws *WebsocketClient) ReadMessage() {
	fmt.Println("Reading")
	ws.mutex.Lock()
	conn := ws.Connection
	ws.mutex.Unlock()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			sameConn := conn == ws.Connection
			if sameConn {
				fmt.Println("Error reading message, attempting reconnect")
				ws.Connection.Close()
				ws.AttemptReconnect()
			}
		}
		select {
		case <-ws.listenChan:
			return
		default:
			ws.HandleMessage(message)
		}
	}
}

func (ws *WebsocketClient) SendMessage(message []byte) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if ws.Connection == nil {
		return websocket.ErrCloseSent
	}

	err := ws.Connection.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		fmt.Println("Error sending message: ", err)
		ws.AttemptReconnect()
	}
	return err
}

func (ws *WebsocketClient) AttemptReconnect() {
	if ws.reconnecting {
		fmt.Println("Reconnect in progress. Skipping...")
		return
	}

	ws.reconnecting = true
	defer func() { ws.reconnecting = false }()

	ws.Connection.SetReadDeadline(time.Now())
	time.Sleep(5000 * time.Millisecond)

	fmt.Printf("Active goroutines: %d\n", runtime.NumGoroutine())

	ws.mutex.Lock()
	ws.Connection = nil
	ws.mutex.Unlock()

	for ws.retryCount < ws.maxRetries {
		delay := time.Duration((ws.reconnectDelay) * (1 << ws.retryCount))
		if delay > ws.maxDelay {
			delay = ws.maxDelay
		}
		fmt.Printf("Reconnecting in %v\n", delay)
		time.Sleep(delay)

		ws.retryCount++
		ws.Connect(ws.ReconnectURL)
		if ws.Connection != nil {
			fmt.Println("Reconnect successful")
			return
		}
	}

	fmt.Println("Max retries reached, giving up")
	ws.SessionId = ""
	ws.SequenceNum = 0
	ws.retryCount = 0
	ws.Connect(ws.URL)
}

func (ws *WebsocketClient) Close() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if ws.Connection != nil {
		ws.Connection.SetReadDeadline(time.Now())
		ws.Connection.Close()
		ws.Connection = nil
		fmt.Println("Websocket connection closed")
	}
}
