package webrtc

import (
	"fmt"
	"github.com/pion/webrtc/v4"
	"sync"
)

type Peer struct {
	mu            sync.RWMutex
	id            string
	manager       *WebRTCManager
	connection    *webrtc.PeerConnection
	configuration *webrtc.Configuration
}

type WebRTCManager struct {
	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
	videoCodec *webrtc.RTPCodecType
	audioCodec *webrtc.RTPCodecType
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
