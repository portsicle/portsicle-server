package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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

var (
	clients   = make(map[string]*websocket.Conn)
	responses = make(map[string]chan *Message)
	mu        sync.Mutex
)

func handleSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to ws: %v", err)
		return
	}

	sessionId := uuid.New().String()

	mu.Lock()
	clients[sessionId] = conn
	responses[sessionId] = make(chan *Message)
	mu.Unlock()

	log.Printf("New WebSocket connection established with session ID: %s", sessionId)

	err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Session Id: %s", sessionId)))
	if err != nil {
		log.Println("Error sending session ID:", err)
		return
	}

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("Client closed connection for session ID: %s", sessionId)
			} else {
				log.Printf("Error reading message for session ID %s: %v", sessionId, err)
			}
			mu.Lock()
			delete(clients, sessionId)
			close(responses[sessionId])
			delete(responses, sessionId)
			mu.Unlock()
			conn.Close()
			break
		}

		// Handle response from client
		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		if message.Response != nil {
			mu.Lock()
			if ch, exists := responses[sessionId]; exists {
				ch <- &message
			}
			mu.Unlock()
		}

		log.Printf("Message type: %d from session %s: %s", messageType, sessionId, msg)
	}
}

func handleGET(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Path[len("/"):]

	mu.Lock()
	conn, exists := clients[sessionId]
	responseChan, responseExists := responses[sessionId]
	mu.Unlock()

	if !exists || !responseExists {
		http.Error(w, "Invalid session ID", http.StatusNotFound)
		return
	}

	body, _ := io.ReadAll(r.Body)
	message := Message{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: r.Header,
		Body:    string(body),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		http.Error(w, "Error crafting message", http.StatusInternalServerError)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		http.Error(w, "Error sending message to WebSocket", http.StatusInternalServerError)
		log.Printf("Error sending message to session %s: %v", sessionId, err)
		return
	}

	// Wait for response msg from client
	select {
	case response := <-responseChan:
		// Copy response headers
		for key, values := range response.Response.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		// Set status code
		w.WriteHeader(response.Response.StatusCode)
		// Write body
		fmt.Fprint(w, response.Response.Body)
	case <-r.Context().Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return
	}
}

func main() {
	http.HandleFunc("/ws", handleSocket)
	http.HandleFunc("/", handleGET)
	log.Println("Starting server on: ws://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
