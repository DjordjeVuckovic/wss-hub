package main

import (
	"github.com/gorilla/websocket"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
)

func upgradeToWs() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO Not Allow connections from any origin
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

type Room struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Clients map[string]*Client `json:"clients"`
}

type Hub struct {
	Rooms      map[string]*Room
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message, 5),
	}
}

func (h *Hub) Run() {
	r := Room{
		ID:      "123",
		Name:    "room 123",
		Clients: make(map[string]*Client),
	}
	h.Rooms[r.ID] = &r
	for {
		select {
		case cl := <-h.Register:
			slog.Info("Registering client", "client", cl.ID, "room", cl.RoomID)
			if _, ok := h.Rooms[cl.RoomID]; ok {
				r := h.Rooms[cl.RoomID]

				if _, ok := r.Clients[cl.ID]; !ok {
					r.Clients[cl.ID] = cl
				}
			}
		case cl := <-h.Unregister:
			if _, ok := h.Rooms[cl.RoomID]; ok {
				if _, ok := h.Rooms[cl.RoomID].Clients[cl.ID]; ok {
					if len(h.Rooms[cl.RoomID].Clients) != 0 {
						h.Broadcast <- &Message{
							Content:  "user left the chat",
							RoomID:   cl.RoomID,
							Username: cl.Username,
						}
					}

					delete(h.Rooms[cl.RoomID].Clients, cl.ID)
					close(cl.Message)
				}
			}

		case m := <-h.Broadcast:
			slog.Info("Broadcasting message", "message", m)
			if _, ok := h.Rooms[m.RoomID]; ok {
				slog.Info("Broadcasting message to room", "room", m.RoomID, "message", m.Content)
				for _, cl := range h.Rooms[m.RoomID].Clients {
					slog.Info("Broadcasting message to client", "client", cl.ID, "message", m.Content)
					cl.Message <- m
				}
			}
		}
	}
}

func (h *Hub) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgradeToWs().Upgrade(w, r, nil)
	var roomId = r.PathValue("roomId")
	if err != nil {
		slog.Error("Error upgrading connection", "err", err)
		return
	}
	id := strconv.Itoa(rand.Int())
	cl := &Client{
		Conn:     conn,
		Message:  make(chan *Message, 10),
		ID:       id,
		RoomID:   roomId,
		Username: id,
	}

	m := &Message{
		Content:  "A new user has joined the room",
		RoomID:   id,
		Username: id,
	}
	h.Register <- cl
	h.Broadcast <- m

	go cl.readMessage(h)
	go cl.writeMessage()
}
