package main

import (
	"WebChatMIREA/pkg/chat"
	"WebChatMIREA/pkg/database"
	"WebChatMIREA/pkg/database/handlers"
	"WebChatMIREA/pkg/middleware"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	database.Init()
	router := gin.Default()
	hub := chat.NewHub()
	wsHandler := chat.NewHandler(hub)
	go hub.Run()

	router.POST("/signup", handlers.SignUp)
	router.POST("/login", handlers.Login)
	router.POST("/ws/createRoom", wsHandler.CreateRoom)

	router.GET("/validate", middleware.RequireAuth, handlers.Validate)
	router.GET("/ws/joinRoom/:roomId", wsHandler.JoinRoom)

	err := router.Run()
	if err != nil {
		log.Fatal("failed to start server")
	}
}
