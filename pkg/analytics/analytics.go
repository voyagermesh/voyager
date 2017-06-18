package analytics

import (
	"strings"
	"sync"

	"github.com/appscode/log"
	ga "github.com/jpillora/go-ogle-analytics"
)

const (
	trackingID = "UA-62096468-13"
)

var (
	mu     sync.Mutex
	client *ga.Client
)

func Enable() {
	mu.Lock()
	defer mu.Unlock()
	client = mustNewClient()
}

func Disable() {
	mu.Lock()
	defer mu.Unlock()
	client = nil
}

func send(e *ga.Event) {
	mu.Lock()
	c := client
	mu.Unlock()

	if c == nil {
		return
	}

	// error is ignored intentionally. we try to send event to GA in a best effort approach.
	c.Send(e)
}

func mustNewClient() *ga.Client {
	client, err := ga.NewClient(trackingID)
	if err != nil {
		log.Fatalln(err)
	}
	return client
}

func Send(category, action string, label ...string) {
	ev := ga.NewEvent(category, action)
	if len(label) > 0 {
		ev.Label(strings.Join(label, ","))
	}
	send(ev)
}
