package websocketclient

import (
	"encoding/json"
	"fmt"
	"os"
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
		fmt.Printf("Websocket closed with code: %d %s", code, text)
		ws.AttemptReconnect()
		return nil
	})

	go ws.ReadMessage()
}

func (ws *WebsocketClient) ReadMessage() {
	fmt.Println("Reading")
	for {
		_, message, err := ws.Connection.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				fmt.Printf("WebSocket closed. Code: %d, Reason: %s\n", closeErr.Code, closeErr.Text)

				switch closeErr.Code {
				case websocket.CloseNormalClosure:
					fmt.Println("Normal closure")
					ws.AttemptReconnect()
				case websocket.CloseAbnormalClosure:
					fmt.Println("Abnormal closure")
					ws.AttemptReconnect()
				case 4009:
					fmt.Println("Session timeout. You need to reconnect.")
					ws.Close()
					ws.Connect(ws.URL)
				default:
					fmt.Printf("Unhandled close code: %d\n", closeErr.Code)
				}
			} else {
				fmt.Printf("Error reading WebSocket message: %v", err)
			}
			break
		}
		ws.HandleMessage(message)
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

	ws.Close()

	message := structs.Message{
		Op: 6,
		D: map[string]interface{}{
			"token":      "Bot " + ws.token,
			"session_id": ws.SessionId,
			"seq":        ws.SequenceNum,
		},
	}

	sendMessageJSON, err := json.Marshal(message)
	fmt.Println(string(sendMessageJSON))
	if err != nil {
		fmt.Println("Error marshalling Resume message")
		return
	}

	for ws.retryCount < ws.maxRetries {
		delay := time.Duration((ws.reconnectDelay) * (1 << ws.retryCount))
		if delay > ws.maxDelay {
			delay = ws.maxDelay
		}
		fmt.Printf("Reconnecting in %v", delay)
		time.Sleep(delay)

		ws.retryCount++
		ws.Connect(ws.ReconnectURL)
		if ws.Connection != nil {
			fmt.Println("Reconnect sucessful")
			ws.SendMessage(sendMessageJSON)
			return
		}
	}
	fmt.Println("Max retries reached, giving up")
}

func (ws *WebsocketClient) Close() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if ws.Connection != nil {
		ws.Connection.Close()
		ws.Connection = nil
		fmt.Println("Websocket connection closed")
	}
}
