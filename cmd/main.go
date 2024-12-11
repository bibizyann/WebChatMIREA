package main

import (
	"WebChatMIREA/pkg/chat"
	"WebChatMIREA/pkg/database"
	"WebChatMIREA/pkg/database/handlers"
	"WebChatMIREA/pkg/middleware"
	"WebChatMIREA/pkg/webrtc"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func main() {
	database.Init()
	router := gin.Default()
	router.Use(middleware.CORSMiddleware())
	hub := chat.NewHub()
	wsHandler := chat.NewHandler(hub)
	storage := webrtc.NewStorage()
	storageHandler := webrtc.NewStorageHandler(storage)
	go hub.Run()

	router.POST("/signup", handlers.SignUp)
	router.POST("/login", handlers.Login)
	router.POST("/ws/createChat", wsHandler.CreateChat)
	router.POST("/logout", handlers.Logout)
	router.POST("/passrcv", handlers.PasswordRecoveryPost)
	router.POST("/avatar", handlers.UpdateUserData)

	router.GET("/test-cookie", func(c *gin.Context) {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "TestCookie",
			Value:    "TestValue",
			Path:     "/",
			Domain:   "webchatfront-6xch.vercel.app",
			Expires:  time.Now().Add(24 * time.Hour),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		})
		c.JSON(http.StatusOK, gin.H{"message": "Cookie set!"})
	})

	router.GET("/validate", middleware.RequireAuth, handlers.Validate)
	router.GET("/ws/joinChat/:chatId", wsHandler.JoinChat)
	router.GET("/ws/getClients/:chatId", wsHandler.GetClients)
	router.GET("/room/create", webrtc.RoomCreate)
	router.GET("/room/:uuid", storageHandler.RoomRender)
	router.GET("/room/:uuid/websocket", storageHandler.RoomHandler)
	router.GET("/room/:uuid/chat/websocket", storageHandler.RoomChatWebsocket)

	go dispatchKeyFrames(storage)

	err := router.Run()
	if err != nil {
		log.Fatal("failed to start server")
	}
}

func dispatchKeyFrames(s *webrtc.Storage) {
	for range time.NewTicker(time.Second * 3).C {
		for _, room := range s.Rooms {
			room.Peers.DispatchKeyFrame()
		}
	}
}
