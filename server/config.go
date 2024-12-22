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

var (
	clients   = make(map[string]*websocket.Conn)
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
