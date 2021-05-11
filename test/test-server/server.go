/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pires/go-proxyproto"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

const LOADMAX = 80

type Response struct {
	Type       string             `json:"type,omitempty"`
	Host       string             `json:"host,omitempty"`
	PodName    string             `json:"podName,omitempty"`
	ServerPort string             `json:"serverPort,omitempty"`
	Path       string             `json:"path,omitempty"`
	Method     string             `json:"method,omitempty"`
	Headers    http.Header        `json:"headers,omitempty"`
	Body       string             `json:"body,omitempty"`
	Proxy      *proxyproto.Header `json:"proxy,omitempty"`
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
	utilruntime.Must(json.NewEncoder(w).Encode(resp))
}

func runHTTP(port string) {
	fmt.Println("http server running on port", port)
	klog.Fatal(http.ListenAndServe(port, HTTPHandler{port}))
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
	utilruntime.Must(json.NewEncoder(w).Encode(resp))
}

// cd ~/go/src/voyagermesh.dev/voyager/test/test-server
// go run *.go --ca
// curl --cacert cert.pem 'https://ssl.appscode.test:6443' --resolve ssl.appscode.test:6443:127.0.0.1
func runHTTPS(port string) {
	fmt.Println("https server running on port", port)
	utilruntime.Must(http.ListenAndServeTLS(port, "cert.pem", "key.pem", HTTPSHandler{port}))
}

type TCPServer interface {
	getPort() string
	ServeTCP(net.Conn)
}

type TCPHandler struct {
	port string
}

func (h TCPHandler) getPort() string {
	return h.port
}

func (h TCPHandler) ServeTCP(conn net.Conn) {
	fmt.Println("request on", conn.LocalAddr().String())
	resp := &Response{
		Type:       "tcp",
		Host:       conn.LocalAddr().String(),
		ServerPort: h.port,
	}
	utilruntime.Must(json.NewEncoder(conn).Encode(resp))
}

type ProxyAwareHandler struct {
	port string
}

func (h ProxyAwareHandler) getPort() string {
	return h.port
}

func (h ProxyAwareHandler) ServeTCP(conn net.Conn) {
	header, err := proxyproto.ReadTimeout(bufio.NewReader(conn), time.Second)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("request on", conn.LocalAddr().String())
	resp := &Response{
		Type:       "tcp",
		Host:       conn.LocalAddr().String(),
		ServerPort: h.port,
		Proxy:      header,
	}
	utilruntime.Must(json.NewEncoder(conn).Encode(resp))
}

type agentTCPHandler struct {
	port string
}

func (h agentTCPHandler) getPort() string {
	return h.port
}

func (h agentTCPHandler) ServeTCP(conn net.Conn) {
	fmt.Println("request on", conn.LocalAddr().String())

	ctx := context.Background()
	v, _ := load.AvgWithContext(ctx)

	//totalCPU denotes total number of logical cpus
	totalCPU, _ := cpu.Counts(true)

	//calculate cpu load percentage based on last 5 minutes stats
	load5util := (v.Load5 / float64(totalCPU)) * 100
	fmt.Println("CPU Load Percentage: ", load5util)

	// Rules: if cpu load is less than 80%
	// agent sever output = up 100%
	// if it is 80% to less than 100%, output = drain
	// if 100%, this server will be marked as down

	resp := ""
	if load5util < LOADMAX {
		resp = "up 100%"
	} else if load5util >= LOADMAX && load5util < 100 {
		resp = "drain"
	} else {
		resp = "down#CPU overload"
	}

	_, _ = conn.Write([]byte(resp + "\n"))
}

func runTCP(h TCPServer) {
	port := h.getPort()
	ln, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("tcp server listening on port", port)
	for {
		con, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go h.ServeTCP(con)
	}
}

func main() {
	flag.Parse()

	go runHTTP(":8080")
	go runHTTP(":8989")
	go runHTTP(":9090")

	go runTCP(TCPHandler{":4343"})
	go runTCP(TCPHandler{":4545"})
	go runTCP(TCPHandler{":5656"})

	// run proxy
	go runTCP(ProxyAwareHandler{":6767"})

	GenCert("http.appscode.test,ssl.appscode.test")
	go runHTTPS(":6443")
	go runHTTPS(":3443")

	// run agent-check server
	go runTCP(agentTCPHandler{":5555"})

	hold()
}

func hold() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}
