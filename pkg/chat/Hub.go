package chat

import "sync"

type Room struct {
	mu      sync.RWMutex
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Clients map[string]*Client `json:"clients"`
}

type Hub struct {
	mu         sync.RWMutex
	Rooms      map[string]*Room
	Broadcast  chan *Message
	Register   chan *Client
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan *Message, 5),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Rooms:      make(map[string]*Room),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if r, ok := h.Rooms[client.RoomID]; ok {
				r.mu.Lock()
				if _, ok := r.Clients[client.ID]; !ok {
					r.Clients[client.ID] = client
				}
				r.mu.Unlock()
			}
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if r, ok := h.Rooms[client.RoomID]; ok {
				r.mu.Lock()
				if _, ok := r.Clients[client.ID]; ok {
					if r.Clients != nil {
						h.Broadcast <- &Message{
							Content:  string(client.Username) + "left chat",
							Username: client.Username,
						}
					}
					delete(r.Clients, client.ID)
					close(client.Message)
				}
				r.mu.Unlock()
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			if r, ok := h.Rooms[message.RoomID]; ok {
				go func(message *Message, room *Room) {
					room.mu.RLock()
					defer room.mu.RUnlock()
					for _, client := range room.Clients {
						select {
						case client.Message <- message:
						default:
							close(client.Message)
						}
					}
				}(message, r)
			}
		}
	}
}
