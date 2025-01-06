package server

import (
	"log"
	"net/http"
)

func Health(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Write([]byte("Instance is Healthy!!!"))
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		log.Print("Non GET method received on /health.")
	}
}
