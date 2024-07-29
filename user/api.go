package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	db         *sql.DB
	kafka      *kafka.Writer
	nastConn   *nats.Conn
}

func NewAPIServer(
	listenAddr string,
	db *sql.DB,
	kafkaWriter *kafka.Writer,
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
	mux.HandleFunc("/create_user", s.HandleCreateUser)
	mux.HandleFunc("/balance", s.HandleBalance)
	log.Fatal(http.ListenAndServe(s.listenAddr, mux))
}

func (s *APIServer) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	user.CreatedAt = time.Now()
	user.UserID = uuid.New()
	if err := s.createUser(r.Context(), &user); err != nil {
		log.Printf("can't create user: %v", err)
		http.Error(w, "Can't create user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request processed successfully\n"))
}

func (s *APIServer) createUser(ctx context.Context, user *User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO "user" (
    user_id, email, created_at
) VALUES (
    $1, $2, $3
);`, user.UserID, user.Email, user.CreatedAt)
	if err != nil {
		return err
	}

	message, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshaling failed : %v", err)
	}
	msg := kafka.Message{Value: message}
	if err := s.kafka.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("can't send to kafka: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *APIServer) HandleBalance(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	err := s.db.QueryRow(`SELECT user_id, created_at 
FROM "user"
WHERE email = $1;`, user.Email).Scan(
		&user.UserID, &user.CreatedAt,
	)
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
	jsonData, err := json.Marshal(OkResp)
	if err != nil {
		log.Printf("err marshalling result: %v\n", err)
		http.Error(w, "Failed to encode json", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
