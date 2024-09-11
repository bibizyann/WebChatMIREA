package main

import (
	"WebChatMIREA/pkg/database"
	"WebChatMIREA/pkg/database/handlers"
	"WebChatMIREA/pkg/middleware"
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	database.Init()
	router := gin.Default()

	router.POST("/signup", handlers.SignUp)
	router.POST("/login", handlers.Login)
	router.GET("/validate", middleware.RequireAuth, handlers.Validate)

	err := router.Run()
	if err != nil {
		log.Fatal("failed to start server")
	}
}
