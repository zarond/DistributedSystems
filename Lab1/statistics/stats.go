package main

import (
	//"context"
	//"flag"
	//"fmt"
	//"os/exec"
	//"math/rand"

	//"html"
	//"io/ioutil"
	"fmt"
	"log"
	"sort"

	"encoding/json"
	//"fmt"
	//"net/http"

	"os"
	"os/signal"
	"syscall"

	"github.com/streadway/amqp"
)

var ChannelInfo *amqp.Channel
var QueueInfo amqp.Queue
var QueueInfoReply amqp.Queue
var msgs <-chan amqp.Delivery

//var msgsInner chan stat_entry
var stats map[string]int

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

type stat_entry struct {
	Clicks int    `json:"n"`
	Word   string `json:"v"`
}

type InfoMsg struct {
	Action string `json:"action"`
	Value  string `json:"value"`
}

type InfoMsgReply struct {
	Action string `json:"action"`
	Value  string `json:"value"`
	Key    string `json:"key"`
}

type jsonArr struct {
	Array []stat_entry
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@172.17.0.1:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ChannelInfo, err = conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ChannelInfo.Close()

	QueueInfo, err = ChannelInfo.QueueDeclare(
		"stat_info", // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	failOnError(err, "Failed to declare a queue")

	QueueInfoReply, err = ChannelInfo.QueueDeclare(
		"stat_info_reply", // name
		false,             // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err = ChannelInfo.Consume(
		QueueInfo.Name, // queue
		"",             // consumer
		true,           // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
	failOnError(err, "Failed to register a consumer")

	//msgsInner = make(chan stat_entry, 16)
	stats = make(map[string]int)
	go listen_for_messages()
	//go doWork()

	// make graceful close
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupt
	log.Println("Interruption received, closing")

}

func listen_for_messages() {
	//log.Println("listening for replies")
	for d := range msgs {
		tmp := InfoMsg{}
		err := json.Unmarshal(d.Body, &tmp)
		if err != nil {
			log.Println("error info msg format ", err, d.Body)
		}
		if tmp.Action == "add" {
			var word string = tmp.Value
			val, ok := stats[word]
			if ok {
				stats[word] = val + 1
			} else {
				stats[word] = 1
				val = 0
			}
			log.Println(word, val)
		}
		if tmp.Action == "get" {
			var word string = tmp.Value
			val, ok := stats[word]
			if !ok {
				val = 0
			}
			log.Println(word, val)
			reply := InfoMsgReply{tmp.Action, fmt.Sprint(val), tmp.Value}
			body, _ := json.Marshal(reply)
			err = ChannelInfo.Publish(
				"",                  // exchange
				QueueInfoReply.Name, // routing key
				false,               // mandatory
				false,               // immediate
				amqp.Publishing{
					ContentType:   "application/json",
					CorrelationId: d.CorrelationId,
					Body:          body,
				})
			failOnError(err, "Failed to publish a response")
		}
		if tmp.Action == "getList" {
			keys := returnsorted()
			for _, kv := range keys {
				fmt.Println(kv.Word, " - ", kv.Clicks)
			}
			reply := jsonArr{keys}
			body, _ := json.Marshal(reply)
			err = ChannelInfo.Publish(
				"",                  // exchange
				QueueInfoReply.Name, // routing key
				false,               // mandatory
				false,               // immediate
				amqp.Publishing{
					ContentType:   "application/json",
					CorrelationId: d.CorrelationId,
					Body:          body,
				})
			failOnError(err, "Failed to publish a response")
		}

	}
	//log.Println("no more replies")
}

func returnsorted() []stat_entry {
	keys := make([]stat_entry, 0, len(stats))
	for k, v := range stats {
		keys = append(keys, stat_entry{v, k})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Clicks > keys[j].Clicks })

	return keys
}

/*
func doWork() {
	for d := range msgsInner {
		fmt.Printf(d.id)
	}
}*/
