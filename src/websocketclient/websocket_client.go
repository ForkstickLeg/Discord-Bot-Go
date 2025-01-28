package websocketclient

import (
	"context"
	"fmt"
	"os"
	"runtime"
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
	ctx               context.Context
	cancel            context.CancelFunc
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

	if ws.cancel != nil {
		ws.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	ws.ctx = ctx
	ws.cancel = cancel

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
		select {
		case <-ws.ctx.Done():
			fmt.Println("Stopping read message loop")
			return
		default:
			ws.mutex.Lock()
			conn := ws.Connection
			ws.mutex.Unlock()
			if conn == nil {
				fmt.Println("Websocket is nil, stopping read")
				return
			} else {
				_, message, err := conn.ReadMessage()
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
						fmt.Printf("Error reading WebSocket message: %v\n", err)
					}
					break
				}
				ws.HandleMessage(message)
			}

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

	if ws.cancel != nil {
		ws.cancel()
		ws.Connection.SetReadDeadline(time.Now())
		time.Sleep(5000 * time.Millisecond)
	}

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

	if ws.cancel != nil {
		ws.cancel()
	}

	if ws.Connection != nil {
		ws.Connection.SetReadDeadline(time.Now())
		ws.Connection.Close()
		ws.Connection = nil
		fmt.Println("Websocket connection closed")
	}
}
