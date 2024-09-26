package database

import (
	"fmt"
	"log"
	"messenger-service/config"
	"messenger-service/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Postgres *gorm.DB

func PostgresConnect() {
	var err error
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Config("POSTGRES_HOST"),
		config.Config("POSTGRES_PORT"),
		config.Config("POSTGRES_USER"),
		config.Config("POSTGRES_PASSWORD"),
		config.Config("POSTGRES_DB"),
	)
	Postgres, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect postgres")
	}

	log.Printf("Connection opened to Postgres")
	Postgres.AutoMigrate(
		&model.User{},
		&model.MessengerDialog{},
		&model.MessengerMessage{},
		&model.MessengerImage{},
	)
	log.Printf("Postgres Database Migrated")
}
