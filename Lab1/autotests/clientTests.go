package main

import (
	//"context"
	//"flag"
	//"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	//"sync"

	//"os/exec"
	//"os"
	//"os/signal"
	//"syscall"

	//"html"
	//"io/ioutil"
	"log"

	"encoding/json"
	"net/http"
	"time"
	//"encoding/xml"
)

var numberoferrors int

type XMLEntry struct {
	mes string
	err error
}

var OUT []XMLEntry

func main() {
	//time.Sleep(time.Minute)
	//flag.Parse()
	//log.SetFlags(log.LstdFlags)
	samples = [16]msg{{"a", "A"}, {"b", "B"}, {"c", "C"}, {"one", "ValueOne"}, {"two", "ValueTwo"}, {"three", "ValueTree"}, {"qwertyuiop[]", "QWERTYUIOP{}"}, {"asdfghjkl;'\\", "ASDFGHJKL:\"\\"},
		{"`zxcvbnm,./", "~ZXCVBNM<>?"}, {"word", "WORDlongWord"}, {"123456789", "!@#$%^&*("}, {"a1", "A1"}, {"b1", "B1"}, {"c1", "C1"}, {"one1", "ValueOne1"}, {"two1", "ValueTwo1"}}

	start()

	log.Println("All test done")
	writeAllOK()
	if numberoferrors == 0 {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
	// make graceful close
	/*
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-interrupt
		log.Println("Interruption received, closing")*/

}

func start( /*ctx context.Context*/ ) {
	//http.HandleFunc("/", answer)
	//log.Println("`listening on localhost:8080")
	//log.Fatal(http.ListenAndServe(":8080", nil))
	client := &http.Client{}
	client.Timeout = time.Second
	for i := 0; i < 16; i++ { // save
		message := samples[i]
		data, _ := json.Marshal(message)
		req, err := http.NewRequest("POST", "http://127.0.0.1:8080", strings.NewReader(string(data)))
		if err != nil {
			Fail("error creating request", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			Fail("error sending request", err)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		messageGet := msg{}
		status := resp.StatusCode
		err = json.Unmarshal(body, &messageGet)
		if err != nil || status != 200 {
			if err != nil {
				Fail("error response", err)
			} else {
				Fail("error response", nil)
			}
		} else {
			log.Println(i, " OK")
		}
	}
	for i := 0; i < 16; i++ { // get
		message := samples[i]
		realValue := message.Value
		message.Value = ""
		data, _ := json.Marshal(message)
		req, err := http.NewRequest("PUT", "http://127.0.0.1:8080", strings.NewReader(string(data)))
		if err != nil {
			Fail("error creating request", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			Fail("error sending request", err)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		messageGet := msg{}
		status := resp.StatusCode
		err = json.Unmarshal(body, &messageGet)
		if err != nil || status != 200 || messageGet.Value != realValue {
			if err != nil {
				Fail("error response", err)
			} else {
				Fail("error response", nil)
			}
		} else {
			log.Println(i, " OK")
		}
	}
	randomMessages := make([]msg, 0)
	for i := 0; i < 1024; i = i + 16 { // save
		message := msg{randomString(i), randomString(i)}
		randomMessages = append(randomMessages, message)
		data, _ := json.Marshal(message)
		req, err := http.NewRequest("POST", "http://127.0.0.1:8080", strings.NewReader(string(data)))
		if err != nil {
			Fail("error creating request", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			Fail("error sending request", err)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		messageGet := msg{}
		status := resp.StatusCode
		err = json.Unmarshal(body, &messageGet)
		if err != nil || status != 200 {
			if err != nil {
				Fail("error response", err)
			} else {
				Fail("error response", nil)
			}
		} else {
			log.Println(i, " OK")
		}
	}
	for i := 0; i < 64; i++ { // get
		message := randomMessages[i]
		//log.Println(message, len(randomMessages))
		realValue := message.Value
		message.Value = ""
		data, _ := json.Marshal(message)
		req, err := http.NewRequest("PUT", "http://127.0.0.1:8080", strings.NewReader(string(data)))
		if err != nil {
			Fail("error creating request", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			Fail("error sending request", err)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		messageGet := msg{}
		status := resp.StatusCode
		err = json.Unmarshal(body, &messageGet)
		if err != nil || status != 200 || messageGet.Value != realValue {
			if err != nil {
				Fail("error response", err)
			} else {
				Fail("error response", nil)
			}
		} else {
			log.Println(i, " OK")
		}
	}
	log.Println("test ended")
}

type msg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var samples [16]msg

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(65 + rand.Intn(25))
	}
	return string(bytes)
}

func writeXML() {

}

func writeAllOK() {
	if numberoferrors == 0 {
		OUT = append(OUT, XMLEntry{"ALL TEST PASSED", nil})
	}
}

func writeError(mes string, err error) {
	OUT = append(OUT, XMLEntry{mes, err})
}

func Fail(mes string, err error) {
	log.Println(mes, err)
	numberoferrors += 1
	writeError(mes, err)
	if numberoferrors > 50 {
		log.Println("Too many errors, aborting")
		os.Exit(1)
	}
}
