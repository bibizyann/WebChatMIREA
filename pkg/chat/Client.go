package chat

import (
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	maxMessageSize = 512
	writeTime      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 60 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	/*
		CheckOrigin: func(r *http.Request) bool {
			return true
			},
		allow connections from any origin — but, in production, this should be restricted for security reasons
	*/
} //I don`t remained ideas to use it as it should be

type Client struct {
	Conn     *websocket.Conn
	ID       string `json:"id"`
	RoomID   string `json:"roomID"`
	Username string `json:"username"`
	Message  chan *Message
}

type Message struct {
	Content      string `json:"content"`
	Username     string `json:"username"`
	CreationTime string `json:"creationTime"`
	RoomID       string `json:"roomID"`
}

func (c *Client) readMessage(hub *Hub) {
	defer func() {
		err := c.Conn.Close()
		if err != nil {
			return
		}
		hub.unregister <- c
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	err := c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		return
	}
	c.Conn.SetPongHandler(func(string) error {
		err := c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}
		return nil
	})

	for {
		_, m, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message := &Message{
			Content:  string(m),
			Username: c.Username,
		}
		hub.broadcast <- message
	}
}

func (c *Client) writeMessage() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		err := c.Conn.Close()
		if err != nil {
			return
		}
	}()
	for {
		select {
		case message, ok := <-c.Message:
			err := c.Conn.SetWriteDeadline(time.Now().Add(writeTime))
			if err != nil {
				return
			}
			if !ok {
				return
			}

			err = c.Conn.WriteJSON(message)
			if err != nil {
				return
			}

		case <-ticker.C:
			err := c.Conn.SetWriteDeadline(time.Now().Add(writeTime))
			if err != nil {
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
