package backend

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nicholas-fedor/htmx-hardware-monitor/pkg/backend/internal/hardware"
	"github.com/spf13/viper"
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
	// Allow CORS from localhost:5500
	writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5500")
	writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if req.Method == "OPTIONS" {
		// Handle preflight request
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	err := s.subscribe(req.Context(), writer, req)
	if err != nil {
		// Log the error but don't send headers again if WebSocket upgrade has started
		fmt.Println("Failed to subscribe:", err)
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
	subscriber := &subscriber{
		msgs: make(chan []byte, s.subscriberMessageBuffer),
	}
	s.addSubscriber(subscriber)

	// Use a custom Upgrader to handle CORS checks
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			fmt.Println("Received request with Origin:", req.Header.Get("Origin"))
			// Allow connections from localhost:5500
			return r.Header.Get("Origin") == "http://localhost:5500"
		},
	}

	c, err := upgrader.Upgrade(writer, req, nil)
	if err != nil {
		return fmt.Errorf("failed to upgrade to WebSocket: %v", err)
	}
	defer c.Close()

	ctx, closeCancel := context.WithCancel(ctx)
	go func() {
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				closeCancel()
				return
			}
		}
	}()

	for {
		select {
		case msg := <-subscriber.msgs:
			// Create a new context with timeout for each message send operation
			writeCtx, writeCancel := context.WithTimeout(ctx, time.Second)
			defer writeCancel() // Ensure we cancel this context after we're done with it
			err := c.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				return fmt.Errorf("failed to write message: %v", err)
			}
			// Use writeCtx to ensure the operation respects the timeout
			select {
			case <-writeCtx.Done():
				if writeCtx.Err() == context.DeadlineExceeded {
					return fmt.Errorf("operation timed out")
				}
			default:
				// Operation completed within the timeout
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

func Execute() {
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

			timeStamp := time.Now().Format("2006-01-02 15:04:05")

			html := `
			<div hx-swap-oob="innerHTML:#update-timestamp"> ` + timeStamp + `</div>
			<div hx-swap-oob="innerHTML:#system-data"> ` + systemData + `</div>
			<div hx-swap-oob="innerHTML:#disk-data"> ` + diskData + `</div>
			`

			s.broadcast([]byte(html))

			time.Sleep(refreshDelay * time.Second)
		}
	}(srv)

	port := viper.GetString("backend.port")
	log.Printf("Backend server listening on :%s...", port)
	if err := http.ListenAndServe(":"+port, &srv.mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Backend server error: %v", err)
	}
}
