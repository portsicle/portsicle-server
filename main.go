package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/portsicle/portsicle-server/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}
	http.HandleFunc("/ws", server.HandleSocket)
	http.HandleFunc("/", server.HandleGET)
	http.HandleFunc("/health", server.Health)
	log.Println("Starting server on port ", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
