package webrtc

import "github.com/pion/webrtc/v4"

var Config = webrtc.Configuration{
	ICETransportPolicy: webrtc.ICETransportPolicyRelay,
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
		{
			URLs:       []string{"turn:turn.localhost:3478"},
			Username:   "username",
			Credential: "password",
		},
	},
	SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
}
