package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pp "github.com/pires/go-proxyproto"
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

type HTTPHandler struct {
	port string
}

func (h HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if delay, err := time.ParseDuration(r.URL.Query().Get("delay")); err == nil {
		time.Sleep(delay)
	}

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

func runHTTP(port string) {
	fmt.Println("http server running")
	http.ListenAndServe(port, HTTPHandler{port})
}

type HTTPSHandler struct {
	port string
}

func (h HTTPSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if delay, err := time.ParseDuration(r.URL.Query().Get("delay")); err == nil {
		time.Sleep(delay)
	}

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

func runHTTPS(port string) {
	fmt.Println("https server running")
	GenCert("http.appscode.test,ssl.appscode.test")
	http.ListenAndServeTLS(port, "cert.pem", "key.pem", HTTPSHandler{port})
}

type TCPHandler struct {
	port string
}

func (h TCPHandler) ServeTCP(con net.Conn) {
	fmt.Println("request on", con.LocalAddr().String())
	resp := &Response{
		Type:       "tcp",
		Host:       con.LocalAddr().String(),
		ServerPort: h.port,
	}
	json.NewEncoder(con).Encode(resp)
}

type ProxyAwareHandler struct {
	port string
}

func (h ProxyAwareHandler) ServeTCP(con net.Conn) {
	b := make([]byte, 4096)
	if _, err := con.Read(b); err != nil {
		fmt.Println(err)
	}
	header, err := pp.Read(bufio.NewReader(bytes.NewReader(b)))
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("request on", con.LocalAddr().String())
	resp := &Response{
		Type:       "tcp",
		Host:       con.LocalAddr().String(),
		ServerPort: h.port,
	}

	// set proxy-proto header to response body
	if header != nil {
		resp.Body = fmt.Sprintf("%+v", *header)
	}

	json.NewEncoder(con).Encode(resp)
}

func runTCP(port string) {
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
		h := TCPHandler{port}
		go h.ServeTCP(con)
	}
}

func runProxy(port string) {
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
		h := ProxyAwareHandler{port}
		go h.ServeTCP(con)
	}
}

func main() {
	flag.Parse()

	go runHTTP(":8080")
	go runHTTP(":8989")
	go runHTTP(":9090")

	go runProxy(":4343")
	go runTCP(":4545")
	go runTCP(":5656")

	go runHTTPS(":6443")
	go runHTTPS(":3443")

	hold()
}

func hold() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}
