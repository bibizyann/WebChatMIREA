package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func Init() *gorm.DB {
	var err error
	dsn := "host=localhost user=postgres password=ajh4wVNE_ dbname=postgres port=6432 sslmode=disable TimeZone=Europe/Moscow"
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	err = DB.AutoMigrate(&Users{}, &Notifications{}, &Messages{}, &Chats{}, &ChatMembers{}, &Calls{})
	if err != nil {
		log.Fatal(err)
	}

	return DB
}
