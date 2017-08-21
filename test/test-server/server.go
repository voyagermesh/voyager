package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Response struct {
	Type       string      `json:"type,omitempty"`
	Host       string      `json:"host,omitempty"`
	PodName    string      `json:"podName,omitempty"`
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
		PodName:    os.Getenv("POD_NAME"),
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

type HttpsServerHandler struct {
	port string
}

func (h HttpsServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := &Response{
		Type:       "http",
		PodName:    os.Getenv("POD_NAME"),
		Host:       r.Host,
		ServerPort: h.port,
		Path:       r.URL.Path,
		Method:     r.Method,
		Headers:    r.Header,
	}
	fmt.Println("Request on url", r.URL.Path)
	json.NewEncoder(w).Encode(resp)
}

func RunHTTPsServerOnPort(port string) {
	fmt.Println("https server running")
	GenCert("http.appscode.dev,ssl.appscode.dev")
	http.ListenAndServeTLS(port, "cert.pem", "key.pem", HttpsServerHandler{port})
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

func main() {
	flag.Parse()

	go RunHTTPServerOnPort(":8080")
	go RunHTTPServerOnPort(":8989")
	go RunHTTPServerOnPort(":9090")

	go RunTCPServerOnPort(":4343")
	go RunTCPServerOnPort(":4545")
	go RunTCPServerOnPort(":5656")

	go RunHTTPsServerOnPort(":6443")
	go RunHTTPsServerOnPort(":3443")

	hold()
}

func hold() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}
