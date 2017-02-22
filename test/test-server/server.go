package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	go RunHTTPServerOnPort(":8080")
	go RunHTTPServerOnPort(":8989")
	go RunHTTPServerOnPort(":9090")

	go RunTCPServerOnPort(":4343")
	go RunTCPServerOnPort(":4545")
	go RunTCPServerOnPort(":5656")

	hold()
}

func hold() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}

type Response struct {
	Type       string      `json:"type,omitempty"`
	Host       string      `json:"host,omitempty"`
	ServerPort string      `json:"serverPort,omitempty"`
	Path       string      `json:"path,omitempty"`
	Method     string      `json:"method,omitempty"`
	Headers    http.Header `json:"headers,omitempty"`
	Body       string      `json:"body,omitempty"`
}

type HttpServerHandler struct {
	port string
}

func (h HttpServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := &Response{
		Type:       "http",
		Host:       r.Host,
		ServerPort: h.port,
		Path:       r.URL.Path,
		Method:     r.Method,
		Headers:    r.Header,
	}
	fmt.Println("Request on url", r.URL.Path)
	json.NewEncoder(w).Encode(resp)
}

func RunHTTPServerOnPort(port string) {
	fmt.Println("http server running")
	http.ListenAndServe(port, HttpServerHandler{port})
}

type TCPServerHandler struct {
	port string
}

func (h TCPServerHandler) ServeTCP(con net.Conn) {
	fmt.Println("request on", con.LocalAddr().String())
	resp := &Response{
		Type:       "tcp",
		Host:       con.LocalAddr().String(),
		ServerPort: h.port,
	}
	json.NewEncoder(con).Encode(resp)
	con.Close()
}

func RunTCPServerOnPort(port string) {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("tcp server listening")
	for {
		con, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		h := TCPServerHandler{port}
		go h.ServeTCP(con)
	}
}
