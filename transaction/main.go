package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/segmentio/kafka-go"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("hello")
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	kafkaAddr := os.Getenv("KAFKA_ADDR")
	natsAddr := os.Getenv("NATS_ADDR")
	serverAddr := os.Getenv("LISTEN_ADDR")

	db, err := newDB(dbName, dbUser, dbPass, dbHost, dbPort)
	if err != nil {
		log.Fatal(err)
	}

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaAddr}, // Kafka brokers
		Topic:   "user-created",      // Kafka topic
		// Logger:  log.New(log.Writer(), "kafka-reader: ", log.LstdFlags),
	})
	defer kafkaReader.Close()

	nc, err := nats.Connect(natsAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	log.Printf("starting server: %s", serverAddr)
	server := NewAPIServer(serverAddr, db, kafkaReader, nc)
	server.Run()
}

func newDB(
	dbname string,
	dbuser string,
	dbpassword string,
	dbhost string,
	dbport string,
) (*sql.DB, error) {
	connString := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		dbuser,
		dbpassword,
		dbname,
		dbhost,
		dbport,
	)
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
