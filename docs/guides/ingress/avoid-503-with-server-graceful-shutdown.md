---
title: Avoid 503 with Server Graceful Shutdown | Voyager
menu:
  product_voyager_10.0.0:
    identifier: avoid-503-with-server-graceful-shutdown
    name: Avoid 503 with Server Graceful Shutdown
    parent: ingress-guides
    weight: 50
product_name: voyager
menu_name: product_voyager_10.0.0
section_menu_id: guides
---
> New to Voyager? Please start [here](/docs/concepts/overview.md).

# Avoid 503 with Server Graceful Shutdown

Multiple voyager users have been facing an issue regarding getting 503 response
randomly after requesting through ingress. So we tried to look into what was going wrong and here's
the solution and the reason behind this problem.

HAProxy supports a graceful and a hard stop. For voyager, we are using graceful stop. The graceful stop is triggered when the
SIGUSR1 signal is sent to the haproxy process. It consists in only unbinding
from listening ports, but continue to process existing connections until they
close. Once the last connection is closed, the process leaves.

So suppose you scaled down or deleted some of your pods. These incidents will happen simultaneously:
1. The pod will go into terminating status and drop all the existing connections and you will get 503 response.
2. It will be removed from service endpoint.
3. HAProxy configuration will be reloaded - the dead pods will be removed from the config (and new pods will be inserted if that's the case).

If you do have a graceful shutdown for your backend pods, as soon as your backend pod dies, it will be removed from
your service endpoint and HAProxy configuration - but the pod will (hopefully) finish serving the existing requests.

Here are 2 ways for the pod to finish serving existing requests.

### 1. Catch SIGTERM and Ignore

So how are we gonna implement the graceful shutdown of the backend server?

From kubernetes doc:

> When a user requests deletion of a Pod, the system records the intended grace period 
before the Pod is allowed to be forcefully killed, and a TERM signal is sent to the main process 
in each container. Once the grace period has expired, the KILL signal is sent to those processes, 
and the Pod is then deleted from the API server.

One difference between SIGTERM and SIGKILL is - the process can register a handler for SIGTERM and 
choose to ignore it. But SIGKILL cannot be ignored.

So when the containers in your pod receives SIGTERM, in default state when you don't catch it,
the container is killed immediately. But as you want to finish serving existing connections, you need to
catch this signal to wait for grace period to be finished (mentioned by `terminationGracePeriodSeconds`
in pod's spec). After this period ends, SIGKILL is sent to the container and there's nothing else you can 
do to stop it from being dead!

One thing worth mentioning here, as soon as a pod receive SIGTERM it goes into `terminating` state no matter whether
you handle this signal or not (and new pods are created if your deployment needs). Also, simultaneously with this,
this pod is removed from endpoints list for corresponding service.

Now, what are you gonna do after catching SIGTERM? Here are two samples:

#### i. Add Sleep

A simple example is shown here written Golang. Let's say you have one server running like this:
```
http.ListenAndServe(":8080", nil)
```

Put that in a goroutine and add these lines to your main process

```
go http.ListenAndServe(":8080", nil)

ch := make(chan os.Signal, 1)
signal.Notify(ch, syscall.SIGTERM)

<-ch
time.Sleep(30 * time.Second)
```

This `30` came from the default value of `terminationGracePeriodSeconds`. These lines are gonna 
catch the SIGTERM signal and make sure that your server in the other process runs until the grace period ends.

Full code:

```
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Print("Hello world received a request.")
	fmt.Fprintf(w, "Hello %s!\n", "world")
}

func main() {

	log.Print("Hello world sample started.")
	http.HandleFunc("/", handler)
	go http.ListenAndServe(":8080", nil)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	<-ch
	time.Sleep(30 * time.Second)

}

```

#### ii. Add Server Shutdown

You can also greacefully shutdown each server individually after catching SIGTERM.


```
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type httpHandler struct {
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(5 * time.Second)
	log.Print("Hello world received a request.")
	fmt.Fprintf(w, "Hello %s!\n", "world")
}

func main() {

	log.Print("Hello world sample started.")
	server := &http.Server{Addr: ":8080", Handler: httpHandler{}}
	go server.ListenAndServe()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	<-ch

	time.Sleep(5 * time.Second)		// in case pod receives new request

	ctx, _ := context.WithTimeout(context.Background(), 22*time.Second)
	if err := server.Shutdown(ctx); err != nil {
	}

}

```

When `server.Shutdown(ctx)` is called, it stops taking new requests and tries to finish serving all current requests, but doesn't wait more than 22 seconds.

### 2. Add a preStop hook

You can also add a prestop hook to container to make sure the container isn't killed until grace period ends.

```
spec:
      containers:
      - image: kfoozminus/graceful:v1
        imagePullPolicy: Always
        lifecycle:
          preStop:
            exec:
              command:
              - sleep
              - "30"
        name: nginx
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
```
preStop is executed before SIGTERM is sent. So in this case, SIGTERM is sent 30s after shutdown is initiated. The preStop hook cannot block more than the grace period though. If "sleep 60" was executed as preStop hook, the pod will terminate after 30s, that is, the grace period.


