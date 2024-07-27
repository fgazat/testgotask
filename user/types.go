package main

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UserID    uuid.UUID `json:",omitempty"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:",omitempty"`
}
