package websocketclient

import (
	"fmt"
	"os"
	"sync"
	"time"

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
			fmt.Println("Error reading message: ", err)
			ws.AttemptReconnect()
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
