package server

import (
	"encoding/base64"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shamaton/msgpack/v2"
)

func HandleSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to ws: %v", err)
		return
	}

	sessionId := uuid.New().String()
	b64SessionId := hashSessionId(sessionId, 12)
	clientConn := &clientConnection{conn: conn}

	mu.Lock()
	clients[b64SessionId] = clientConn
	responses[b64SessionId] = make(chan *Message)
	mu.Unlock()

	log.Printf("New client connection established with session ID: %s", b64SessionId)

	// the very first message client will receive from server is the sessionId
	err = clientConn.writeMessage(websocket.TextMessage, []byte(b64SessionId))
	if err != nil {
		log.Println("Error sending session ID:", err)
		return
	}

	for {

		/*
			Infinitely read from this client connection untill error occurs.
			on error: close the responses channel, delete everything related to this session and close the connection.
			on receiving a response: Decode the msgpack & write the raw response into the responses channel so the handleGET function can read from it for this channel.
		*/

		_, msg, err := conn.ReadMessage()
		log.Print("received compressed body: ", len(msg), " bytes")
		msg = lz4decompress(msg)
		log.Print("after uncompression: ", len(msg), " bytes")
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("Client closed connection for session ID: %s", b64SessionId)
			} else {
				log.Printf("Error reading message for session ID %s: %v", b64SessionId, err)
			}
			mu.Lock()
			delete(clients, b64SessionId)
			close(responses[b64SessionId])
			delete(responses, b64SessionId)
			mu.Unlock()
			conn.Close()
			break
		}

		// decoding response from client
		var message Message
		if err := msgpack.Unmarshal(msg, &message); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// if there's a non-nil message, write it into the responses channel
		if message.Response != nil {
			mu.Lock()
			if ch, exists := responses[b64SessionId]; exists {
				ch <- &message
			}
			mu.Unlock()
		}

	}
}

// hashSessionId tends to generate a collision proof Base64 encoded uuid which is shorter in length.
func hashSessionId(sessionId string, part int) string {
	u, err := uuid.Parse(sessionId)
	if err != nil {
		log.Print("error parsing uuid: ", err)
	}

	if part <= 0 {
		log.Print("Invalid part encountered, falling back to long uuid.")
		return sessionId
	}

	// URL safe base 64 encoding of last all the 16 bytes
	b64SessionId := base64.URLEncoding.EncodeToString([]byte(u[:]))
	b64 := b64SessionId[:part]

	// adding hyphen at  part/2 length.
	mid := part / 2
	if mid > 0 {
		b64 = b64[:mid] + "-" + b64[mid:]
	}

	// this returned string is a shorter version which would look like: VucWqW-mjT2yJ
	return b64
}
