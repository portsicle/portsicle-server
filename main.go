package main

import (
	"log"
	"net/http"

	"github.com/portsicle/portsicle-server/server"
)

func main() {
	http.HandleFunc("/ws", server.HandleSocket)
	http.HandleFunc("/", server.HandleGET)
	http.HandleFunc("/health", server.Health)
	log.Println("Starting server on port :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
