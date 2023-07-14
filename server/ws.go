package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

var wsConn *websocket.Conn

var m sync.Map

type Response struct {
	Status string
	Body   string
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	rawRequest, err := http.ReadRequest(bufio.NewReader(conn)) // 读取请求

	if err != nil {
		log.Printf("Failed to read request: %s", err)
		return
	}

	headers := ""
	for k, v := range rawRequest.Header {
		for _, vv := range v {
			h := k + ": " + vv
			if !strings.HasPrefix(h, "Proxy-") {
				headers += h + ";"
			}

		}
	}

	// 读取HTTP请求的Body字段
	body, err := ioutil.ReadAll(rawRequest.Body)
	if err != nil {
		// fmt.Println("Failed to read request body:", err)
		return
	}

	type Request struct {
		Method  string
		Proto   string
		URL     string
		Headers string
		Body    string
	}

	request := Request{
		Method:  rawRequest.Method,
		Proto:   rawRequest.Proto,
		URL:     rawRequest.URL.String(),
		Headers: headers,
		Body:    string(body),
	}

	requestJSON, _ := json.Marshal(request)

	log.Println("requestJSON:", string(requestJSON))

	// 向客户端发送消息
	if err := websocket.Message.Send(wsConn, string(requestJSON)); err != nil {
		// log.Println("发送消息失败:", err.Error())
		return
	}

	for i := 0; i < 6; i++ {
		if val, ok := m.Load("msg"); ok {
			str := fmt.Sprintf("%s", val)

			var response Response
			json.Unmarshal([]byte(str), &response)

			conn.Write([]byte("HTTP/1.1 " + response.Status + "\r\n"))
			conn.Write([]byte("Content-Length: " + strconv.Itoa(len(response.Body)) + "\r\n"))
			conn.Write([]byte("Connection: close\r\n"))
			conn.Write([]byte("\r\n"))
			io.Copy(conn, strings.NewReader(response.Body))
			m.Delete("msg")
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	msg := "client not ready or request error"
	conn.Write([]byte("HTTP/1.1 999 ERROR\r\n"))
	conn.Write([]byte("Content-Type: application/json; charset=utf-8\r\n"))
	conn.Write([]byte("Content-Length: " + strconv.Itoa(len(msg)) + "\r\n"))
	conn.Write([]byte("Connection: close\r\n"))
	conn.Write([]byte("\r\n"))
	io.Copy(conn, strings.NewReader(msg))

}

func wsHandler(conn *websocket.Conn) {

	wsConn = conn
	// 向客户端发送消息

	// 接收客户端消息
	for {
		var msg string
		if err := websocket.Message.Receive(wsConn, &msg); err != nil {
			// fmt.Println("接收消息失败:", err.Error())
			return
		}
		log.Println("ws received:", msg)

		if msg != "ping" {
			m.Store("msg", msg)
		} else {
			if err := websocket.Message.Send(wsConn, "pong"); err != nil {
				log.Println(err.Error())
				return
			}
		}
	}
}

func main() {

	websocketPort := flag.String("wp", "8000", "websocket server port")
	httpProxyPort := flag.String("hp", "8080", "websocket server port")
	flag.Parse()

	wsaddr := "0.0.0.0:" + *websocketPort
	hpaddr := "0.0.0.0:" + *httpProxyPort

	// 注册WebSocket路由
	http.Handle("/ws", websocket.Handler(wsHandler))

	// 启动HTTP服务器
	log.Println("websocket server running:", wsaddr)
	go http.ListenAndServe(wsaddr, nil)

	log.Println("http proxy server running:", hpaddr)
	tcporxy, err := net.Listen("tcp", hpaddr)
	if err != nil {
		os.Exit(1)
	}

	for {
		proxyConn, err := tcporxy.Accept() // 接收请求
		if err != nil {
			continue
		}

		go handleConnection(proxyConn) // 处理请求
	}
}
