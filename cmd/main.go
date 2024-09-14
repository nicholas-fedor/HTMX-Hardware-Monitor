package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/nicholas-fedor/htmx-hardware-monitor/internal/hardware"
)

type server struct {
	subscriberMessageBuffer int
	subscribers             map[*subscriber]struct{}
	subscribersMutex        sync.Mutex
	mux                     http.ServeMux
}

type subscriber struct {
	msgs chan []byte
}

var refreshDelay time.Duration = 3 // Seconds

func NewServer() *server {
	s := &server{
		subscriberMessageBuffer: 10,
		subscribers:             make(map[*subscriber]struct{}),
	}

	s.mux.Handle("/", http.FileServer(http.Dir("./src/htmx")))
	s.mux.HandleFunc("/ws", s.subscribeHandler)

	return s
}

func (s *server) subscribeHandler(writer http.ResponseWriter, req *http.Request) {
	err := s.subscribe(req.Context(), writer, req)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (s *server) addSubscriber(subscriber *subscriber) {
	s.subscribersMutex.Lock()
	s.subscribers[subscriber] = struct{}{}
	s.subscribersMutex.Unlock()
	fmt.Println("Added subscriber", subscriber)
}

func (s *server) subscribe(ctx context.Context, writer http.ResponseWriter, req *http.Request) error {
	var c *websocket.Conn
	subscriber := &subscriber{
		msgs: make(chan []byte, s.subscriberMessageBuffer),
	}
	s.addSubscriber(subscriber)

	c, err := websocket.Accept(writer, req, nil)
	if err != nil {
		return err
	}
	defer c.CloseNow()

	ctx = c.CloseRead(ctx)
	for {
		select {
		case msg := <-subscriber.msgs:
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			err := c.Write(ctx, websocket.MessageText, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *server) broadcast(msg []byte) {
	s.subscribersMutex.Lock()
	for subscriber := range s.subscribers {
		subscriber.msgs <- msg
	}
	s.subscribersMutex.Unlock()
}

func main() {
	fmt.Println("Starting system monitor...")

	srv := NewServer()

	go func(s *server) {
		for {
			systemData, err := hardware.GetSystemSection()
			if err != nil {
				fmt.Println(err)
			}
			diskData, err := hardware.GetDiskSection()
			if err != nil {
				fmt.Println(err)
			}
			cpuData, err := hardware.GetCpuSection()
			if err != nil {
				fmt.Println(err)
			}

			timeStamp := time.Now().Format("2006-01-02 15:04:05")

			html := `
			<div hx-swap-oob="innerHTML:#update-timestamp"> ` + timeStamp + `</div>
			<div hx-swap-oob="innerHTML:#system-data"> ` + systemData + `</div>
			<div hx-swap-oob="innerHTML:#disk-data"> ` + diskData + `</div>
			<div hx-swap-oob="innerHTML:#cpu-data"> ` + cpuData + `</div>
			`

			s.broadcast([]byte(html))

			// s.broadcast([]byte(systemSection))
			// s.broadcast([]byte(diskSection))
			// s.broadcast([]byte(cpuSection))

			time.Sleep(refreshDelay * time.Second)
		}
	}(srv)

	err := http.ListenAndServe(":8080", &srv.mux)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
