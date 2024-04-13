package main

import (
	"fmt"
	"net/http"
)

func main() {
	server := http.NewServeMux()
	hub := NewHub()
	go hub.Run()
	server.HandleFunc("/ws/{roomId}", hub.serveWs)
	if err := http.ListenAndServe(":8081", server); err != nil {
		fmt.Println("Could not start server", err)
	}
}
