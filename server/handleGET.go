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

			/*
				this if block will handle the case when the sessionId would be in referer header instead of path.
				like if the browser wants to fetch a resource like a css or a script file.
				eg: if the requested url by browser is : http://localhost:8081/assets/index-t7wc3-JJ.js, then:

				referer Header: http://localhost:8081/6a205f60-8221-4c36-a63b-d71768b0216c  -> this header will contain our sessionId.
				referer path:  	6a205f60-8221-4c36-a63b-d71768b0216c  -> fetched from Referer Header.
				session id:  		6a205f60-8221-4c36-a63b-d71768b0216c  -> session id is set to referer path.
				request path:  	/assets/index-t7wc3-JJ.js            	-> request path is now the resource we want to request!
			*/

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

		/*
			this if block will handle the case when the sessionId will be available in the path itself.
			like the browser is requesting just the base HTML file or something served on base url '/'
			eg: if the requested url by browser is : http://localhost:8081/6a205f60-8221-4c36-a63b-d71768b0216c, then:

			path: 0fcafcb0-630b-40ab-afa3-adfbb544c7a5 			 -> path contains the sessionId itself
			referer Header: "" 												 			 -> no referer header
			session id: 0fcafcb0-630b-40ab-afa3-adfbb544c7a5 -> session id directly from the path
			request path: /																	 -> request path is now the base path!
		*/

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

	body, _ := io.ReadAll(r.Body)
	message := Message{
		Method:  r.Method,
		Path:    requestPath,
		Headers: r.Header,
		Body:    string(body),
	}

	/*
		a normal message will look like:
		{
			Method:  GET
			Path: 	 /
			Headers: map[
				Accept:[text/html,application/xhtml+xml,application/xml;q=0.9;q=0.8]
				Accept-Encoding:[gzip, deflate, br, zstd]
				Accept-Language:[en-US,en;q=0.5]
				Connection:[keep-alive]
				User-Agent:[Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0]
				]
			Body:    <nil>
			}
	*/

	// json encoding of the raw message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		http.Error(w, "Error crafting message", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	conn, exists := clients[sessionId]
	responseChan, responseExists := responses[sessionId]
	mu.Unlock()

	if !exists || !responseExists {
		http.Error(w, "Invalid session ID", http.StatusNotFound)
		return
	}

	// send this request message to the client
	err = conn.WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		http.Error(w, "Error sending message to WebSocket", http.StatusInternalServerError)
		log.Printf("Error sending message to session %s: %v", sessionId, err)
		return
	}

	/*
		After routing this GET message to client via socket, we will now read from the responses channel for this session.
		We expect that the client will now interact with the local server and reply with a response.
		So we	wait for response message from client untill the channel is closed.
	*/

	select {
	case response, ok := <-responseChan:
		if !ok {
			http.Error(w, "Connection closed", http.StatusServiceUnavailable)
			return
		}

		/*
			A respose will look like:
			{
				Method:  GET
				Path:    /assets/index-DAWgUG2K.css
				Headers: map[
					Accept:[text/css]
					Accept-Encoding:[gzip, deflate, br, zstd]
					Connection:[keep-alive]
					Referer:[http://localhost:8081/114f1771d-426f-9f4c-c14958ac] -> A referer header which indicates that its a response to resource request
					User-Agent:[Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0]
				]
				Body:   0xc00011e060
			}
		*/

		if response == nil || response.Response == nil {
			http.Error(w, "Invalid response received", http.StatusInternalServerError)
			return
		}

		for key, values := range response.Response.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(response.Response.StatusCode) // Write Status code
		fmt.Fprint(w, response.Response.Body)       // Write body -> this is the actual response received for a request

	case <-r.Context().Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return
	}
}
