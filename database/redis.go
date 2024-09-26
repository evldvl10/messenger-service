package database

import (
	"fmt"
	"log"
	"messenger-service/config"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

var Redis = make(map[int]*redis.Client)

func RedisConnect() {
	for _, db := range strings.Split(config.Config("REDIS_DB"), ",") {
		dbNumber, _ := strconv.Atoi(db)

		options := &redis.Options{
			Addr: fmt.Sprintf(
				"%s:%s",
				config.Config("REDIS_HOST"),
				config.Config("REDIS_PORT"),
			),
			Password: config.Config("REDIS_PASSWORD"),
			DB:       dbNumber,
		}

		Redis[dbNumber] = redis.NewClient(options)
	}

	log.Printf("Connections opened to Redis")
}
