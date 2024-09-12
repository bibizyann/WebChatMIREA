package chat

import (
	"bytes"
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

var _ = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
} //I don`t remained ideas to use it as it should be

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
	Hub  *Hub
}

func (c *Client) readPump() {
	defer func() {
		err := c.Conn.Close()
		if err != nil {
			return
		}
		c.Hub.unregister <- c
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
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, []byte{'\n'}, []byte{' '}, -1))
		c.Hub.broadcast <- message
	}
}

func (c *Client) writePump() {
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
		case message, ok := <-c.Send:
			err := c.Conn.SetWriteDeadline(time.Now().Add(writeTime))
			if err != nil {
				return
			}
			if !ok {
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, err = w.Write(message)
			if err != nil {
				return
			}

			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, err = w.Write([]byte{'\n'})
				if err != nil {
					return
				}
				_, err = w.Write(<-c.Send)
				if err != nil {
					return
				}
			}

			if err := w.Close(); err != nil {
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

func PeerChatConn(c *websocket.Conn, hub *Hub) {
	//Just a draft function
	client := &Client{Hub: hub, Conn: c, Send: make(chan []byte, 256)}
	client.Hub.register <- client

	go client.writePump()
	client.readPump()
}
