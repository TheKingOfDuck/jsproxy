package main

import (
	"flag"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	ip := flag.String("i", "0.0.0.0", "IP to bind to")
	port := flag.String("p", "80", "Port to bind to")
	flag.Parse()
	r := gin.Default()
	r.Use(cors.Default())
	r.StaticFile("/", "../client/index.html")
	r.StaticFile("/sw.js", "../client/sw.js")
	r.StaticFile("/wasm_exec.js", "../client/wasm_exec.js")
	r.StaticFile("/agent.wasm", "../client/agent.wasm")

	addr := *ip + ":" + *port
	r.Run(addr)
}
