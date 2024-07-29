package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	mux.HandleFunc("/create_user", s.HandleCreateUser)
	mux.HandleFunc("/balance", s.HandleBalance)
	log.Fatal(http.ListenAndServe(s.listenAddr, mux))
}

func (s *APIServer) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("can't create user in db: %v", err)
		http.Error(w, "Can't create user", http.StatusInternalServerError)
		return
	}

	if err := s.sendUserCreatedEvent(user); err != nil {
		log.Printf("can't send event to kafka: %v", err)
		http.Error(w, "Can't finish creating user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request processed successfully\n"))
}

func (s *APIServer) sendUserCreatedEvent(user User) error {
	message, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshaling failed : %v", err)
	}
	msg := kafka.Message{Value: message}
	if err := s.kafka.WriteMessages(context.Background(), msg); err != nil {
		return fmt.Errorf("can't send to kafka: %v", err)
	}
	return nil
}

func (s *APIServer) HandleBalance(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var user *User
	if err = json.Unmarshal(body, &user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	user, err = s.store.GetUserByEmail(user.Email)
	if err != nil {
		log.Printf("err while getting user: %v", err)
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	resp, err := s.nastConn.Request("balance", []byte(user.UserID.String()), 2*time.Second)
	if err != nil {
		log.Printf("err publishing message to nats: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	respSring := string(resp.Data)
	if strings.HasPrefix(respSring, "error") {
		log.Printf("err on transaction service site: %s\n", respSring)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	OkResp := map[string]string{
		"email":   user.Email,
		"balance": respSring,
	}
	w.WriteHeader(http.StatusOK)
	data, err := json.Marshal(OkResp)
	if err != nil {
		log.Printf("err marshalling result: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
