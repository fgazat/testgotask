package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/segmentio/kafka-go"
)

type APIServer struct {
	listenAddr string
	store      Storage
	kafka      *kafka.Writer
	nastConn   *nats.Conn
}

func NewAPIServer(listenAddr string, store Storage, kafkaWriter *kafka.Writer) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
		kafka:      kafkaWriter,
	}
}

func (s *APIServer) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/add_money", s.HandleAddMoney)
	mux.HandleFunc("/transfer_money", s.HandleTransferMoney)
	log.Fatal(http.ListenAndServe(s.listenAddr, mux))
}

func (s *APIServer) HandleAddMoney(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	user.CreatedAt = time.Now()
	user.UserID = uuid.New()

	if err := s.store.CreateUser(&user); err != nil {
		http.Error(w, "Can't create user", http.StatusInternalServerError)
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

			userId, err := uuid.FromBytes(msg.Data)
			if err != nil {
				log.Printf("can't get uuid: %v", err)
				sendErr(msg, err)
				continue
			}

			user, err := s.store.GetUser(userId)
			if err != nil {
				log.Printf("can't get user: %v", err)
				sendErr(msg, err)
				continue
			}
			buf := new(bytes.Buffer)
			if err = binary.Write(buf, binary.LittleEndian, user.Balance); err != nil {
				log.Printf("can't convert to binary: %v", err)
				sendErr(msg, err)
				continue
			}
			if err = msg.Respond(buf.Bytes()); err != nil {
				log.Printf("err sending responce: %v", err)
				sendErr(msg, err)
				continue
			}
		}
	}()

	<-sigCh
	log.Println("shutting down")
}

func sendErr(msg *nats.Msg, err error) {
	msg.Respond([]byte("error: " + err.Error()))
}

// Kafka
func (s *APIServer) ListendCreateUserRequests() {
	// Publish messages to the "updates" subject every 3 seconds
}
