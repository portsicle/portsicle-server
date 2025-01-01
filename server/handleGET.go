package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

func HandleGET(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/"):]
	var sessionId string
	var requestPath string

	/*
		While requesting other static files except html document,
		the broswer will use / as base path instead of /<session-id>
		eg: /style.css instead of /<session-id>/style.css

		So to handle linked static files, we need to check the Referer header to validate the session id
	*/

	// try to get session from referer
	referer := r.Header.Get("Referer")

	if referer != "" {
		if refererUrl, err := url.Parse(referer); err == nil {

			refererPath := refererUrl.Path[1:]
			refererParts := strings.SplitN(refererPath, "/", 2)

			if len(refererParts) > 0 {

				canBeSessionId := refererParts[0]
				mu.Lock()
				_, exists := clients[canBeSessionId]
				mu.Unlock()

				if exists {
					sessionId = canBeSessionId
					requestPath = "/" + path
				}

			}
		}
	}

	// If no session-id in referer, fallback to path
	if sessionId == "" {
		if !strings.Contains(path, "/") {
			sessionId = path
			requestPath = "/"
		} else {
			parts := strings.SplitN(path, "/", 2)
			canBeSessionId := parts[0]

			mu.Lock()
			_, exists := clients[canBeSessionId]
			mu.Unlock()

			if exists {
				sessionId = canBeSessionId
				requestPath = "/" + parts[1]
			}
		}
	}

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
		Path:    requestPath,
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
