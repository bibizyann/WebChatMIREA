package webrtc

import (
	"WebChatMIREA/pkg/chat"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"log"
	"sync"
	"time"
)

type WebRTCManager struct {
	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
	videoCodec *webrtc.RTPCodecType
	audioCodec *webrtc.RTPCodecType
}

type Storage struct {
	Rooms   map[string]*Room
	Streams map[string]*Room
	mu      sync.RWMutex
}

func NewStorage() *Storage {
	return &Storage{
		Rooms:   make(map[string]*Room),
		Streams: make(map[string]*Room),
	}
}

type StorageHandler struct {
	storage *Storage
}

func NewStorageHandler(s *Storage) *StorageHandler {
	return &StorageHandler{
		storage: s,
	}
}

type Room struct {
	Peers *Peers
	ID    string `json:"id"`
	Hub   *chat.Hub
}

type Peers struct {
	mu          sync.RWMutex
	Connections []PeerConnectionState
	TrackLocals map[string]*webrtc.TrackLocalStaticRTP
}

type PeerConnectionState struct {
	PeerConnection *webrtc.PeerConnection
	Websocket      *ThreadSafeWriter
}

type ThreadSafeWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func (t *ThreadSafeWriter) WriteJSON(v interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn.WriteJSON(v)
}

// DispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
func (p *Peers) DispatchKeyFrame() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := range p.Connections {
		for _, receiver := range p.Connections[i].PeerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = p.Connections[i].PeerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

// SignalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks
func (p *Peers) SignalPeerConnections() {
	p.mu.Lock()
	defer func() {
		p.mu.Unlock()
		p.DispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range p.Connections {
			if p.Connections[i].PeerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				p.Connections = append(p.Connections[:i], p.Connections[i+1:]...)
				log.Println("a", p.Connections)
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are sending, so we don't double send
			existingSenders := map[string]bool{}
			for _, sender := range p.Connections[i].PeerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to an existing track remove and signal
				if _, ok := p.TrackLocals[sender.Track().ID()]; !ok {
					if err := p.Connections[i].PeerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range p.Connections[i].PeerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range p.TrackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := p.Connections[i].PeerConnection.AddTrack(p.TrackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := p.Connections[i].PeerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = p.Connections[i].PeerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}

			if err = p.Connections[i].Websocket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				p.SignalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

// AddTrack add to list of tracks and fire renegotation for all PeerConnections
func (p *Peers) AddTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	p.mu.Lock()
	defer func() {
		p.mu.Unlock()
		p.SignalPeerConnections()
	}()

	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	p.TrackLocals[t.ID()] = trackLocal
	return trackLocal
}

func (p *Peers) RemoveTrack(t *webrtc.TrackLocalStaticRTP) {
	p.mu.Lock()
	defer func() {
		p.mu.Unlock()
		p.SignalPeerConnections()
	}()

	delete(p.TrackLocals, t.ID())
}

func (m *WebRTCManager) createTrack(codecName string) (*webrtc.TrackLocalStaticRTP, *webrtc.RTPCodecCapability, error) {
	var codec webrtc.RTPCodecCapability

	switch codecName {
	case "VP8":
		codec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}
	case "VP9":
		codec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP9, ClockRate: 90000}
	case "H264":
		codec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}
	case "Opus":
		codec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000}
	case "G722":
		codec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeG722, ClockRate: 8000}
	case "PCMU":
		codec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000}
	case "PCMA":
		codec = webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA, ClockRate: 8000}
	default:
		return nil, nil, fmt.Errorf("unknown codec %s", codecName)
	}

	// Создание нового трека с использованием новой версии API
	track, err := webrtc.NewTrackLocalStaticRTP(codec, "stream", "stream")
	if err != nil {
		return nil, nil, err
	}

	return track, &codec, nil
}
