package server

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type clientConnection struct {
	// custom client connection with mutex to avoid concurrent writes to same connection

	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *clientConnection) writeMessage(messageType int, data []byte) error {
	/*
		Gorilla WebSocket docs states that ws connections support only one concurrent reader and writer.
		In our case, browser will initiate multiple parallel resource requests to fetch other static files.
		As this all happens in same client connection, this lock will prevent race condition caused by parallel browser resource requests.
	*/

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(messageType, data)
}

var (
	clients   = make(map[string]*clientConnection)
	responses = make(map[string]chan *Message)
	mu        sync.Mutex
)

type Message struct {
	Method   string      `json:"method"`
	Path     string      `json:"path"`
	Headers  http.Header `json:"headers"`
	Body     string      `json:"body"`
	Response *Response   `json:"response,omitempty"`
}

type Response struct {
	StatusCode int         `json:"statusCode"`
	Headers    http.Header `json:"headers"`
	Body       string      `json:"body"`
}
