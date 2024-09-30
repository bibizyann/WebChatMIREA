package chat

import "sync"

type Chat struct {
	mu      sync.RWMutex
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Clients map[string]*Client `json:"clients"`
}

type Hub struct {
	mu         sync.RWMutex
	Chats      map[string]*Chat
	Broadcast  chan *Message
	Register   chan *Client
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan *Message, 5),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Chats:      make(map[string]*Chat),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if r, ok := h.Chats[client.ChatID]; ok {
				r.mu.Lock()
				if _, ok := r.Clients[client.ID]; !ok {
					r.Clients[client.ID] = client
				}
				r.mu.Unlock()
			}
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if r, ok := h.Chats[client.ChatID]; ok {
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
			if r, ok := h.Chats[message.ChatID]; ok {
				go func(message *Message, chat *Chat) {
					chat.mu.RLock()
					defer chat.mu.RUnlock()
					for _, client := range chat.Clients {
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
