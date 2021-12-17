package main

import (
	//"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

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
var msgs <-chan amqp.Delivery

var QueueInfo amqp.Queue
var QueueInfoReply amqp.Queue
var StatsMessages <-chan amqp.Delivery

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
RESTART:

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     *redisaddr,
		Password: *redispass,
		DB:       *redisdb,
	})
	_, err = RedisClient.Ping().Result()
	//if err != nil {
	//	log.Panicf("Redis connection err: %s", err)
	//}
	if err != nil {
		log.Println("%s: %s", "Redis connection err:", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}
	defer RedisClient.Close()

	//conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	conn, err := amqp.Dial("amqp://guest:guest@172.17.0.1:5672/")

	//failOnError(err, "Failed to connect to RabbitMQ")
	if err != nil {
		log.Println("%s: %s", "Failed to connect to RabbitMQ", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}
	defer conn.Close()

	ChannelRequest, err = conn.Channel()
	//failOnError(err, "Failed to open channel")
	if err != nil {
		log.Println("%s: %s", "Failed to open channel", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}
	defer ChannelRequest.Close()

	QueueRequest, err = ChannelRequest.QueueDeclare(
		"tasks_requests", // name
		false,            // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	//failOnError(err, "Failed to declare a queue")
	if err != nil {
		log.Println("%s: %s", "Failed to declare a queue", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}

	QueueResponse, err = ChannelRequest.QueueDeclare(
		"tasks_responses", // name
		false,             // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	//failOnError(err, "Failed to declare a queue")
	if err != nil {
		log.Println("%s: %s", "Failed to declare a queue", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}

	QueueInfo, err = ChannelRequest.QueueDeclare(
		"stat_info", // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	//failOnError(err, "Failed to declare a queue")
	if err != nil {
		log.Println("%s: %s", "Failed to declare a queue", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}

	QueueInfoReply, err = ChannelRequest.QueueDeclare(
		"stat_info_reply", // name
		false,             // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	//failOnError(err, "Failed to declare a queue")
	if err != nil {
		log.Println("%s: %s", "Failed to declare a queue", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}

	msgs, err = ChannelRequest.Consume(
		QueueRequest.Name, // queue
		"",                // consumer
		false,             //true,              // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	//failOnError(err, "Failed to register a consumer")
	if err != nil {
		log.Println("%s: %s", "Failed to register a consumer", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}

	StatsMessages, err = ChannelRequest.Consume(
		QueueInfoReply.Name, // queue
		"",                  // consumer
		true,                // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	//failOnError(err, "Failed to register a consumer")
	if err != nil {
		log.Println("%s: %s", "Failed to register a consumer", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}

	/*
		go func() {
			for d := range StatsMessages {
				message := reply_msg{d.CorrelationId, d.Body}
				msgsInner <- message
			}
		}()*/
	go passStatMessages()

	go listenToGateway()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	//start( /*ctx*/ )

	// make graceful close
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupt
	log.Println("Interruption received, closing")

}

func listenToGateway() {
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
			message.Time = time.Now().Unix()
			goto End
		}
		method = messageD.Method
		err = json.Unmarshal([]byte(messageD.Data), &message)

		if err != nil {
			log.Println("message data error ", err)
			message.Code = http.StatusBadRequest
			message.Error = "message data error " + err.Error()
			message.Time = time.Now().Unix()
			goto End
		}
		if method == "PUT" { // get value
			if message.Key == "" {
				message.Code = http.StatusBadRequest
				message.Error = "Invalid json payload "
				message.Time = time.Now().Unix()
				goto End
			}
			val, err := RedisClient.Get(message.Key).Result()
			if err != nil {
				var msg string
				if err == redis.Nil {
					msg = fmt.Sprint("no such key: ", message.Key)
				} else {
					msg = fmt.Sprint("redis err: ", err)
				}
				message.Code = http.StatusBadRequest
				message.Error = msg
				message.Time = time.Now().Unix()
				goto End
			}
			log.Print("Get Result k:", message.Key, ", v:", val)
			// need to parse redis answer
			redismessage := RedisMsg{}
			err = json.Unmarshal([]byte(val), &redismessage)
			if err != nil {
				log.Println("redis data inner structure error ", err)
				message.Value = val
				message.Code = http.StatusBadRequest
				message.Error = "redis data inner structure error " + err.Error()
				message.Time = time.Now().Unix()
				goto End
			}
			message.Value = redismessage.Data //val
			message.Code = http.StatusOK
			message.Time = time.Now().Unix()
			message.TimeUpdated = redismessage.LastUpdated
			message.TimeUpHuman = time.Unix(redismessage.LastUpdated, 0).Format(time.ANSIC)
			// send to statistic
			go sendToStat(message.Key, 0)

		} else if method == "POST" { // save value
			if message.Key == "" || message.Value == "" {
				message.Code = http.StatusBadRequest
				message.Error = "Invalid json payload"
				message.Time = time.Now().Unix()
				goto End
			}
			redismessage := RedisMsg{}
			redismessage.Data = message.Value
			redismessage.LastUpdated = time.Now().Unix()
			tmp, _ := json.Marshal(redismessage)
			//err = RedisClient.Set(message.Key, message.Value, 0).Err()
			err = RedisClient.Set(message.Key, string(tmp), 0).Err()
			if err != nil {
				msg := fmt.Sprint("redis err: ", err)
				message.Code = http.StatusBadRequest
				message.Error = msg
				message.Time = time.Now().Unix()
				goto End
			}
			message.Code = http.StatusOK
			message.Time = time.Now().Unix()
			message.TimeUpdated = message.Time
			message.TimeUpHuman = time.Unix(message.Time, 0).Format(time.ANSIC)
		}
		if method == "VIEW" {
			go statRequestResponse(messageD.Data, d.CorrelationId)
			d.Ack(false)
			continue
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
}

type msgDelivery struct {
	Method string `json:"method"`
	Data   string `json:"data"`
}

type msg struct {
	Code        int    `json:"code"`
	Error       string `json:"err"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Time        int64  `json:"time"`
	TimeUpdated int64  `json:"time_updated"`
	TimeUpHuman string `json:"time_updated_human"`
}

type RedisMsg struct {
	Data        string `json:"value"`
	LastUpdated int64  `json:"time"`
}

type InfoMsg struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

type InfoMsgReply struct {
	Action string `json:"action"`
	Value  string `json:"value"`
	Error  string `json:"err"`
	Time   int64  `json:"time"`
}

func sendToStat(word string, action int) {
	tmp := InfoMsg{}
	if action == 0 {
		tmp.Action = "add"
		tmp.Value = word
	}
	if action == 1 {
		tmp.Action = "get"
		tmp.Value = word
	}
	if action == 2 {
		tmp.Action = "getList"
		tmp.Value = word
	}
	corrId := randomString(32)
	sendToStatO(tmp, corrId)
}

func sendToStatO(tmp InfoMsg, id string) {
	txt, _ := json.Marshal(tmp)
	err := ChannelRequest.Publish(
		"",             // exchange
		QueueInfo.Name, // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: id,
			Body:          txt,
		})
	failOnError(err, "Failed to publish a stat info")
}

type reply_msg struct {
	id   string
	data []byte
}

//var msgsInner chan reply_msg

func statRequestResponse(data string, corrId string) {
	tmp := InfoMsg{}
	reply := InfoMsgReply{}
	err := json.Unmarshal([]byte(data), &tmp)
	if err != nil {
		msg := fmt.Sprint("info json err: ", err)
		reply.Error = msg
		tmp.Action = "none"
	}
	reply.Time = time.Now().Unix()
	//corrId := randomString(32)
	if tmp.Action == "add" {
		sendToStatO(tmp, corrId)
		txt, _ := json.Marshal(reply)
		err = ChannelRequest.Publish(
			"",                 // exchange
			QueueResponse.Name, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType:   "application/json",
				CorrelationId: corrId,
				Body:          txt,
			})
		failOnError(err, "Failed to publish a stat info")
	} else if tmp.Action == "get" || tmp.Action == "getList" {
		sendToStatO(tmp, corrId)

	} else {
		txt, _ := json.Marshal(reply)
		err = ChannelRequest.Publish(
			"",                 // exchange
			QueueResponse.Name, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType:   "application/json",
				CorrelationId: corrId,
				Body:          txt,
			})
		failOnError(err, "Failed to publish a stat info")
	}
}

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(65 + rand.Intn(25))
	}
	return string(bytes)
}

func passStatMessages() {
	for d := range StatsMessages {
		err := ChannelRequest.Publish(
			"",                 // exchange
			QueueResponse.Name, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType:   "application/json",
				CorrelationId: d.CorrelationId,
				Body:          d.Body,
			})
		failOnError(err, "Failed to publish a stat info")
	}
}
