package chat

type Room struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Clients map[string]*Client `json:"clients"`
}

type Hub struct {
	Rooms      map[string]*Room
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *Message, 5),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Rooms:      make(map[string]*Room),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if r, ok := h.Rooms[client.RoomID]; ok {
				if _, ok := r.Clients[client.ID]; ok {
					r.Clients[client.ID] = client
				}
			}
		case client := <-h.unregister:
			if r, ok := h.Rooms[client.RoomID]; ok {
				if _, ok := r.Clients[client.ID]; ok {
					if r.Clients != nil {
						h.broadcast <- &Message{
							Content:  string(client.Username) + "left chat",
							Username: client.Username,
						}
					}
					delete(r.Clients, client.ID)
					close(client.Message)
				}
			}
		case message := <-h.broadcast:
			if _, ok := h.Rooms[message.RoomID]; ok {
				go func(message *Message) {
					for _, client := range h.Rooms[message.RoomID].Clients {
						select {
						case client.Message <- message:
						default:
							close(client.Message)
						}
					}
				}(message)
			}
		}
	}
}
