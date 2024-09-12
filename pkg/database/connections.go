package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func Init() *gorm.DB {
	var err error

	dsn := "host=faintly-pro-arapaima.data-1.use1.tembo.io user=postgres password=jhVZoAcnaYp7r4Tr dbname=postgres port=5432 sslmode=require"
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
