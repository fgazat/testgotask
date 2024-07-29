package main

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Storage interface {
	CreateUser(user *User) error
	GetUser(uuid uuid.UUID) (*User, error)
}

type PostgresStore struct {
	DB *sql.DB
}

func NewStore(
	dbname string,
	dbuser string,
	dbpassword string,
	dbhost string,
	dbport string,
) (*PostgresStore, error) {
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
	return &PostgresStore{DB: db}, nil
}

func (p *PostgresStore) CreateUser(user *User) error {
	_, err := p.DB.Exec(
		`INSERT INTO "user" (
    user_id, email, created_at
) VALUES (
    $1, $2, $3
);`, user.UserID, user.Email, user.CreatedAt)
	return err
}

func (p *PostgresStore) GetUserByEmail(email string) (*User, error) {
	var user User
	err := p.DB.QueryRow(`SELECT user_id, email, created_at FROM "user" WHERE email = '$1';`, email).Scan(
		&user.UserID, &user.Email, &user.CreatedAt,
	)
	return &user, err
}
