package event

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"messenger-service/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventChannelData struct {
	Action string
	Data   []byte
	Out    EventChannelOutData
}
type EventChannelOutData struct {
	Send bool
	Log  bool
}

type RabbitMQSubscribeListener struct {
	Queue   string
	Channel chan EventChannelData
}

type EventLogData struct {
	Time    int64  `json:"time"`
	Service string `json:"service"`
	Action  string `json:"action"`
	Data    string `json:"data"`
}

const RabbitMQActionHeader string = "x-action"
const RabbitMQInLogFile string = "log/in.log"
const RabbitMQOutLogFile string = "log/out.log"

var (
	RabbitMQConnection *amqp.Connection
	RabbitMQChannel    *amqp.Channel
	RabbitMQQueue      = make(map[string]amqp.Queue)
	RabbitMQListeners  = make(map[string]chan EventChannelData)

	InLogFile  *os.File
	OutLogFile *os.File
	err        error
)

func RabbitMQConnect(queues []string) {
	// Connect to RabbitMQ server
	RabbitMQConnection, err = amqp.Dial(fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		config.Config("RABBITMQ_USER"),
		config.Config("RABBITMQ_PASSWORD"),
		config.Config("RABBITMQ_HOST"),
		config.Config("RABBITMQ_PORT"),
	))
	if err != nil {
		panic("failed to connect to RabbitMQ")
	}
	log.Printf("connection opened to RabbitMQ server")

	// Open a RabbitMQ channel
	RabbitMQChannel, err = RabbitMQConnection.Channel()
	if err != nil {
		panic("failed to open a RabbitMQ channel")
	}
	log.Printf("opened a RabbitMQ channel")

	// Declare a queues
	for _, name := range queues {
		queue, err := RabbitMQChannel.QueueDeclare(
			name,  // name
			false, // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			panic("failed to declare a RabbitMQ queue")
		}

		RabbitMQQueue[name] = queue
		log.Printf("success declare a RabbitMQ queue: %s", name)
	}

	// Open event log files
	InLogFile, err = os.OpenFile(RabbitMQInLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	OutLogFile, err = os.OpenFile(RabbitMQOutLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
}

func RabbitMQSubscribe(queues []RabbitMQSubscribeListener) {
	for _, queue := range queues {
		RabbitMQListeners[queue.Queue] = queue.Channel

		msgs, err := RabbitMQChannel.Consume(
			queue.Queue, // queue
			"",          // consumer
			false,       // auto-ack
			false,       // exclusive
			false,       // no-local
			false,       // no-wait
			nil,         // args
		)
		if err != nil {
			panic("failed to register a consumer")
		}
		log.Printf("success subscribe to RabbitMQ [%s] queue", queue.Queue)

		go func() {
			for msg := range msgs {
				action := msg.Headers[RabbitMQActionHeader].(string)

				if config.Config("EVENT_MODE") != "DISABLE" {
					InLog(EventLogData{
						Time:    time.Now().UnixMicro(),
						Service: queue.Queue,
						Action:  action,
						Data:    string(msg.Body[:]),
					})
				}

				msg.Ack(false)

				queue.Channel <- EventChannelData{
					Action: action,
					Data:   msg.Body,
					Out: EventChannelOutData{
						Send: true,
						Log:  true,
					},
				}
			}
		}()
	}
}

func Emit(service string, action string, data []byte, log bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := RabbitMQChannel.PublishWithContext(
		ctx,
		"",      // exchange
		service, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Headers: amqp.Table{
				RabbitMQActionHeader: action,
			},
			Body: data,
		},
	)
	if err != nil {
		panic("failed to publish a message")
	}

	if log && config.Config("EVENT_MODE") != "DISABLE" {
		OutLog(EventLogData{
			Time:    time.Now().UnixMicro(),
			Service: service,
			Action:  action,
			Data:    string(data[:]),
		})
	}
}

func InLog(data EventLogData) {
	eventJson, _ := json.Marshal(data)
	if _, err = InLogFile.WriteString(string(eventJson) + "\n"); err != nil {
		panic(err)
	}
}

func OutLog(data EventLogData) {
	eventJson, _ := json.Marshal(data)
	if _, err = OutLogFile.WriteString(string(eventJson) + "\n"); err != nil {
		panic(err)
	}
}

func Init() {
	switch config.Config("EVENT_MODE") {
	case "IN_SEND_LOG":
		InitIn(EventChannelOutData{
			Send: true,
			Log:  true,
		})
	case "IN_SEND":
		InitIn(EventChannelOutData{
			Send: true,
			Log:  false,
		})
	case "IN":
		InitIn(EventChannelOutData{
			Send: false,
			Log:  false,
		})
	case "OUT":
		InitOut()
	}
}

func InitIn(out EventChannelOutData) {
	inLog, err := os.Open(RabbitMQInLogFile)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	scanner := bufio.NewScanner(inLog)
	for scanner.Scan() {
		data := EventLogData{}
		json.Unmarshal([]byte(scanner.Text()), &data)
		RabbitMQListeners[data.Service] <- EventChannelData{
			Action: data.Action,
			Data:   []byte(data.Data),
			Out:    out,
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	inLog.Close()
}

func InitOut() {
	outLog, err := os.Open(RabbitMQOutLogFile)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	scanner := bufio.NewScanner(outLog)
	for scanner.Scan() {
		data := EventLogData{}
		json.Unmarshal([]byte(scanner.Text()), &data)
		Emit(
			data.Service,
			data.Action,
			[]byte(data.Data),
			false,
		)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	outLog.Close()
}
