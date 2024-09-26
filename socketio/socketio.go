package socketio

import (
	"context"
	"time"

	"messenger-service/database"
	"messenger-service/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/zishang520/engine.io/v2/log"
	"github.com/zishang520/socket.io-go-redis/adapter"
	r_type "github.com/zishang520/socket.io-go-redis/types"
	"github.com/zishang520/socket.io/v2/socket"
)

var server *socket.Server

func Init(app *fiber.App) *socket.Server {
	log.DEBUG = true

	options := socket.DefaultServerOptions()
	options.SetServeClient(true)
	options.SetAllowEIO3(true)
	options.SetPingInterval(300 * time.Millisecond)
	options.SetPingTimeout(200 * time.Millisecond)
	options.SetMaxHttpBufferSize(100000000)
	options.SetConnectTimeout(1000 * time.Millisecond)
	options.SetAdapter(&adapter.RedisAdapterBuilder{
		Redis: r_type.NewRedisClient(context.Background(), database.Redis[1]),
		Opts:  &adapter.RedisAdapterOptions{},
	})

	server = socket.NewServer(nil, nil)

	server.Use(func(client *socket.Socket, next func(*socket.ExtendedError)) {
		token, auth := client.Conn().Request().Query().Get("token")

		if auth {
			claims, err := utils.CheckAndExtractTokenMetadata(token, "JWT_ACCESS_KEY")

			if err == nil {
				if !claims.Otp {
					client.Join(socket.Room(claims.Id))
					client.SetData(claims)
				}
			}
		}

		next(nil)
	})

	app.Get("/socket.io/", adaptor.HTTPHandler(server.ServeHandler(options)))
	app.Post("/socket.io/", adaptor.HTTPHandler(server.ServeHandler(options)))

	return server
}

func Broadcast(event string, message any) {
	server.FetchSockets()(func(sockets []*socket.RemoteSocket, _ error) {
		for _, socket := range sockets {
			socket.Emit(event, message)
		}
	})
}

func Emit(id string, event string, message any) {
	server.To(socket.Room(id)).Emit(event, message)
}
