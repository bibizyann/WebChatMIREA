package main

import (
	"WebChatMIREA/pkg/chat"
	"WebChatMIREA/pkg/database"
	"WebChatMIREA/pkg/database/handlers"
	"WebChatMIREA/pkg/middleware"
	"WebChatMIREA/pkg/webrtc"
	"github.com/gin-gonic/gin"
	"html/template"
	"log"
	"os"
)

func main() {
	database.Init()
	router := gin.Default()
	hub := chat.NewHub()
	wsHandler := chat.NewHandler(hub)
	storage := webrtc.NewStorage()
	storageHandler := webrtc.NewStorageHandler(storage)
	go hub.Run()
	// Read index.html from disk into memory, serve whenever anyone requests /
	indexHTML, err := os.ReadFile("C:/Users/User/GolandProjects/WebChatMIREA/pkg/webrtc/index.html")
	if err != nil {
		panic(err)
	}
	indexTemplate := template.Must(template.New("").Parse(string(indexHTML)))

	router.POST("/signup", handlers.SignUp)
	router.POST("/login", handlers.Login)
	router.POST("/ws/createChat", wsHandler.CreateChat)

	router.GET("/validate", middleware.RequireAuth, handlers.Validate)
	router.GET("/ws/joinChat/:chatId", wsHandler.JoinChat)
	router.GET("/ws/getClients/:chatId", wsHandler.GetClients)
	router.GET("/", func(c *gin.Context) {
		if err := indexTemplate.Execute(c.Writer, "ws://"+c.Request.Host+"/websocket"); err != nil {
			log.Fatal(err)
		}
	})
	router.GET("/room/create", webrtc.RoomCreate)
	router.GET("/room/:uuid", storageHandler.RoomRender)
	router.GET("/room/:uuid/websocket", storageHandler.RoomHandler)
	router.GET("/room/:uuid/chat/websocket", storageHandler.RoomChatWebsocket)

	err = router.Run()
	if err != nil {
		log.Fatal("failed to start server")
	}
}
