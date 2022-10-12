package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

var (
	sed             = rand.NewSource(time.Now().UnixNano())
	messageBuffsize = 1024
	broker          = &Broker{
		Messages:       make(chan string, messageBuffsize),
		newClients:     make(chan clientContext),
		closingClients: make(chan string),
		clients:        make(map[string]chan string),
	}
)

type clientContext struct {
	ID     string
	Buffer chan string
}

type Broker struct {
	// Events are pushed to this channel by the main events-gathering routine
	Messages chan string

	// New client connections
	newClients chan clientContext

	// Closed client connections
	closingClients chan string

	// Client connections registry
	clients map[string]chan string
}

func (b *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:
			broker.clients[s.ID] = s.Buffer
			log.Printf("Client added. %s, %d registered clients", s.ID, len(broker.clients))
		case clientID := <-broker.closingClients:
			delete(broker.clients, clientID)
			log.Printf("Removed client. %s, %d registered clients", clientID, len(broker.clients))
		case msg := <-broker.Messages:
			log.Println("received message", msg, len(broker.clients))
			for _, clientMessageChan := range broker.clients {
				log.Println("send message")
				clientMessageChan <- msg
			}
		}
	}
}

func handleSse(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	messageChan := make(chan string)
	clientID := fmt.Sprintf("%d-%d", time.Now().Nanosecond(), rand.New(sed).Int())
	broker.newClients <- clientContext{ID: clientID, Buffer: messageChan}

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		log.Println("set body writer")
		for msg := range messageChan {
			log.Println("send message", msg)
			fmt.Fprintf(w, "data: %s\n\n", msg)
			err := w.Flush()
			if err != nil {
				// Refreshing page in web browser will establish a new
				// SSE connection, but only (the last) one is alive, so
				// dead connections must be closed here.
				fmt.Printf("Error while flushing: %v. Closing http connection.\n", err)
				broker.closingClients <- clientID
				break
			}
		}
	}))

	return nil
}
