package chat

import (
	"WebChatMIREA/pkg/database"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
)

type HubHandler struct {
	hub *Hub
}

func NewHandler(h *Hub) *HubHandler {
	return &HubHandler{
		hub: h,
	}
}

type CreateChatReq struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IsGroup bool   `json:"group"`
}

func (h *HubHandler) CreateChat(c *gin.Context) {
	var req CreateChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.hub.mu.Lock()
	h.hub.Chats[req.ID] = &Chat{
		ID:      req.ID,
		Name:    req.Name,
		Clients: make(map[string]*Client),
	}
	h.hub.mu.Unlock()

	chat := database.Chats{Name: req.Name, CreatedAt: time.Now(), IsGroup: req.IsGroup}
	if err := database.DB.Create(&chat).Error; err != nil {
		log.Println("error creating chat:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create room"})
		return
	}

	c.JSON(http.StatusOK, req)
}

func (h *HubHandler) JoinChat(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	chatID := c.Param("chatId")
	clientID := c.Query("userId")
	username := c.Query("username")

	client := &Client{
		Conn:     conn,
		Message:  make(chan *Message, 10),
		ID:       clientID,
		ChatID:   chatID,
		Username: username,
	}

	message := &Message{
		Content:      "A new user has joined the room",
		ChatID:       chatID,
		Username:     username,
		CreationTime: time.Now().String(),
	}

	UserId, err := strconv.Atoi(clientID)
	ChatId, err := strconv.Atoi(chatID)
	user := database.ChatMembers{ChatID: ChatId, UserID: UserId, Name: username, JoinedAt: time.Now()}
	result := database.DB.Create(&user)
	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create user"})
		return
	}
	h.hub.Register <- client
	h.hub.Broadcast <- message

	go client.WriteMessage()
	client.ReadMessage(h.hub)
}

type ClientRes struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (h *HubHandler) GetClients(c *gin.Context) {
	var clients []ClientRes
	chatId := c.Param("chatId")

	h.hub.mu.RLock()
	if chat, ok := h.hub.Chats[chatId]; !ok {
		clients = make([]ClientRes, 0)
		c.JSON(http.StatusOK, clients)
	} else {
		chat.mu.RLock()
		for _, c := range chat.Clients {
			clients = append(clients, ClientRes{
				ID:       c.ID,
				Username: c.Username,
			})
		}
		chat.mu.RUnlock()

		c.JSON(http.StatusOK, clients)
	}
	h.hub.mu.RUnlock()
}
