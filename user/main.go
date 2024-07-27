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
	db, err := NewStore(dbName, dbUser, dbPass, dbHost, dbPort)
	if err != nil {
		log.Fatal(err)
	}
	kafkaAddr := os.Getenv("kAFKA_ADDR")
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(kafkaAddr),
		Topic:    "user-created",
		Balancer: &kafka.LeastBytes{},
	}
	server := NewAPIServer(os.Getenv("LISTEN_ADDR"), db, kafkaWriter)
	log.Println("starting server")
	server.Run()
}
