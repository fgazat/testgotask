package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
)

func main() {
	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		fmt.Printf("Error connecting to NATS: %v\n", err)
		return
	}
	defer nc.Close()

	// Subscribe to the "updates" subject
	sub, err := nc.SubscribeSync("updates")
	if err != nil {
		fmt.Printf("Error subscribing to subject: %v\n", err)
		return
	}

	// Handle system interrupts for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			// Wait for a message
			msg, err := sub.NextMsg(nats.DefaultTimeout)
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				fmt.Printf("Error receiving message: %v\n", err)
				continue
			}
			msg.Respond([]byte("hello my friend"))

			// Print the message
			fmt.Printf("Received message: %s\n", string(msg.Data))

			// Return a result (for demonstration purposes, just print a result)
			result := processMessage(string(msg.Data))
			fmt.Printf("Processed result: %s\n", result)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the subscriber
	<-sigCh
	fmt.Println("Shutting down subscriber...")
}

func processMessage(msg string) string {
	// Simulate processing the message and returning a result
	return "Processed: " + msg
}
