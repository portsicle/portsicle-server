package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func HandleGET(w http.ResponseWriter, r *http.Request) {
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

	// Waiting for response msg from client
	select {
	case response, ok := <-responseChan:
		if !ok {
			http.Error(w, "Connection closed", http.StatusServiceUnavailable)
			return
		}

		if response == nil || response.Response == nil {
			http.Error(w, "Invalid response received", http.StatusInternalServerError)
			return
		}

		for key, values := range response.Response.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(response.Response.StatusCode) // Write header
		fmt.Fprint(w, response.Response.Body)       // Write body
	case <-r.Context().Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return
	}
}
