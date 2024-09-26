package main

import (
	"fmt"
	"log"
	"messenger-service/config"
	"messenger-service/database"
	"messenger-service/event"
	"messenger-service/event/listener"
	"messenger-service/router"
	"messenger-service/socketio"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	log.SetPrefix("messenger-service: ")

	rest := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		StrictRouting:         true,
		AppName:               "messenger-service",
	})

	rest.Use(cors.New())

	database.RedisConnect()
	database.PostgresConnect()

	event.RabbitMQConnect([]string{
		// Connect to queues
		"api",
		"backoffice",
	})

	// Run "ome" listener
	go listener.Api()

	// Subscribe listener channel to "api" events
	event.RabbitMQSubscribe([]event.RabbitMQSubscribeListener{
		{
			Queue:   "api",
			Channel: listener.ApiChannel,
		},
	})

	// Init event logs
	event.Init()

	socket := socketio.Init(rest)

	router.Rest(rest)
	router.Socket(socket)

	go rest.Listen(fmt.Sprintf(":%s", config.Config("SERVER_PORT")))

	exit := make(chan struct{})
	SignalC := make(chan os.Signal, 1)

	signal.Notify(SignalC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				close(exit)
				return
			}
		}
	}()

	<-exit
	socket.Close(nil)
	event.RabbitMQChannel.Close()
	event.RabbitMQConnection.Close()
	event.InLogFile.Close()
	event.OutLogFile.Close()
	os.Exit(0)
}
