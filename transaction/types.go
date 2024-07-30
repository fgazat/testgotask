package main

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UserID    uuid.UUID `json:"user_id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type Transaction struct {
	UserID uuid.UUID `json:"user_id"`
	Amount int64     `json:"amount"`
}
