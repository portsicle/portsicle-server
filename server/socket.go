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
	clientConn := &clientConnection{conn: conn}

	mu.Lock()
	clients[sessionId] = clientConn
	responses[sessionId] = make(chan *Message)
	mu.Unlock()

	log.Printf("New client connection established with session ID: %s", sessionId)

	// the very first message client will receive from server is the sessionId
	err = clientConn.writeMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s", sessionId)))
	if err != nil {
		log.Println("Error sending session ID:", err)
		return
	}

	for {

		/*
			Infinitely read from this client connection untill error occurs.
			on error: close the responses channel, delete everything related to this session and close the connection.
			on receiving a response: Decode the json & write the raw response into the responses channel so the handleGET function can read from it for this channel.
		*/

		_, msg, err := conn.ReadMessage()
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

		// decoding response from client
		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// if there's a non-nil message, write it into the responses channel
		if message.Response != nil {
			mu.Lock()
			if ch, exists := responses[sessionId]; exists {
				ch <- &message
			}
			mu.Unlock()
		}

	}
}
