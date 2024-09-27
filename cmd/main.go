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
	router.POST("/ws/createChat", wsHandler.CreateChat)

	router.GET("/validate", middleware.RequireAuth, handlers.Validate)
	router.GET("/ws/joinChat/:chatId", wsHandler.JoinChat)
	router.GET("/ws/getClients/:chatId", wsHandler.GetClients)

	err := router.Run()
	if err != nil {
		log.Fatal("failed to start server")
	}
}
