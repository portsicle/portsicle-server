package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func HandleSocket(w http.ResponseWriter, r *http.Request) {
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

		// response from client
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
