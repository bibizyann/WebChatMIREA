package chat

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type HubHandler struct {
	hub *Hub
}

func NewHandler(h *Hub) *HubHandler {
	return &HubHandler{
		hub: h,
	}
}

type CreateRoomReq struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *HubHandler) CreateRoom(c *gin.Context) {
	var req CreateRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.hub.mu.Lock()
	h.hub.Rooms[req.ID] = &Room{
		ID:      req.ID,
		Name:    req.Name,
		Clients: make(map[string]*Client),
	}
	h.hub.mu.Unlock()
	c.JSON(http.StatusOK, req)
}

func (h *HubHandler) JoinRoom(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	roomID := c.Param("roomId")
	clientID := c.Query("userId")
	username := c.Query("username")

	client := &Client{
		Conn:     conn,
		Message:  make(chan *Message, 10),
		ID:       clientID,
		RoomID:   roomID,
		Username: username,
	}

	message := &Message{
		Content:  "A new user has joined the room",
		RoomID:   roomID,
		Username: username,
	}

	h.hub.Register <- client
	h.hub.Broadcast <- message

	go client.writeMessage()
	client.readMessage(h.hub)
}

type ClientRes struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (h *HubHandler) GetClients(c *gin.Context) {
	var clients []ClientRes
	roomId := c.Param("roomId")

	h.hub.mu.RLock()
	if room, ok := h.hub.Rooms[roomId]; !ok {
		clients = make([]ClientRes, 0)
		c.JSON(http.StatusOK, clients)
	} else {
		room.mu.RLock()
		for _, c := range room.Clients {
			clients = append(clients, ClientRes{
				ID:       c.ID,
				Username: c.Username,
			})
		}
		room.mu.RUnlock()

		c.JSON(http.StatusOK, clients)
	}
	h.hub.mu.RUnlock()
}
