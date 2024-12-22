package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/ws", handleSocket)
	http.HandleFunc("/", handleGET)
	log.Println("Starting server on port :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
