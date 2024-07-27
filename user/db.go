package main

import (
	"database/sql"

	"github.com/fgazat/testgotask/db"
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
	pgDB, err := db.New(dbname, dbuser, dbpassword, dbhost, dbport)
	if err != nil {
		return nil, err
	}
	return &PostgresStore{DB: pgDB}, nil
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
