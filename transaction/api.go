package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/segmentio/kafka-go"
)

type APIServer struct {
	listenAddr string
	db         *sql.DB
	kafka      *kafka.Reader
	nastConn   *nats.Conn
}

func NewAPIServer(
	listenAddr string,
	db *sql.DB,
	kafkaWriter *kafka.Reader,
	natsConn *nats.Conn,
) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		db:         db,
		kafka:      kafkaWriter,
		nastConn:   natsConn,
	}
}

func (s *APIServer) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/add_money", s.HandleAddMoney)
	mux.HandleFunc("/transfer_money", s.HandleTransferMoney)
	go s.ListendBalanceRequest()
	go s.ListendCreateUserRequests()
	log.Fatal(http.ListenAndServe(s.listenAddr, mux))
}

func (s *APIServer) HandleAddMoney(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request processed successfully\n"))
}

func (s *APIServer) HandleTransferMoney(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var user *User
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// get from kafka
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request processed successfully\n"))
}

// nats
func (s *APIServer) ListendBalanceRequest() {
	sub, err := s.nastConn.SubscribeSync("balance")
	if err != nil {
		fmt.Printf("error subscribing to subject: %v\n", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sendErr := func(msg *nats.Msg, err error) {
		log.Printf("err while processing nats event: %v", err)
		msg.Respond([]byte("error: " + err.Error()))
	}

	go func() {
		for {
			msg, err := sub.NextMsg(nats.DefaultTimeout)
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				log.Printf("error receiving message: %v\n", err)
				continue
			}
			userId, err := uuid.Parse(string(msg.Data))
			if err != nil {
				sendErr(msg, fmt.Errorf("can't get uuid: %v", err))
				continue
			}
			var user User
			err = s.db.QueryRow(`SELECT user_id, balance, created_at FROM "user" WHERE user_id = $1;`, userId).Scan(
				&user.UserID, &user.Balance, &user.CreatedAt,
			)
			if err != nil {
				sendErr(msg, fmt.Errorf("can't get user: %v", err))
				continue
			}
			str := strconv.Itoa(int(user.Balance))
			if err = msg.Respond([]byte(str)); err != nil {
				sendErr(msg, fmt.Errorf("err sending responce: %v", err))
				continue
			}
		}
	}()

	<-sigCh
	log.Println("shutting down")
}

// Kafka
func (s *APIServer) ListendCreateUserRequests() {
	for {
		msg, err := s.kafka.ReadMessage(context.Background())
		if err != nil {
			log.Printf("error reading message: %v", err)
			continue
		}
		var user User
		if err = json.Unmarshal(msg.Value, &user); err != nil {
			log.Printf("error unmarshalling: %v", err)
			continue
		}
		_, err = s.db.Exec(
			`INSERT INTO "user" (
            user_id, email, created_at
        ) VALUES (
            $1, $2, $3
        );`, user.UserID, user.Balance, user.CreatedAt)
		if err != nil {
			log.Printf("err creating user: %v", err)
			continue
		}
		fmt.Printf("user %s created", user.UserID.String())
	}
}
