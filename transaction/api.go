package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Supported method: POST\n"))
		return
	}
	var transaction Transaction
	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	idempotencyKey := uuid.New()
	updated, err := addBalance(s.db, transaction.UserID, transaction.Amount, idempotencyKey)
	if err != nil {
		log.Printf("err while trying to add money: %v", err)
		http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
		return

	}
	result := map[string]int64{
		"updated_balance": updated,
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func addBalance(db *sql.DB, userID uuid.UUID, amount int64, idempotencyKey uuid.UUID) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var processed bool
	err = tx.QueryRow("SELECT processed FROM transaction WHERE idempotency_key = $1;", idempotencyKey).Scan(&processed)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if processed {
		return 0, fmt.Errorf("operation is already processed")
	}

	var balance int64
	err = tx.QueryRow(`SELECT balance FROM "user" WHERE user_id = $1 FOR UPDATE;`, userID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	balance += amount
	_, err = tx.Exec(`UPDATE "user" SET balance = $1 WHERE user_id = $2;`, balance, userID)
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(`INSERT INTO transaction (user_id, amount, idempotency_key, processed, event_timestamp) 
VALUES ($1, $2, $3, $4, $5);`, userID, amount, idempotencyKey, true, time.Now())
	if err != nil {
		return 0, err
	}
	return balance, tx.Commit()
}

// { from_user_id: , to_user_id, amount_to_transfer
func (s *APIServer) HandleTransferMoney(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Supported method: POST\n"))
		return
	}
	var input struct {
		FromUserID uuid.UUID `json:"from_user_id"`
		ToUserID   uuid.UUID `json:"to_user_id"`
		Amount     int64     `json:"amount_to_transfer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if input.FromUserID == input.ToUserID || input.Amount == 0 {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	log.Println(input)

	idempotencyKey := uuid.New()
	if err := transferMoney(s.db, input.FromUserID, input.ToUserID, input.Amount, idempotencyKey); err != nil {
		http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// get from kafka
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request processed successfully\n"))
}

type Wallet struct {
	ID      int
	Balance int
	Version int
}

func transferMoney(db *sql.DB, fromID, toID uuid.UUID, amount int64, idempotencyKey uuid.UUID) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var processed bool
	err = tx.QueryRow("SELECT processed FROM transaction WHERE idempotency_key = $1;", idempotencyKey).Scan(&processed)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if processed {
		return fmt.Errorf("operation is already processed")
	}

	var fromUserBalance, toUserBalance int64
	// Lock rows
	err = tx.QueryRow(`SELECT balance FROM "user" WHERE user_id = $1 FOR UPDATE;`, fromID).Scan(&fromUserBalance)
	if err != nil {
		return err
	}
	err = tx.QueryRow(`SELECT balance FROM "user" WHERE user_id = $1 FOR UPDATE;`, toID).Scan(&toUserBalance)
	if err != nil {
		return err
	}

	if fromUserBalance < amount {
		return fmt.Errorf("insufficient funds")
	}
	fromUserBalance -= amount
	toUserBalance += amount

	_, err = tx.Exec(`UPDATE "user" SET balance = $1 WHERE user_id = $2;`, fromUserBalance, fromID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE "user" SET balance = $1 WHERE user_id = $2;`, toUserBalance, toID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO transaction (
    user_id, amount, idempotency_key, processed, event_timestamp
) VALUES (
    $1, $2, $3, $4, $5
), (
    $6, $7, $8, $4, $5
) ;`,
		fromID,
		-amount,
		idempotencyKey,
		true,
		time.Now(),
		toID,
		amount,
		// FIXME — another table for transactions maybe?
		uuid.New(),
	)
	if err != nil {
		return err
	}
	return tx.Commit()
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
            user_id, balance, created_at
        ) VALUES (
            $1, $2, $3
        );`, user.UserID, user.Balance, user.CreatedAt)
		if err != nil {
			log.Printf("err creating user: %v", err)
			continue
		}
		log.Printf("user %s created", user.UserID.String())
	}
}
