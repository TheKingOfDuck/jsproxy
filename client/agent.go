package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"golang.org/x/sync/semaphore"
)

type jsproxy struct {
	lock *semaphore.Weighted
}

type Request struct {
	Method  string
	Proto   string
	URL     string
	Headers string
	Body    string
}

type Response struct {
	Status string
	Body   string
}

func proxyHttp(request Request, timeout time.Duration) Response {

	req, err := http.NewRequest(request.Method, request.URL, bytes.NewReader([]byte(request.Body)))
	if err != nil {
		// 处理错误
	}
	for _, header := range strings.Split(request.Headers, ";") {
		// log.Print(header)
		if strings.Contains(header, ": ") {
			key := strings.Split(header, ": ")[0]
			val := strings.Split(header, ": ")[1]
			req.Header.Add(key, val)
		}
	}

	// req.Header.Add("js.fetch:mode", "no-cors")

	// req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// 处理错误
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
	}

	response := Response{
		Status: resp.Status,
		Body:   string(body),
	}
	log.Println(response)

	return response

}

func (jp *jsproxy) proxy(timeout time.Duration) {

	wg := sync.WaitGroup{}
	jp.lock.Acquire(context.TODO(), 1)

	// 创建WebSocket连接
	ws := js.Global().Get("WebSocket").New("ws://localhost:8000/ws")

	// receive
	// body := make(chan string)
	onMessage := js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		if len(args) > 0 {
			msg := args[0].Get("data").String()

			if msg != "pong" {
				wg.Add(1)

				var request Request
				json.Unmarshal([]byte(msg), &request)
				log.Println(request)

				go func(request Request, timeout time.Duration) {

					response := proxyHttp(request, timeout)
					responseJSON, _ := json.Marshal(response)

					log.Println("http response:", string(responseJSON))
					ws.Call("send", string(responseJSON))
					defer wg.Done()

				}(request, timeout)

			} else {
				log.Println("received:", msg)
			}
		}
		return nil
	})
	defer onMessage.Release()
	ws.Set("onmessage", onMessage)

	wg.Wait()

	// 握手
	heartbeat := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Println("send:", "ping")
		ws.Call("send", "ping")

		return nil
	})
	defer heartbeat.Release()
	ws.Set("onopen", heartbeat)

	log.Println("jsproxy agent injected...")
	http.ListenAndServe(":8080", nil)
	defer jp.lock.Release(1)

}

func main() {
	jp := &jsproxy{
		lock: semaphore.NewWeighted(200),
	}
	jp.proxy(500 * time.Millisecond)
}
