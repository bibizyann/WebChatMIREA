package webrtc

import (
	"WebChatMIREA/pkg/chat"
	"crypto/sha256"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"log"
	"net/http"
	"os"
	"sync"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func PeerChatConn(c *gin.Context, h *chat.Hub) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	client := &chat.Client{
		Conn:     conn,
		ID:       c.Query("userId"),
		ChatID:   c.Param("uuid"),
		Username: c.Query("username"),
	}
	h.Register <- client

	go client.WriteMessage()
	client.ReadMessage(h)
}

func RoomConn(c *gin.Context, p *Peers) {
	unsafeconn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	conn := &ThreadSafeWriter{
		sync.Mutex{},
		unsafeconn,
	}

	defer conn.conn.Close()

	peerConnection, err := webrtc.NewPeerConnection(Config)
	if err != nil {
		log.Print(err)
		return
	}
	defer peerConnection.Close()

	// Accept one audio and one video track incoming
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return
		}
	}

	newPeer := PeerConnectionState{
		PeerConnection: peerConnection,
		Websocket:      conn,
	}

	p.mu.Lock()
	p.Connections = append(p.Connections, newPeer)
	p.mu.Unlock()

	log.Println(p.Connections)

	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}

		if writeErr := newPeer.Websocket.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			log.Println(writeErr)
		}
	})

	peerConnection.OnConnectionStateChange(func(pp webrtc.PeerConnectionState) {
		switch pp {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			p.SignalPeerConnections()
		default:
			log.Print("unhandled default case")
		}
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// Create a track to fan out our incoming video to all peers
		trackLocal := p.AddTrack(t)
		if trackLocal == nil {
			return
		}
		defer p.RemoveTrack(trackLocal)

		buf := make([]byte, 1500)
		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	})

	p.SignalPeerConnections()
	message := &websocketMessage{}
	for {
		_, raw, err := conn.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Println(err)
				return
			}

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Println(err)
				return
			}

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func RoomCreate(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/room/%s/websocket", uuid.New().String()))
}

func (s *StorageHandler) RoomRender(c *gin.Context) {
	id := c.Param("uuid")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create room"})
	}
	_ = "ws"
	if os.Getenv("ENVIRONMENT") == "PRODUCTION" {
		_ = "wss"
	}

	_, _, _ = s.CreateOrGetRoom(id)

	//such as Jinja))
	c.HTML(200, "index", gin.H{})
}

func (s *StorageHandler) CreateOrGetRoom(uuid string) (string, string, *Room) {
	s.storage.mu.Lock()
	defer s.storage.mu.Unlock()

	h := sha256.New()
	h.Write([]byte(uuid))
	suuid := fmt.Sprintf("%x", h.Sum(nil))

	if room := s.storage.Rooms[uuid]; room != nil {
		if _, ok := s.storage.Streams[suuid]; !ok {
			s.storage.Streams[suuid] = room
		}
		return uuid, suuid, room
	}

	hub := chat.NewHub()
	p := &Peers{}
	p.TrackLocals = make(map[string]*webrtc.TrackLocalStaticRTP)
	room := &Room{
		Peers: p,
		Hub:   hub,
	}
	s.storage.Rooms[uuid] = room
	s.storage.Streams[suuid] = room
	go hub.Run()
	return uuid, suuid, room
}

func (s *StorageHandler) RoomHandler(c *gin.Context) {
	uuid := c.Param("uuid")
	if uuid == "" {
		return
	}

	_, _, room := s.CreateOrGetRoom(uuid)
	RoomConn(c, room.Peers)
}

func (s *StorageHandler) RoomChatWebsocket(c *gin.Context) {
	uuid := c.Param("uuid")
	if uuid == "" {
		return
	}

	s.storage.mu.Lock()
	room := s.storage.Rooms[uuid]
	s.storage.mu.Unlock()
	if room == nil {
		return
	}
	if room.Hub == nil {
		return
	}
	PeerChatConn(c, room.Hub)
}

//TODO: fix: some problems with handlers; add: some basic html's
