# messenger-service

Backend for Messenger with
* Fiber
* JWT (access/refresh)
* OTP
* Casbin
* Socket.io
* RabbitMQ
* Postgres
* Redis
* GORM

## Sockets

```go
import (
	"messenger-service/socketio"
)

// Emit broadcast event
socketio.Broadcast(event, data)

// Emit user event
socketio.Emit(user, event, data)
```

## Events

### Emit
```go
// main.go

package main

import (
	"messenger-service/event"
)

// Connect to RabbitMQ and declare queues
event.RabbitMQConnect([]string{"api"})

// Emit event to [api] queue
event.Emit("api", "action", "data")
```

### Subscribe
```go
// main.go

package main

import (
	"messenger-service/event"
	"messenger-service/event/listener"
)

// Connect to RabbitMQ and declare queues
event.RabbitMQConnect([]string{"api"})

// Run "api" listener
go listener.Api()

// Subscribe listener channel to "api" events
event.RabbitMQSubscribe([]event.RabbitMQSubscribeListener{
	{
		Queue:   "api",
		Channel: listener.ApiChannel,
	},
})
```

```go
// event/listener/api.go

package listener

import (
	"messenger-service/event"
)

var (
	ApiChannel = make(chan event.EventChannelData)
)

func Api() {
	for event := range ApiChannel {
		// handler
	}
}
```

## Databases

### Postgres
```go
import (
	"messenger-service/database"
)

if err := database.Postgres.First(&userModel, id).Error; err != nil {
	// error
}
```

### Redis
```go
import (
	"messenger-service/database"
)

if err := database.Redis[0].Set(context.Background(), id, data, 0).Err(); err != nil {
	// error
}
```

### Casbin
```go
import (
	"messenger-service/database"
)

database.Casbin().AddGroupingPolicy(id, role)
```
