package main

import (
	"log"
	"os"

	"github.com/segmentio/kafka-go"
)

func main() {
	log.Println("hello")
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	kafkaAddr := os.Getenv("KAFKA_ADDR")
	serverAddr := os.Getenv("LISTEN_ADDR")

	db, err := NewStore(dbName, dbUser, dbPass, dbHost, dbPort)
	if err != nil {
		log.Fatal(err)
	}
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(kafkaAddr),
		Topic:    "user-created",
		Balancer: &kafka.LeastBytes{},
	}
	log.Printf("starting server: %s", serverAddr)
	server := NewAPIServer(serverAddr, db, kafkaWriter)
	server.Run()
}
