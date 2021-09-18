package main

import (
	//"context"
	"flag"
	"fmt"

	//"html"

	"log"

	"encoding/json"
	//"fmt"
	"net/http"

	"os"
	"os/signal"
	"syscall"

	//"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
	"gopkg.in/redis.v3"
)

var RedisClient *redis.Client

var redisaddr = flag.String("redis", "127.0.0.1:6379", "Redis server")
var redisdb = flag.Int64("redisdb", 0, "Redis db")
var redispass = flag.String("redispass", "", "Redis password (default empty)")

var ChannelRequest *amqp.Channel
var QueueRequest amqp.Queue
var QueueResponse amqp.Queue

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags)

	var err error

	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     *redisaddr,
		Password: *redispass,
		DB:       *redisdb,
	})
	_, err = RedisClient.Ping().Result()
	if err != nil {
		log.Panicf("Redis connection err: %s", err)
	}
	defer RedisClient.Close()

	//conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	conn, err := amqp.Dial("amqp://guest:guest@172.17.0.1:5672/")

	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ChannelRequest, err = conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ChannelRequest.Close()

	QueueRequest, err = ChannelRequest.QueueDeclare(
		"tasks_requests", // name
		false,            // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	failOnError(err, "Failed to declare a queue")

	QueueResponse, err = ChannelRequest.QueueDeclare(
		"tasks_responses", // name
		false,             // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ChannelRequest.Consume(
		QueueRequest.Name, // queue
		"",                // consumer
		false,             //true,              // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	failOnError(err, "Failed to register a consumer")

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			messageD := msgDelivery{}
			err := json.Unmarshal(d.Body, &messageD)
			message := msg{}
			var method string
			if err != nil {
				log.Println("message delivery error ", err)
				message.Code = http.StatusBadRequest
				message.Error = "message delivery error " + err.Error()
				goto End
				//d.Ack(false)
				//continue
			}
			method = messageD.Method
			err = json.Unmarshal([]byte(messageD.Data), &message)

			if err != nil {
				log.Println("message data error ", err)
				message.Code = http.StatusBadRequest
				message.Error = "message data error " + err.Error()
				goto End
				//d.Ack(false)
				//continue
			}
			if method == "PUT" {
				if message.Key == "" {
					message.Code = http.StatusBadRequest
					message.Error = "Invalid json payload "
					goto End
					//d.Ack(false)
					//continue
				}
				val, err := RedisClient.Get(message.Key).Result()
				if err != nil {
					var msg string
					if err == redis.Nil {
						msg = fmt.Sprintf("no such key: ", message.Key)
					} else {
						msg = fmt.Sprintf("redis err: ", err)
					}
					message.Code = http.StatusBadRequest
					message.Error = msg
					goto End
					//d.Ack(false)
					//continue
				}
				log.Print("Get Result k:", message.Key, ", v:", val)
				message.Value = val
				message.Code = http.StatusOK

			} else if method == "POST" {
				if message.Key == "" || message.Value == "" {
					message.Code = http.StatusBadRequest
					message.Error = "Invalid json payload"
					goto End
					//d.Ack(false)
					//continue
				}
				err = RedisClient.Set(message.Key, message.Value, 0).Err()
				if err != nil {
					msg := fmt.Sprintf("redis err: ", err)
					message.Code = http.StatusBadRequest
					message.Error = msg
					goto End
					//d.Ack(false)
					//continue
				}
				message.Code = http.StatusOK
			}
		End:
			body, _ := json.Marshal(message)
			err = ChannelRequest.Publish(
				"",                 // exchange
				QueueResponse.Name, // routing key
				false,              // mandatory
				false,              // immediate
				amqp.Publishing{
					ContentType:   "application/json",
					CorrelationId: d.CorrelationId,
					Body:          body,
				})
			d.Ack(false)
			failOnError(err, "Failed to publish a response")
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	//start( /*ctx*/ )

	// make graceful close
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupt
	log.Println("Interruption received, closing")

}

type msgDelivery struct {
	Method string `json:"method"`
	Data   string `json:"data"`
}

type msg struct {
	Code  int    `json:"code"`
	Error string `json:"err"`
	Key   string `json:"key"`
	Value string `json:"value"`
}
