package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateUser(user *User) error
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
	log.Println(connString)
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
