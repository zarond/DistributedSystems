package main

import (
	//"context"
	//"flag"
	"fmt"
	"time"

	//"os/exec"
	"math/rand"

	"html"
	"io/ioutil"
	"log"

	"encoding/json"
	//"fmt"
	"net/http"

	"os"
	"os/signal"
	"syscall"

	"github.com/streadway/amqp"
)

var ChannelRequest *amqp.Channel
var QueueRequest amqp.Queue
var QueueResponse amqp.Queue
var msgs <-chan amqp.Delivery
var msgsInner chan reply_msg

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

//func failRestart(err error, msg string) {
//	if err != nil {
//		log.Println("%s: %s", msg, err)
//		time.Sleep(5 * time.Second)
//		goto RESTART
//	}
//}

func main() {
	//flag.Parse()
	//log.SetFlags(log.LstdFlags)

RESTART:

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

	msgs, err = ChannelRequest.Consume(
		QueueResponse.Name, // queue
		"",                 // consumer
		true,               // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	//failOnError(err, "Failed to register a consumer")
	if err != nil {
		log.Println("%s: %s", "Failed to register a consumer", err)
		time.Sleep(5 * time.Second)
		goto RESTART
	}

	msgsInner = make(chan reply_msg, 16)
	go listen_for_replies()

	start( /*ctx*/ )

	// make graceful close
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupt
	log.Println("Interruption received, closing")

}

func start( /*ctx context.Context*/ ) {
	http.HandleFunc("/", answer)
	log.Println("`listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type msg struct {
	Method string `json:"method"`
	Data   string `json:"data"`
}

type reply_msg struct {
	id   string
	data []byte
}

func listen_for_replies() {
	//log.Println("listening for replies")
	for d := range msgs {
		message := reply_msg{d.CorrelationId, d.Body}
		//log.Println("in :", message)
		msgsInner <- message
		//log.Println("reply")
	}
	//log.Println("no more replies")
}

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(65 + rand.Intn(25))
	}
	return string(bytes)
}

func answer(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Fprint(w, "Hello", html.EscapeString(r.URL.Path))
		return
	}
	if value := r.Header.Get("Content-Type"); value != "application/json" {
		msg := fmt.Sprintf("Invalid Content-Type, should be application/json")
		http.Error(w, msg, http.StatusBadRequest)
	} else if r.Method == "PUT" || r.Method == "POST" || r.Method == "VIEW" {
		// send
		corrId := randomString(32)
		text, _ := ioutil.ReadAll(r.Body)
		message := msg{r.Method, string(text)}
		body, _ := json.Marshal(message)
		err := ChannelRequest.Publish(
			"",                // exchange
			QueueRequest.Name, // routing key
			false,             // mandatory
			false,             // immediate
			amqp.Publishing{
				ContentType:   "application/json",
				CorrelationId: corrId,
				Body:          body,
			})
		failOnError(err, "Failed to publish a message")
		//wait for response,
		for d := range msgsInner {
			//log.Println("reply in")
			if corrId == d.id {
				w.Header().Set("Content-Type", "application/json")
				w.Write(d.data)
				//w.WriteHeader(200)
				break
			} else {
				msgsInner <- d
			}
		}

	} else {
		msg := fmt.Sprintf("Invalid HTTP Method")
		http.Error(w, msg, http.StatusBadRequest)
	}
	//log.Println("no more reply in")
}
