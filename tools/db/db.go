package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewStore(
	dbname string,
	dbuser string,
	dbpassword string,
	dbhost string,
	dbport string,
) (*sql.DB, error) {
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
	return db, nil
}
